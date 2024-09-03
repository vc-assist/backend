package powerschool

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"vcassist-backend/lib/textutil"
	"vcassist-backend/lib/timezone"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"
)

var homeworkPassesKeywords = []string{
	"hwpass",
	"homeworkpass",
}
var periodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func toSisCourses(ctx context.Context, input []CourseData) []*sisv1.CourseData {
	courses := make([]*sisv1.CourseData, len(input))
	for i, course := range input {
		currentDay := ""
		matches := periodRegex.FindStringSubmatch(course.Period)
		if len(matches) < 3 {
			slog.WarnContext(
				ctx, "period regex",
				"period", course.Period,
				"matches", matches,
				"err", "not enough matches",
			)
		} else {
			currentDay = matches[2]
		}

		now := timezone.Now().Unix()
		var overallGrade int64 = -1
		for _, term := range course.Terms {
			start, err := DecodeTimestamp(term.Start)
			if err != nil {
				slog.WarnContext(ctx, "failed to parse term start time", "time", term.Start, "err", err)
				continue
			}
			end, err := DecodeTimestamp(term.End)
			if err != nil {
				slog.WarnContext(ctx, "failed to parse term end time", "time", term.End, "err", err)
				continue
			}

			if now >= start.Unix() && now < end.Unix() {
				overallGrade = int64(term.FinalGrade.Percent)
				break
			}
		}

		homeworkPasses := 0
		var assignments []*sisv1.AssignmentData
		for _, assign := range course.Assignments {
			if textutil.MatchName(assign.Title, homeworkPassesKeywords) && assign.PointsEarned != nil {
				homeworkPasses = int(*assign.PointsEarned)
				continue
			}

			dueDate, err := DecodeTimestamp(assign.DueDate)
			if err != nil {
				slog.WarnContext(
					ctx, "failed to parse assignment due date",
					"due_date", assign.DueDate,
					"err", err,
				)
			}

			assignments = append(assignments, &sisv1.AssignmentData{
				Title:          assign.Title,
				Category:       assign.Category,
				DueDate:        dueDate.Unix(),
				Description:    assign.Description,
				PointsEarned:   assign.PointsEarned,
				PointsPossible: assign.PointsPossible,
				IsMissing:      assign.AttributeMissing,
				IsLate:         assign.AttributeLate,
				IsCollected:    assign.AttributeCollected,
				IsExempt:       assign.AttributeExempt,
				IsIncomplete:   assign.AttributeIncomplete,
			})
		}

		courses[i] = &sisv1.CourseData{
			Guid:         course.Guid,
			Name:         course.Name,
			Room:         course.Room,
			Period:       course.Period,
			Teacher:      fmt.Sprintf("%s %s", course.TeacherFirstName, course.TeacherLastName),
			TeacherEmail: course.TeacherEmail,
			Assignments:  assignments,
			Meetings:     nil,

			DayName:        currentDay,
			OverallGrade:   float32(overallGrade),
			HomeworkPasses: int32(homeworkPasses),
		}
	}

	return courses
}

func patchSisCourseMeetings(out []*sisv1.CourseData, input []CourseMeeting) {
	if len(out) == 0 {
		return
	}

	for _, course := range out {
		for _, courseMeeting := range input {
			if course.GetGuid() != courseMeeting.CourseGuid {
				continue
			}

			start, err := DecodeTimestamp(courseMeeting.Start)
			if err != nil {
				slog.Warn(
					"failed to parse start date of course meeting",
					"date", courseMeeting.Start,
					"err", err,
				)
				continue
			}
			stop, err := DecodeTimestamp(courseMeeting.Stop)
			if err != nil {
				slog.Warn(
					"failed to parse stop date of course meeting",
					"date", courseMeeting.Stop,
					"err", err,
				)
				continue
			}

			course.Meetings = append(course.Meetings, &sisv1.Meeting{
				Start: start.Unix(),
				Stop:  stop.Unix(),
			})
			break
		}
	}
}

func toSisSchools(input []SchoolData) []*sisv1.SchoolData {
	schools := make([]*sisv1.SchoolData, len(input))
	for i, school := range input {
		schools[i] = &sisv1.SchoolData{
			Name:          school.Name,
			Fax:           school.Fax,
			Phone:         school.Phone,
			Email:         school.Email,
			StreetAddress: school.StreetAddress,
			City:          school.City,
			State:         school.State,
			Zip:           school.Zip,
			Country:       school.Country,
		}
	}
	return schools
}

func toSisBulletins(input []Bulletin) []*sisv1.Bulletin {
	bulletins := make([]*sisv1.Bulletin, len(input))
	for i, bulletin := range input {
		start, err := DecodeTimestamp(bulletin.StartDate)
		if err != nil {
			slog.Warn(
				"failed to parse bulletin start time",
				"time", bulletin.StartDate,
				"err", err,
			)
			continue
		}
		stop, err := DecodeTimestamp(bulletin.EndDate)
		if err != nil {
			slog.Warn(
				"failed to parse bulletin end time",
				"time", bulletin.EndDate,
				"err", err,
			)
			continue
		}

		bulletins[i] = &sisv1.Bulletin{
			Title:     bulletin.Title,
			Body:      bulletin.Body,
			StartDate: start.Unix(),
			EndDate:   stop.Unix(),
		}
	}
	return bulletins
}

func ToSISData(
	ctx context.Context,
	profile StudentProfile,
	data *GetStudentDataResponse,
	courseMeetings []CourseMeeting,
) *sisv1.Data {
	gpa, err := strconv.ParseFloat(profile.CurrentGpa, 32)
	if err != nil {
		slog.WarnContext(ctx, "parse gpa", "gpa", profile.CurrentGpa, "err", err)
	}

	if len(data.Student.Courses) == 0 {
		slog.WarnContext(ctx, "student data unavailable, only returning profile...")
		return &sisv1.Data{
			Profile: &sisv1.StudentProfile{
				Guid:       profile.Guid,
				CurrentGpa: float32(gpa),
				Name:       fmt.Sprintf("%s %s", profile.FirstName, profile.LastName),
				// photo is disabled for now as it doesn't have a use
				// Photo: "",
			},
		}
	}

	courses := toSisCourses(ctx, data.Student.Courses)
	patchSisCourseMeetings(courses, courseMeetings)
	schools := toSisSchools(profile.Schools)
	bulletins := toSisBulletins(profile.Bulletins)

	return &sisv1.Data{
		Profile: &sisv1.StudentProfile{
			Guid:       profile.Guid,
			CurrentGpa: float32(gpa),
			Name:       fmt.Sprintf("%s %s", profile.FirstName, profile.LastName),
		},
		Schools:   schools,
		Bulletins: bulletins,
		Courses:   courses,
	}
}
