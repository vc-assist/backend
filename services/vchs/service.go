package vchs

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"
	"vcassist-backend/lib/scrapers/powerschool"
	authdb "vcassist-backend/services/auth/db"
	"vcassist-backend/services/auth/verifier"
	pspb "vcassist-backend/services/powerschool/api"
	psrpc "vcassist-backend/services/powerschool/api/apiconnect"
	"vcassist-backend/services/studentdata/api"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("services/vchs")

type Service struct {
	powerschool psrpc.PowerschoolServiceClient
}

func getProfile(ctx context.Context) authdb.User {
	span := trace.SpanFromContext(ctx)
	profile, _ := verifier.ProfileFromContext(ctx)
	span.SetAttributes(attribute.KeyValue{
		Key:   "student_email",
		Value: attribute.StringValue(profile.Email),
	})
	return profile
}

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[api.GetCredentialStatusRequest]) (*connect.Response[api.GetCredentialStatusResponse], error) {
	ctx, span := tracer.Start(ctx, "GetCredentialStatus")
	defer span.End()

	profile := getProfile(ctx)

	psoauthflow, err := s.powerschool.GetOAuthFlow(ctx, &connect.Request[pspb.GetOAuthFlowRequest]{Msg: &pspb.GetOAuthFlowRequest{}})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	psauthstatus, err := s.powerschool.GetAuthStatus(ctx, &connect.Request[pspb.GetAuthStatusRequest]{
		Msg: &pspb.GetAuthStatusRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.GetCredentialStatusResponse]{
		Msg: &api.GetCredentialStatusResponse{
			Statuses: []*api.CredentialStatus{
				{
					Id:   "powerschool",
					Name: "PowerSchool",
					LoginFlow: &api.CredentialStatus_Oauth{
						Oauth: psoauthflow.Msg.Flow,
					},
					Provided: psauthstatus.Msg.IsAuthenticated,
				},
			},
		},
	}, nil
}

func (s Service) ProvideCredential(ctx context.Context, req *connect.Request[api.ProvideCredentialRequest]) (*connect.Response[api.ProvideCredentialResponse], error) {
	ctx, span := tracer.Start(ctx, "ProvideCredential")
	defer span.End()

	profile := getProfile(ctx)

	switch req.Msg.Id {
	case "powerschool":
		_, err := s.powerschool.ProvideOAuth(ctx, &connect.Request[pspb.ProvideOAuthRequest]{
			Msg: &pspb.ProvideOAuthRequest{
				StudentId: profile.Email,
				Token:     req.Msg.GetOauthToken().GetToken(),
			},
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	case "moodle":
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("moodle login is not yet implemented"))
	}

	return &connect.Response[api.ProvideCredentialResponse]{Msg: &api.ProvideCredentialResponse{}}, nil
}

func assignmentFromPSAssignment(ctx context.Context, a *powerschool.AssignmentData) *api.Assignment {
	span := trace.SpanFromContext(ctx)

	state := api.AssignmentState_UNSET
	switch {
	case a.GetAttributeLate():
		state = api.AssignmentState_LATE
	case a.GetAttributeCollected():
		state = api.AssignmentState_SUBMITTED
	case a.GetAttributeMissing():
		state = api.AssignmentState_MISSING
	case a.GetAttributeIncomplete():
		state = api.AssignmentState_INCOMPLETE
	case a.GetAttributeExempt():
		state = api.AssignmentState_EXEMPT
	}

	duedate, err := powerschool.DecodeAssignmentTime(a.GetDueDate())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return &api.Assignment{
		Name:               a.GetTitle(),
		Description:        a.GetDescription(),
		Scored:             float32(a.GetPointsEarned()),
		Total:              float32(a.GetPointsPossible()),
		State:              state,
		Time:               duedate.Unix(),
		AssignmentTypeName: a.GetCategory(),
	}
}

var periodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func courseFromPSCourse(ctx context.Context, pscourse *powerschool.CourseData) *api.Course {
	span := trace.SpanFromContext(ctx)

	matches := periodRegex.FindStringSubmatch(pscourse.Period)
	if len(matches) < 2 {
		err := fmt.Errorf("could not run regex on course period '%s'", pscourse.Period)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil
	}
	currentDay := matches[1]

	now := time.Now().Unix()
	var overallGrade int64 = -1
	for _, term := range pscourse.GetTerms() {
		start, err := powerschool.DecodeCourseTermTime(term.Start)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil
		}
		end, err := powerschool.DecodeCourseTermTime(term.End)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil
		}

		if now >= start.Unix() && now < end.Unix() {
			overallGrade = int64(term.FinalGrade.GetPercent())
			break
		}
	}

	course := &api.Course{
		Name: pscourse.GetName(),
		Room: pscourse.GetRoom(),
		Teacher: fmt.Sprintf(
			"%s %s",
			pscourse.GetTeacherFirstName(),
			pscourse.GetTeacherLastName(),
		),
		TeacherEmail: pscourse.GetTeacherEmail(),
		DayName:      currentDay,
		// compute homework pass heuristic on frontend,
		// just reply truthfully with what you know on backend
		HomeworkPasses: -1,
		OverallGrade:   float32(overallGrade),
	}

	var assignmentTypeNameList []string
	for _, a := range pscourse.Assignments {
		assignment := assignmentFromPSAssignment(ctx, a)
		assignmentTypeName := assignment.GetAssignmentTypeName()
		if !slices.Contains(assignmentTypeNameList, assignmentTypeName) {
			assignmentTypeNameList = append(assignmentTypeNameList, assignmentTypeName)
		}
		course.Assignments = append(course.Assignments, assignment)
	}
	for _, typename := range assignmentTypeNameList {
		course.AssignmentTypes = append(course.AssignmentTypes, &api.AssignmentType{
			Name:   typename,
			Weight: 0,
		})
	}

	return course
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[api.GetStudentDataRequest]) (*connect.Response[api.GetStudentDataResponse], error) {
	ctx, span := tracer.Start(ctx, "GetStudentData")
	defer span.End()

	profile := getProfile(ctx)

	psres, err := s.powerschool.GetStudentData(ctx, &connect.Request[pspb.GetStudentDataRequest]{
		Msg: &pspb.GetStudentDataRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	gpa, err := strconv.ParseFloat(psres.Msg.GetProfile().CurrentGpa, 32)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	pscourseList := psres.Msg.GetCourseData()

	var courseListGuids []string
	var courseList []*api.Course
	var dayNames []string
	for _, c := range pscourseList {
		course := courseFromPSCourse(ctx, c)
		if course == nil {
			continue
		}
		courseList = append(courseList, course)
		courseListGuids = append(courseListGuids, c.Guid)

		currentDay := course.DayName
		for _, day := range dayNames {
			if day == currentDay {
				dayNames = append(dayNames, currentDay)
				break
			}
		}
	}

	for _, meeting := range psres.Msg.GetMeetings().SectionMeetings {
		var course *api.Course
		for i, guid := range courseListGuids {
			if guid == meeting.SectionGuid {
				course = courseList[i]
				break
			}
		}
		if course == nil {
			err := fmt.Errorf(
				"could not find corresponding course for SectionMeeting with guid '%s'",
				meeting.SectionGuid,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			continue
		}

		start, err := powerschool.DecodeSectionMeetingTimestamp(meeting.GetStart())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			continue
		}
		stop, err := powerschool.DecodeSectionMeetingTimestamp(meeting.GetStop())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			continue
		}

		course.Meetings = append(course.Meetings, &api.CourseMeeting{
			StartTime: start.Unix(),
			EndTime:   stop.Unix(),
		})
	}

	result := &api.StudentData{
		Gpa:      float32(gpa),
		DayNames: dayNames,
		Courses:  courseList,
	}
	return &connect.Response[api.GetStudentDataResponse]{
		Msg: &api.GetStudentDataResponse{
			Data: result,
		},
	}, nil
}
