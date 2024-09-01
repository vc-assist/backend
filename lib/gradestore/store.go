package gradestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"
	"vcassist-backend/lib/gradestore/db"
	"vcassist-backend/lib/timezone"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

type Store struct {
	db  *sql.DB
	qry *db.Queries
}

func NewStore(database *sql.DB) Store {
	return Store{
		db:  database,
		qry: db.New(database),
	}
}

type CourseSnapshot struct {
	Course string
	Value  float64
}

type UserSnapshot struct {
	User    string
	Courses []CourseSnapshot
}

type PushRequest struct {
	Time  time.Time
	Users []UserSnapshot
}

func (s Store) Push(ctx context.Context, req PushRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	startOfToday := time.Date(req.Time.Year(), req.Time.Month(), req.Time.Day(), 0, 0, 0, 0, timezone.Location).Unix()
	startOfTommorow := time.Date(req.Time.Year(), req.Time.Month(), req.Time.Day()+1, 0, 0, 0, 0, timezone.Location).Unix()

	users := make([]string, len(req.Users))
	for i, v := range req.Users {
		users[i] = v.User
	}
	err = txqry.DeleteGradeSnapshotsIn(ctx, db.DeleteGradeSnapshotsInParams{
		After:  startOfToday,
		Before: startOfTommorow,
		Users:  users,
	})
	if err != nil {
		return err
	}

	for _, user := range req.Users {
		for _, course := range user.Courses {
			err := txqry.CreateUserCourse(ctx, db.CreateUserCourseParams{
				User:   user.User,
				Course: course.Course,
			})
			if err != nil {
				return err
			}

			userCourseId, err := txqry.GetUserCourseId(ctx, db.GetUserCourseIdParams{
				User:   user.User,
				Course: course.Course,
			})
			if err != nil {
				return err
			}

			err = txqry.CreateGradeSnapshot(ctx, db.CreateGradeSnapshotParams{
				UserCourseID: userCourseId,
				Time:         req.Time.Unix(),
				Value:        course.Value,
			})
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

type GradeSnapshot struct {
	Time  time.Time
	Value float32
}

type CourseSnapshotSeries struct {
	Course    string
	Snapshots []GradeSnapshot
}

func (s Store) Pull(ctx context.Context, user string) ([]CourseSnapshotSeries, error) {
	rows, err := s.qry.GetGradeSnapshots(ctx, user)
	if err != nil {
		return nil, err
	}

	var courses []CourseSnapshotSeries
	for _, r := range rows {
		if r.Course == "" {
			continue
		}

		var grades db.GetGradeSnapshotsRowGrades
		err = json.Unmarshal([]byte(r.Grades.(string)), &grades)
		if err != nil {
			slog.WarnContext(ctx, "failed to unmarshal db grades", "err", err)
			continue
		}

		snapshots := make([]GradeSnapshot, len(grades))
		for i, tuple := range grades {
			snapshots[i] = GradeSnapshot{
				Time:  time.Unix(int64(tuple[0]), 0),
				Value: float32(tuple[1]),
			}
		}
		courses = append(courses, CourseSnapshotSeries{
			Course:    r.Course,
			Snapshots: snapshots,
		})
	}

	return courses, nil
}
