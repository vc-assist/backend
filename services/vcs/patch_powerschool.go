package vcs

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	pspb "vcassist-backend/services/powerschool/api"
	"vcassist-backend/services/studentdata/api"

	"go.opentelemetry.io/otel/codes"
)

func patchAssignmentWithPowerschool(ctx context.Context, assignment *api.Assignment, psassign *powerschool.AssignmentData) {
	ctx, span := tracer.Start(ctx, "patchAssignment:WithPowerschool")
	defer span.End()

	state := api.AssignmentState_UNSET
	switch {
	case psassign.GetAttributeLate():
		state = api.AssignmentState_LATE
	case psassign.GetAttributeCollected():
		state = api.AssignmentState_SUBMITTED
	case psassign.GetAttributeMissing():
		state = api.AssignmentState_MISSING
	case psassign.GetAttributeIncomplete():
		state = api.AssignmentState_INCOMPLETE
	case psassign.GetAttributeExempt():
		state = api.AssignmentState_EXEMPT
	}

	duedate, err := powerschool.DecodeAssignmentTime(psassign.GetDueDate())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	assignment.Name = psassign.GetTitle()
	assignment.Description = psassign.GetDescription()
	assignment.Scored = float32(psassign.GetPointsEarned())
	assignment.Total = float32(psassign.GetPointsPossible())
	assignment.State = state
	assignment.Time = duedate.Unix()
	assignment.AssignmentTypeName = psassign.GetCategory()
}

var periodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func patchCourseWithPowerschool(ctx context.Context, course *api.Course, pscourse *powerschool.CourseData) error {
	ctx, span := tracer.Start(ctx, "patchCourse:WithPowerschool")
	defer span.End()

	matches := periodRegex.FindStringSubmatch(pscourse.Period)
	if len(matches) < 2 {
		err := fmt.Errorf("could not run regex on course period '%s'", pscourse.Period)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	currentDay := matches[1]

	now := timezone.Now().Unix()
	var overallGrade int64 = -1
	for _, term := range pscourse.GetTerms() {
		start, err := powerschool.DecodeCourseTermTime(term.Start)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		end, err := powerschool.DecodeCourseTermTime(term.End)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		if now >= start.Unix() && now < end.Unix() {
			overallGrade = int64(term.FinalGrade.GetPercent())
			break
		}
	}

	course.Name = pscourse.GetName()
	course.Room = pscourse.GetRoom()
	course.Teacher = fmt.Sprintf(
		"%s %s",
		pscourse.GetTeacherFirstName(),
		pscourse.GetTeacherLastName(),
	)
	course.TeacherEmail = pscourse.GetTeacherEmail()
	course.DayName = currentDay
	course.OverallGrade = float32(overallGrade)

	var assignmentTypeNameList []string
	for _, psassign := range pscourse.Assignments {
		assignment := &api.Assignment{}
		patchAssignmentWithPowerschool(ctx, assignment, psassign)
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

	return nil
}

func patchStudentDataWithPowerschool(ctx context.Context, data *api.StudentData, psdata *pspb.GetStudentDataResponse) error {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithPowerschool")
	defer span.End()

	var courseListGuids []string
	var courseList []*api.Course
	var dayNames []string
	for _, pscourse := range psdata.GetCourseData() {
		var course *api.Course
		for _, c := range data.GetCourses() {
			if pscourse.Name == c.Name {
				course = c
				break
			}
		}
		if course == nil {
			course = &api.Course{}
		}

		err := patchCourseWithPowerschool(ctx, course, pscourse)
		if err == nil {
			continue
		}
		courseList = append(courseList, course)
		courseListGuids = append(courseListGuids, pscourse.Guid)

		currentDay := course.DayName
		for _, day := range dayNames {
			if day == currentDay {
				dayNames = append(dayNames, currentDay)
				break
			}
		}
	}

	var currentDay string
	for _, meeting := range psdata.GetMeetings().SectionMeetings {
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

		if currentDay == "" && start.Day() == timezone.Now().Day() {
			currentDay = course.DayName
		}

		course.Meetings = append(course.Meetings, &api.CourseMeeting{
			StartTime: start.Unix(),
			EndTime:   stop.Unix(),
		})
	}

	gpa, err := strconv.ParseFloat(psdata.GetProfile().CurrentGpa, 32)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	data.Gpa = float32(gpa)
	data.DayNames = dayNames
	data.Courses = courseList
	data.CurrentDay = currentDay
	return nil
}
