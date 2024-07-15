package vcs

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"

	"go.opentelemetry.io/otel/codes"
)

func patchAssignmentWithPowerschool(ctx context.Context, assignment *studentdatav1.Assignment, psassign *powerschoolv1.AssignmentData) {
	ctx, span := tracer.Start(ctx, "patchAssignment:WithPowerschool")
	defer span.End()

	state := studentdatav1.AssignmentState_ASSIGNMENT_STATE_UNSPECIFIED
	switch {
	case psassign.GetAttributeLate():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_LATE
	case psassign.GetAttributeCollected():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_SUBMITTED
	case psassign.GetAttributeMissing():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_MISSING
	case psassign.GetAttributeIncomplete():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_INCOMPLETE
	case psassign.GetAttributeExempt():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_EXEMPT
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

func patchCourseWithPowerschool(ctx context.Context, course *studentdatav1.Course, pscourse *powerschoolv1.CourseData) error {
	ctx, span := tracer.Start(ctx, "patchCourse:WithPowerschool")
	defer span.End()

	matches := periodRegex.FindStringSubmatch(pscourse.GetPeriod())
	if len(matches) < 2 {
		err := fmt.Errorf("could not run regex on course period '%s'", pscourse.GetPeriod())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	currentDay := matches[1]

	now := timezone.Now().Unix()
	var overallGrade int64 = -1
	for _, term := range pscourse.GetTerms() {
		start, err := powerschool.DecodeCourseTermTime(term.GetStart())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		end, err := powerschool.DecodeCourseTermTime(term.GetEnd())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		if now >= start.Unix() && now < end.Unix() {
			overallGrade = int64(term.GetFinalGrade().GetPercent())
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
	for _, psassign := range pscourse.GetAssignments() {
		assignment := &studentdatav1.Assignment{}
		patchAssignmentWithPowerschool(ctx, assignment, psassign)
		assignmentTypeName := assignment.GetAssignmentTypeName()
		if !slices.Contains(assignmentTypeNameList, assignmentTypeName) {
			assignmentTypeNameList = append(assignmentTypeNameList, assignmentTypeName)
		}
		course.Assignments = append(course.Assignments, assignment)
	}
	for _, typename := range assignmentTypeNameList {
		course.AssignmentTypes = append(course.AssignmentTypes, &studentdatav1.AssignmentType{
			Name:   typename,
			Weight: 0,
		})
	}

	return nil
}

func patchStudentDataWithPowerschool(ctx context.Context, data *studentdatav1.StudentData, psdata *powerservicev1.GetStudentDataResponse) error {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithPowerschool")
	defer span.End()

	var courseListGuids []string
	var courseList []*studentdatav1.Course
	var dayNames []string
	for _, pscourse := range psdata.GetCourseData() {
		var course *studentdatav1.Course
		for _, c := range data.GetCourses() {
			if pscourse.GetName() == c.GetName() {
				course = c
				break
			}
		}
		if course == nil {
			course = &studentdatav1.Course{}
		}

		err := patchCourseWithPowerschool(ctx, course, pscourse)
		if err == nil {
			continue
		}
		courseList = append(courseList, course)
		courseListGuids = append(courseListGuids, pscourse.GetGuid())

		currentDay := course.GetDayName()
		for _, day := range dayNames {
			if day == currentDay {
				dayNames = append(dayNames, currentDay)
				break
			}
		}
	}

	var currentDay string
	for _, meeting := range psdata.GetMeetings().GetSectionMeetings() {
		var course *studentdatav1.Course
		for i, guid := range courseListGuids {
			if guid == meeting.GetSectionGuid() {
				course = courseList[i]
				break
			}
		}
		if course == nil {
			err := fmt.Errorf(
				"could not find corresponding course for SectionMeeting with guid '%s'",
				meeting.GetSectionGuid(),
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
			currentDay = course.GetDayName()
		}

		course.Meetings = append(course.Meetings, &studentdatav1.CourseMeeting{
			StartTime: start.Unix(),
			EndTime:   stop.Unix(),
		})
	}

	gpa, err := strconv.ParseFloat(psdata.GetProfile().GetCurrentGpa(), 32)
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
