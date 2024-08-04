package vcsmoodle

import (
	"context"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
	"vcassist-backend/proto/vcassist/services/vcsmoodle/v1/vcsmoodlev1connect"

	"connectrpc.com/connect"
	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("services/vcsmoodle")

const keychainNamespace = "vcsmoodle"

type Service struct {
	cache    *badger.DB
	keychain keychainv1connect.KeychainServiceClient
}

func NewService(cache *badger.DB, keychain keychainv1connect.KeychainServiceClient) vcsmoodlev1connect.MoodleServiceClient {
	s := Service{
		cache:    cache,
		keychain: keychain,
	}
	return vcsmoodlev1connect.NewInstrumentedMoodleServiceClient(s)
}

func (s Service) GetAuthStatus(ctx context.Context, req *connect.Request[vcsmoodlev1.GetAuthStatusRequest]) (*connect.Response[vcsmoodlev1.GetAuthStatusResponse], error) {
	existing, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        req.Msg.GetStudentId(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[vcsmoodlev1.GetAuthStatusResponse]{
		Msg: &vcsmoodlev1.GetAuthStatusResponse{
			Provided: existing.Msg.GetKey() != nil,
		},
	}, nil
}

func (s Service) ProvideUsernamePassword(ctx context.Context, req *connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]) (*connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse], error) {
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
		return nil, err
	}

	return &connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse]{Msg: &vcsmoodlev1.ProvideUsernamePasswordResponse{}}, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[vcsmoodlev1.GetStudentDataRequest]) (*connect.Response[vcsmoodlev1.GetStudentDataResponse], error) {
	res, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        req.Msg.GetStudentId(),
		},
	})
	if err != nil {
		return nil, err
	}

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: "https://learn.vcs.net",
	})
	if err != nil {
		return nil, err
	}
	err = coreClient.LoginUsernamePassword(ctx, res.Msg.GetKey().GetUsername(), res.Msg.GetKey().GetPassword())
	if err != nil {
		return nil, err
	}
	client, err := view.NewClient(ctx, coreClient, view.ClientOptions{
		ClientId: req.Msg.GetStudentId(),
		Cache:    s.cache,
	})
	if err != nil {
		return nil, err
	}

	courses, err := scrapeCourses(ctx, client)
	if err != nil {
		return nil, err
	}

	return &connect.Response[vcsmoodlev1.GetStudentDataResponse]{
		Msg: &vcsmoodlev1.GetStudentDataResponse{
			Courses: courses,
		},
	}, nil
}
