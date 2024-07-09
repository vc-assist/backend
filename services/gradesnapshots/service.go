package gradesnapshots

import (
	"context"
	"database/sql"
	"vcassist-backend/services/gradesnapshots/api"
	"vcassist-backend/services/gradesnapshots/api/apiconnect"
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

	apiconnect.UnimplementedGradeSnapshotsServiceHandler
}

func NewService(database *sql.DB) Service {
	return Service{
		db:  database,
		qry: db.New(database),
	}
}

func (s Service) Push(ctx context.Context, req *connect.Request[api.PushRequest]) (*connect.Response[api.PushResponse], error) {
	ctx, span := tracer.Start(ctx, "Push")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "number_users",
			Value: attribute.IntValue(len(req.Msg.Users)),
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
	for _, user := range req.Msg.GetUsers() {
		for _, course := range user.GetCourses() {
			userCourseId, err := txqry.CreateUserCourse(ctx, db.CreateUserCourseParams{
				User:   user.User,
				Course: course.Course,
			})
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			err = txqry.CreateGradeSnapshot(ctx, db.CreateGradeSnapshotParams{
				Usercourseid: userCourseId,
				Time:         course.Snapshot.Time,
				Value:        float64(course.Snapshot.Time),
			})
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.PushResponse]{Msg: &api.PushResponse{}}, nil
}

func (s Service) Pull(ctx context.Context, req *connect.Request[api.PullRequest]) (*connect.Response[api.PullResponse], error) {
	ctx, span := tracer.Start(ctx, "Pull")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "user",
			Value: attribute.StringValue(req.Msg.GetUser()),
		},
	)

	rows, err := s.qry.GetGradeSnapshots(ctx, req.Msg.User)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var courses []*api.CourseSnapshotList
	var lastCourse *api.CourseSnapshotList

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
			lastCourse = &api.CourseSnapshotList{
				Course: r.Course,
				Snapshots: []*api.Snapshot{
					{
						Time:  r.Time,
						Value: float32(r.Value),
					},
				},
			}
			continue
		}
		lastCourse.Snapshots = append(lastCourse.Snapshots, &api.Snapshot{
			Time:  r.Time,
			Value: float32(r.Value),
		})
	}

	return &connect.Response[api.PullResponse]{
		Msg: &api.PullResponse{
			Courses: courses,
		},
	}, nil
}
