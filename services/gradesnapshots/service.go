package gradesnapshots

import (
	"context"
	"database/sql"
	"time"
	"vcassist-backend/lib/timezone"
	gradesnapshotsv1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
	"vcassist-backend/services/gradesnapshots/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("services/gradesnapshots")

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
	ctx, span := tracer.Start(ctx, "Push")
	defer span.End()

	span.SetAttributes(attribute.String("user", req.Msg.GetUser()))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for _, course := range req.Msg.GetCourses() {
		err := txqry.CreateUserCourse(ctx, db.CreateUserCourseParams{
			User:   req.Msg.GetUser(),
			Course: course.GetCourse(),
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		userCourseId, err := txqry.GetUserCourseId(ctx, db.GetUserCourseIdParams{
			User:   req.Msg.GetUser(),
			Course: course.GetCourse(),
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		err = txqry.CreateGradeSnapshot(ctx, db.CreateGradeSnapshotParams{
			UserCourseID: userCourseId,
			Time:         req.Msg.GetTime(),
			Value:        float64(course.GetValue()),
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[gradesnapshotsv1.PushResponse]{Msg: &gradesnapshotsv1.PushResponse{}}, nil
}

func (s Service) Pull(ctx context.Context, req *connect.Request[gradesnapshotsv1.PullRequest]) (*connect.Response[gradesnapshotsv1.PullResponse], error) {
	ctx, span := tracer.Start(ctx, "Pull")
	defer span.End()

	span.SetAttributes(attribute.String("user", req.Msg.GetUser()))

	rows, err := s.qry.GetGradeSnapshots(ctx, req.Msg.GetUser())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var courses []*gradesnapshotsv1.PullResponse_Course
	var lastCourse *gradesnapshotsv1.PullResponse_Course

	for _, r := range rows {
		// this works because the rows are sorted by course name
		// so all the rows with the same course name will be next to each other
		//
		// ex.
		// course 1 | time 1 | 98
		// course 1 | time 2 | 97
		// course 1 | time 3 | 98
		// course 2 | time 1 | 70
		// course 2 | time 2 | 70
		// etc...
		if r.Course != lastCourse.GetCourse() {
			if lastCourse != nil {
				courses = append(courses, lastCourse)
			}
			lastCourse = &gradesnapshotsv1.PullResponse_Course{
				Course: r.Course,
			}
		}
		lastCourse.Snapshots = append(lastCourse.Snapshots, &gradesnapshotsv1.PullResponse_Course_Snapshot{
			Time:  r.Time,
			Value: float32(r.Value),
		})
	}
	if lastCourse != nil {
		courses = append(courses, lastCourse)
	}

	return &connect.Response[gradesnapshotsv1.PullResponse]{
		Msg: &gradesnapshotsv1.PullResponse{
			Courses: courses,
		},
	}, nil
}
