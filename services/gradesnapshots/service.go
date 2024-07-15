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

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "user",
			Value: attribute.IntValue(len(req.Msg.GetSnapshot().GetUser())),
		},
	)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	now := timezone.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timezone.Location).Unix()

	err = txqry.DeleteGradeSnapshotsAfter(ctx, startOfToday)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	user := req.Msg.GetSnapshot()
	for _, course := range user.GetCourses() {
		userCourseId, err := txqry.CreateUserCourse(ctx, db.CreateUserCourseParams{
			User:   user.GetUser(),
			Course: course.GetCourse(),
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		err = txqry.CreateGradeSnapshot(ctx, db.CreateGradeSnapshotParams{
			Usercourseid: userCourseId,
			Time:         course.GetSnapshot().GetTime(),
			Value:        float64(course.GetSnapshot().GetTime()),
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

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "user",
			Value: attribute.StringValue(req.Msg.GetUser()),
		},
	)

	rows, err := s.qry.GetGradeSnapshots(ctx, req.Msg.GetUser())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var courses []*gradesnapshotsv1.CourseSnapshotList
	var lastCourse *gradesnapshotsv1.CourseSnapshotList

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
			courses = append(courses, lastCourse)
			lastCourse = &gradesnapshotsv1.CourseSnapshotList{
				Course: r.Course,
				Snapshots: []*gradesnapshotsv1.Snapshot{
					{
						Time:  r.Time,
						Value: float32(r.Value),
					},
				},
			}
			continue
		}
		lastCourse.Snapshots = append(lastCourse.Snapshots, &gradesnapshotsv1.Snapshot{
			Time:  r.Time,
			Value: float32(r.Value),
		})
	}

	return &connect.Response[gradesnapshotsv1.PullResponse]{
		Msg: &gradesnapshotsv1.PullResponse{
			Courses: courses,
		},
	}, nil
}
