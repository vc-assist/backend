package vcsis

import (
	"context"
	"log/slog"
	"time"
	"vcassist-backend/lib/gradestore"
	"vcassist-backend/lib/timezone"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"

	"google.golang.org/protobuf/proto"
)

func (s Service) takeGradeSnapshots(ctx context.Context) error {
	studentIds, err := s.qry.GetAllStudents(ctx)
	if err != nil {
		return err
	}

	var userSnapshots []gradestore.UserSnapshot

	// doing these in serial to conserve memory
	// not spam everything all at once and risk OOM again
	for _, studentId := range studentIds {
		row, err := s.qry.GetStudentData(ctx, studentId)
		if err != nil {
			slog.WarnContext(ctx, "get student data", "student", studentId, "err", err)
			continue
		}

		data := &sisv1.Data{}
		err = proto.Unmarshal(row.Data, data)
		if err != nil {
			slog.WarnContext(ctx, "unmarshal data", "student", studentId, "err", err)
			continue
		}

		courseSnapshots := make([]gradestore.CourseSnapshot, len(data.Courses))
		for i, course := range data.Courses {
			courseSnapshots[i] = gradestore.CourseSnapshot{
				Course: course.GetGuid(),
				Value:  float64(course.GetOverallGrade()),
			}
		}
		userSnapshots = append(userSnapshots, gradestore.UserSnapshot{
			User:    studentId,
			Courses: courseSnapshots,
		})
	}

	return s.gradestore.Push(ctx, gradestore.PushRequest{
		Time:  timezone.Now(),
		Users: userSnapshots,
	})
}

func (s Service) gradeSnapshotDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := timezone.Now()
			if !(now.Hour() == 10 || now.Hour() == 18) {
				continue
			}
			err := s.takeGradeSnapshots(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "take grade snapshot", "err", err)
			}
		}
	}
}

func (s Service) preloadAllStudentData(ctx context.Context) error {
	studentIds, err := s.qry.GetAllStudents(ctx)
	if err != nil {
		return err
	}

	// doing these in serial to conserve memory
	// not spam everything all at once and risk OOM again
	for _, id := range studentIds {
		data, err := s.scrape(ctx, id)
		if err != nil {
			slog.WarnContext(ctx, "scrape student", "student_id", id, "err", err)
			continue
		}
		err = s.cacheNewData(ctx, id, data)
		if err != nil {
			slog.WarnContext(ctx, "cache data", "student_id", id, "err", err)
			continue
		}
	}

	return nil
}

func (s Service) preloadStudentDataDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := timezone.Now()
			// try to avoid peak hours
			if !(now.Hour() == 4 || now.Hour() == 20) {
				continue
			}

			ctx, cancel := context.WithTimeout(ctx, time.Hour)
			err := s.preloadAllStudentData(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "preload all student data", "err", err)
			}
			cancel()
		}
	}
}
