package powerservice

import (
	"context"
	"log/slog"
	"vcassist-backend/lib/scrapers/powerschool"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
)

func transformCourses(ctx context.Context, input []powerschool.CourseData) []*powerservicev1.CourseData {
	courses := make([]*powerservicev1.CourseData, len(input))
	for i, course := range input {
		terms := make([]*powerservicev1.TermData, len(course.Terms))
		for i, t := range course.Terms {
			start, err := powerschool.DecodeTimestamp(t.Start)
			if err != nil {
				slog.WarnContext(ctx, "failed to parse term start time", "time", t.Start, "err", err)
			}
			end, err := powerschool.DecodeTimestamp(t.End)
			if err != nil {
				slog.WarnContext(ctx, "failed to parse term end time", "time", t.End, "err", err)
			}

			terms[i] = &powerservicev1.TermData{
				Start: start.Unix(),
				End:   end.Unix(),
				FinalGrade: &powerservicev1.TermData_FinalGrade{
					InProgressStatus: t.FinalGrade.InProgressStatus,
					Percent:          int32(t.FinalGrade.Percent),
				},
			}
		}

		assignments := make([]*powerservicev1.AssignmentData, len(course.Assignments))
		for i, a := range course.Assignments {
			dueDate, err := powerschool.DecodeTimestamp(a.DueDate)
			if err != nil {
				slog.WarnContext(
					ctx, "failed to parse assignment due date",
					"due_date", a.DueDate,
					"err", err,
				)
			}

			assignments[i] = &powerservicev1.AssignmentData{
				Title:               a.Title,
				Category:            a.Category,
				DueDate:             dueDate.Unix(),
				Description:         a.Description,
				PointsEarned:        a.PointsEarned,
				PointsPossible:      a.PointsPossible,
				AttributeMissing:    a.AttributeMissing,
				AttributeLate:       a.AttributeLate,
				AttributeCollected:  a.AttributeCollected,
				AttributeExempt:     a.AttributeExempt,
				AttributeIncomplete: a.AttributeIncomplete,
			}
		}

		courses[i] = &powerservicev1.CourseData{
			Guid:             course.Guid,
			Name:             course.Name,
			Room:             course.Room,
			Period:           course.Period,
			TeacherFirstName: course.TeacherFirstName,
			TeacherLastName:  course.TeacherLastName,
			TeacherEmail:     course.TeacherEmail,
			Terms:            terms,
			Assignments:      assignments,
			Meetings:         nil,
		}
	}

	return courses
}

func transformCourseMeetings(out []*powerservicev1.CourseData, input []powerschool.CourseMeeting) {
	if len(out) == 0 {
		return
	}

	for _, course := range out {
		for _, courseMeeting := range input {
			if course.GetGuid() != courseMeeting.CourseGuid {
				continue
			}

			start, err := powerschool.DecodeTimestamp(courseMeeting.Start)
			if err != nil {
				slog.Warn(
					"failed to parse start date of course meeting",
					"date", courseMeeting.Start,
					"err", err,
				)
				continue
			}
			stop, err := powerschool.DecodeTimestamp(courseMeeting.Stop)
			if err != nil {
				slog.Warn(
					"failed to parse stop date of course meeting",
					"date", courseMeeting.Stop,
					"err", err,
				)
				continue
			}

			course.Meetings = append(course.Meetings, &powerservicev1.Meeting{
				Start: start.Unix(),
				Stop:  stop.Unix(),
			})
			break
		}
	}
}

func transformSchools(input []powerschool.SchoolData) []*powerservicev1.SchoolData {
	schools := make([]*powerservicev1.SchoolData, len(input))
	for i, school := range input {
		schools[i] = &powerservicev1.SchoolData{
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

func transformBulletins(input []powerschool.Bulletin) []*powerservicev1.Bulletin {
	bulletins := make([]*powerservicev1.Bulletin, len(input))
	for i, bulletin := range input {
		start, err := powerschool.DecodeTimestamp(bulletin.StartDate)
		if err != nil {
			slog.Warn(
				"failed to parse bulletin start time",
				"time", bulletin.StartDate,
				"err", err,
			)
			continue
		}
		stop, err := powerschool.DecodeTimestamp(bulletin.EndDate)
		if err != nil {
			slog.Warn(
				"failed to parse bulletin end time",
				"time", bulletin.EndDate,
				"err", err,
			)
			continue
		}

		bulletins[i] = &powerservicev1.Bulletin{
			Title:     bulletin.Title,
			Body:      bulletin.Body,
			StartDate: start.Unix(),
			EndDate:   stop.Unix(),
		}
	}
	return bulletins
}
