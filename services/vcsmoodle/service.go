package vcsmoodle

import (
	"context"
	"database/sql"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
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

func (s Service) ProvideUsernamePassword(ctx context.Context, req *connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]) (*connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse], error) {
	ctx, span := tracer.Start(ctx, "ProvideUsernamePassword")
	defer span.End()

	err := s.qry.CreateStudent(ctx, db.CreateStudentParams{
		ID:       req.Msg.GetStudentId(),
		Username: req.Msg.GetUsername(),
		Password: req.Msg.GetPassword(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse]{Msg: &vcsmoodlev1.ProvideUsernamePasswordResponse{}}, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[vcsmoodlev1.GetStudentDataRequest]) (*connect.Response[vcsmoodlev1.GetStudentDataResponse], error) {
	ctx, span := tracer.Start(ctx, "GetStudentData")
	defer span.End()

	studentRow, err := s.qry.GetStudent(ctx, req.Msg.GetStudentId())
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
		ClientId: req.Msg.GetStudentId(),
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

	return &connect.Response[vcsmoodlev1.GetStudentDataResponse]{
		Msg: &vcsmoodlev1.GetStudentDataResponse{
			Courses: courses,
		},
	}, nil
}
