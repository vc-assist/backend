package gradesnapshots

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"
	"vcassist-backend/lib/timezone"
	gradesnapshotsv1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
	"vcassist-backend/services/gradesnapshots/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var tracer = otel.Tracer("vcassist.services.gradesnapshots")

type Service struct {
	db  *sql.DB
	qry *db.Queries
}

func NewService(database *sql.DB) Service {
	return Service{
		db:  database,
		qry: db.New(database),
	}
}

func (s Service) Push(ctx context.Context, req *connect.Request[gradesnapshotsv1.PushRequest]) (*connect.Response[gradesnapshotsv1.PushResponse], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	snapshotTime := time.Unix(req.Msg.GetTime(), 0)
	startOfToday := time.Date(snapshotTime.Year(), snapshotTime.Month(), snapshotTime.Day(), 0, 0, 0, 0, timezone.Location).Unix()
	startOfTommorow := time.Date(snapshotTime.Year(), snapshotTime.Month(), snapshotTime.Day()+1, 0, 0, 0, 0, timezone.Location).Unix()

	err = txqry.DeleteGradeSnapshotsIn(ctx, db.DeleteGradeSnapshotsInParams{
		After:  startOfToday,
		Before: startOfTommorow,
		User:   req.Msg.GetUser(),
	})
	if err != nil {
		return nil, err
	}

	for _, course := range req.Msg.GetCourses() {
		if course.GetCourse() == "" {
			continue
		}

		err := txqry.CreateUserCourse(ctx, db.CreateUserCourseParams{
			User:   req.Msg.GetUser(),
			Course: course.GetCourse(),
		})
		if err != nil {
			return nil, err
		}

		userCourseId, err := txqry.GetUserCourseId(ctx, db.GetUserCourseIdParams{
			User:   req.Msg.GetUser(),
			Course: course.GetCourse(),
		})
		if err != nil {
			return nil, err
		}

		err = txqry.CreateGradeSnapshot(ctx, db.CreateGradeSnapshotParams{
			UserCourseID: userCourseId,
			Time:         req.Msg.GetTime(),
			Value:        float64(course.GetValue()),
		})
		if err != nil {
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &connect.Response[gradesnapshotsv1.PushResponse]{Msg: &gradesnapshotsv1.PushResponse{}}, nil
}

func (s Service) Pull(ctx context.Context, req *connect.Request[gradesnapshotsv1.PullRequest]) (*connect.Response[gradesnapshotsv1.PullResponse], error) {
	rows, err := s.qry.GetGradeSnapshots(ctx, req.Msg.GetUser())
	if err != nil {
		return nil, err
	}

	var courses []*gradesnapshotsv1.PullResponse_Course
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

		snapshots := make([]*gradesnapshotsv1.PullResponse_Course_Snapshot, len(grades))
		for i, tuple := range grades {
			snapshots[i] = &gradesnapshotsv1.PullResponse_Course_Snapshot{
				Time:  int64(tuple[0]),
				Value: float32(tuple[1]),
			}
		}
		courses = append(courses, &gradesnapshotsv1.PullResponse_Course{
			Course:    r.Course,
			Snapshots: snapshots,
		})
	}

	return &connect.Response[gradesnapshotsv1.PullResponse]{
		Msg: &gradesnapshotsv1.PullResponse{
			Courses: courses,
		},
	}, nil
}
