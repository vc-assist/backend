package vcsmoodle

import (
	"context"
	"database/sql"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/services/vcsmoodle/api"
	"vcassist-backend/services/vcsmoodle/db"

	"connectrpc.com/connect"
	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("services/vcsmoodle")

type Service struct {
	db    *sql.DB
	qry   *db.Queries
	cache *badger.DB
}

func NewService(database *sql.DB, cache *badger.DB) Service {
	return Service{
		db:    database,
		qry:   db.New(database),
		cache: cache,
	}
}

func (s Service) ProvideUsernamePassword(ctx context.Context, req *connect.Request[api.ProvideUsernamePasswordRequest]) (*connect.Response[api.ProvideUsernamePasswordResponse], error) {
	ctx, span := tracer.Start(ctx, "ProvideUsernamePassword")
	defer span.End()

	err := s.qry.CreateStudent(ctx, db.CreateStudentParams{
		ID:       req.Msg.StudentId,
		Username: req.Msg.Username,
		Password: req.Msg.Password,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.ProvideUsernamePasswordResponse]{Msg: &api.ProvideUsernamePasswordResponse{}}, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[api.GetStudentDataRequest]) (*connect.Response[api.GetStudentDataResponse], error) {
	ctx, span := tracer.Start(ctx, "GetStudentData")
	defer span.End()

	studentRow, err := s.qry.GetStudent(ctx, req.Msg.StudentId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: "https://learn.vcs.net",
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	err = coreClient.LoginUsernamePassword(ctx, studentRow.Username, studentRow.Password)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	client, err := view.NewClient(ctx, coreClient, view.ClientOptions{
		ClientId: req.Msg.StudentId,
		Cache:    s.cache,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	courses, err := scrapeCourses(ctx, client)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.GetStudentDataResponse]{
		Msg: &api.GetStudentDataResponse{
			Courses: courses,
		},
	}, nil
}
