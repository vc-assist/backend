package vcsmoodle

import (
	"context"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"

	"connectrpc.com/connect"
	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("services/vcsmoodle")

const keychainNamespace = "vcsmoodle"

type Service struct {
	cache    *badger.DB
	keychain keychainv1connect.KeychainServiceClient
}

func NewService(cache *badger.DB, keychain keychainv1connect.KeychainServiceClient) Service {
	return Service{
		cache:    cache,
		keychain: keychain,
	}
}

func (s Service) GetAuthStatus(ctx context.Context, req *connect.Request[vcsmoodlev1.GetAuthStatusRequest]) (*connect.Response[vcsmoodlev1.GetAuthStatusResponse], error) {
	ctx, span := tracer.Start(ctx, "GetAuthStatus")
	defer span.End()

	existing, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        req.Msg.GetStudentId(),
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[vcsmoodlev1.GetAuthStatusResponse]{
		Msg: &vcsmoodlev1.GetAuthStatusResponse{
			Provided: existing.Msg.GetKey() != nil,
		},
	}, nil
}

func (s Service) ProvideUsernamePassword(ctx context.Context, req *connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]) (*connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse], error) {
	ctx, span := tracer.Start(ctx, "ProvideUsernamePassword")
	defer span.End()

	_, err := s.keychain.SetUsernamePassword(ctx, &connect.Request[keychainv1.SetUsernamePasswordRequest]{
		Msg: &keychainv1.SetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        req.Msg.GetStudentId(),
			Key: &keychainv1.UsernamePasswordKey{
				Username: req.Msg.GetUsername(),
				Password: req.Msg.GetPassword(),
			},
		},
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

	res, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        req.Msg.GetStudentId(),
		},
	})
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
	err = coreClient.LoginUsernamePassword(ctx, res.Msg.GetKey().GetUsername(), res.Msg.GetKey().GetPassword())
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
