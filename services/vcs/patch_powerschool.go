package vcs

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"
	"vcassist-backend/services/vcsmoodle"

	"go.opentelemetry.io/otel/codes"
)

var homeworkPassesKeywords = []string{
	"hwpass",
	"homeworkpass",
}

func patchAssignmentWithPowerschool(ctx context.Context, out *studentdatav1.Assignment, assignment *powerschoolv1.AssignmentData) {
	ctx, span := tracer.Start(ctx, "patchAssignment:WithPowerschool")
	defer span.End()

	state := studentdatav1.AssignmentState_ASSIGNMENT_STATE_UNSPECIFIED
	switch {
	case assignment.GetAttributeLate():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_LATE
	case assignment.GetAttributeCollected():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_SUBMITTED
	case assignment.GetAttributeMissing():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_MISSING
	case assignment.GetAttributeIncomplete():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_INCOMPLETE
	case assignment.GetAttributeExempt():
		state = studentdatav1.AssignmentState_ASSIGNMENT_STATE_EXEMPT
	}

	duedate, err := powerschool.DecodeAssignmentTime(assignment.GetDueDate())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	out.Name = assignment.GetTitle()
	out.Description = assignment.GetDescription()
	out.Scored = float32(assignment.GetPointsEarned())
	out.Total = float32(assignment.GetPointsPossible())
	out.State = state
	out.Time = duedate.Unix()
	out.AssignmentTypeName = assignment.GetCategory()
}

var periodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func patchCourseWithPowerschool(ctx context.Context, out *studentdatav1.Course, course *powerschoolv1.CourseData) error {
	ctx, span := tracer.Start(ctx, "patchCourse:WithPowerschool")
	defer span.End()

	matches := periodRegex.FindStringSubmatch(course.GetPeriod())
	if len(matches) < 3 {
		err := fmt.Errorf("could not run regex on course period '%s'", course.GetPeriod())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	currentDay := matches[2]

	now := timezone.Now().Unix()
	var overallGrade int64 = -1
	for _, term := range course.GetTerms() {
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

	out.Name = course.GetName()
	out.Room = course.GetRoom()
	out.Teacher = fmt.Sprintf(
		"%s %s",
		course.GetTeacherFirstName(),
		course.GetTeacherLastName(),
	)
	out.TeacherEmail = course.GetTeacherEmail()
	out.DayName = currentDay
	out.OverallGrade = float32(overallGrade)

	var assignmentTypeNameList []string
	for _, psassign := range course.GetAssignments() {
		if vcsmoodle.MatchName(psassign.GetTitle(), homeworkPassesKeywords) {
			out.HomeworkPasses = psassign.GetPointsEarned()
			continue
		}

		assignment := &studentdatav1.Assignment{}
		patchAssignmentWithPowerschool(ctx, assignment, psassign)
		assignmentTypeName := assignment.GetAssignmentTypeName()
		if !slices.Contains(assignmentTypeNameList, assignmentTypeName) {
			assignmentTypeNameList = append(assignmentTypeNameList, assignmentTypeName)
		}
		out.Assignments = append(out.Assignments, assignment)
	}
	for _, typename := range assignmentTypeNameList {
		out.AssignmentTypes = append(out.AssignmentTypes, &studentdatav1.AssignmentType{
			Name:   typename,
			Weight: 0,
		})
	}

	return nil
}

func patchStudentDataWithPowerschool(ctx context.Context, out *studentdatav1.StudentData, psdata *powerservicev1.GetStudentDataResponse) error {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithPowerschool")
	defer span.End()

	var courseListGuids []string
	var courseList []*studentdatav1.Course
	var dayNames []string
	for _, pscourse := range psdata.GetCourseData() {
		var course *studentdatav1.Course
		for _, existing := range out.GetCourses() {
			if pscourse.GetName() == existing.GetName() {
				course = existing
				break
			}
		}
		if course == nil {
			course = &studentdatav1.Course{}
		}

		err := patchCourseWithPowerschool(ctx, course, pscourse)
		if err != nil {
			continue
		}

		courseList = append(courseList, course)
		courseListGuids = append(courseListGuids, pscourse.GetGuid())

		currentDay := course.GetDayName()
		if !slices.Contains(dayNames, currentDay) {
			dayNames = append(dayNames, currentDay)
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
	}

	out.Gpa = float32(gpa)
	out.DayNames = dayNames
	out.Courses = courseList
	out.CurrentDay = currentDay

	imageBuff := bytes.NewBufferString(psdata.GetPhoto().GetStudentPhoto().GetImage())
	decoder := base64.NewDecoder(base64.StdEncoding, imageBuff)
	decodedImage, err := io.ReadAll(decoder)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	out.Photo = decodedImage

	return nil
}
