package vcsis

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"vcassist-backend/lib/scrapers/powerschool"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"
)

func Scrape(ctx context.Context, client *powerschool.Client) (*sisv1.Data, error) {
	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		return nil, err
	}
	if len(allStudents.Profiles) == 0 {
		return nil, fmt.Errorf(
			"could not find student profile, are your credentials expired?",
		)
	}

	psStudent := allStudents.Profiles[0]
	studentData, err := client.GetStudentData(ctx, scraper.GetStudentDataRequest{
		Guid: psStudent.Guid,
	})
	if err != nil {
		return nil, err
	}

	guids := make([]string, len(studentData.Student.Courses))
	for i, c := range studentData.Student.Courses {
		guids[i] = c.Guid
	}
	start, stop := timezone.GetCurrentWeek(timezone.Now())
	res, err := client.GetCourseMeetingList(ctx, scraper.GetCourseMeetingListRequest{
		CourseGuids: guids,
		Start:       start.Format(time.RFC3339),
		Stop:        stop.Format(time.RFC3339),
	})
	if err != nil {
		slog.WarnContext(
			ctx,
			"fetch course meetings",
			"err", err,
		)
	}

	// MAY BE USED LATER, DO NOT DELETE
	// studentPhoto, err := client.GetStudentPhoto(ctx, scraper.GetStudentPhotoRequest{
	// 	Guid: psStudent.Guid,
	// })
	// if err != nil {
	// 	span.RecordError(err)
	// 	span.SetStatus(codes.Error, "failed to get student photo")
	// }

	return powerschool.ToSISData(ctx, psStudent, studentData, res.Meetings), nil
}
