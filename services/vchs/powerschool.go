package vchs

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"vcassist-backend/lib/scrapers/powerschool"
	pspb "vcassist-backend/services/powerschool/api"
	"vcassist-backend/services/studentdata/api"
	"vcassist-backend/lib/timezone"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

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

	now := timezone.Now().Unix()
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

func (s Service) studentDataFromPS(ctx context.Context, userEmail string) (*api.StudentData, error) {
	ctx, span := tracer.Start(ctx, "studentDataFromPS")
	defer span.End()

	psres, err := s.powerschool.GetStudentData(ctx, &connect.Request[pspb.GetStudentDataRequest]{
		Msg: &pspb.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var courseListGuids []string
	var courseList []*api.Course
	var dayNames []string
	for _, c := range psres.Msg.GetCourseData() {
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

	gpa, err := strconv.ParseFloat(psres.Msg.GetProfile().CurrentGpa, 32)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &api.StudentData{
		Gpa:      float32(gpa),
		DayNames: dayNames,
		Courses:  courseList,
	}, nil
}
