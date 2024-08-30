package powerservice

import (
	"context"
	"fmt"
	"log/slog"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	"vcassist-backend/services/auth/verifier"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

const keychainNamespace = "powerservice"

type Service struct {
	baseUrl  string
	oauth    OAuthConfig
	keychain keychainv1connect.KeychainServiceClient
}

func NewService(
	keychain keychainv1connect.KeychainServiceClient,
	baseUrl string,
	oauth OAuthConfig,
) Service {
	if oauth.BaseLoginUrl == "" {
		panic("empty base login url")
	}
	return Service{
		baseUrl:  baseUrl,
		oauth:    oauth,
		keychain: keychain,
	}
}

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[powerservicev1.GetCredentialStatusRequest]) (*connect.Response[powerservicev1.GetCredentialStatusResponse], error) {
	span := trace.SpanFromContext(ctx)
	profile, _ := verifier.ProfileFromContext(ctx)

	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}
	if res.Msg.GetKey() == nil || res.Msg.GetKey().GetExpiresAt() < timezone.Now().Unix() {
		span.SetStatus(codes.Ok, "got expired token")
		return &connect.Response[powerservicev1.GetCredentialStatusResponse]{
			Msg: &powerservicev1.GetCredentialStatusResponse{
				Status: &keychainv1.CredentialStatus{
					Name:      "PowerSchool",
					Picture:   "",
					Provided:  false,
					LoginFlow: &keychainv1.CredentialStatus_Oauth{},
				},
			},
		}, nil
	}

	oauthFlow, err := s.oauth.GetOAuthFlow()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create oauth flow")
		return nil, err
	}

	return &connect.Response[powerservicev1.GetCredentialStatusResponse]{
		Msg: &powerservicev1.GetCredentialStatusResponse{
			Status: &keychainv1.CredentialStatus{
				Name:     "PowerSchool",
				Picture:  "",
				Provided: true,
				LoginFlow: &keychainv1.CredentialStatus_Oauth{
					Oauth: oauthFlow,
				},
			},
		},
	}, nil
}

func (s Service) ProvideCredential(ctx context.Context, req *connect.Request[powerservicev1.ProvideCredentialRequest]) (*connect.Response[powerservicev1.ProvideCredentialResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}
	token := req.Msg.GetToken().GetToken()
	expiresAt, err := client.LoginOAuth(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %s", err.Error())
	}

	_, err = s.keychain.SetOAuth(ctx, &connect.Request[keychainv1.SetOAuthRequest]{
		Msg: &keychainv1.SetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
			Key: &keychainv1.OAuthKey{
				Token:      token,
				RefreshUrl: s.oauth.RefreshUrl,
				ClientId:   s.oauth.ClientId,
				ExpiresAt:  expiresAt.Unix(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[powerservicev1.ProvideCredentialResponse]{
		Msg: &powerservicev1.ProvideCredentialResponse{},
	}, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[powerservicev1.GetStudentDataRequest]) (*connect.Response[powerservicev1.GetStudentDataResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}
	if res.Msg.GetKey() == nil {
		err := fmt.Errorf("no oauth credentials provided")
		slog.ErrorContext(ctx, err.Error())
		return nil, err
	}

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create powerschool client", "err", err)
		return nil, err
	}
	_, err = client.LoginOAuth(ctx, res.Msg.GetKey().GetToken())
	if err != nil {
		slog.ErrorContext(ctx, "failed to login to powerschool", "err", err)
		return nil, err
	}

	data, err := Scrape(ctx, client)
	if err != nil {
		err := fmt.Errorf("failed to scrape: %w", err)
		slog.ErrorContext(ctx, err.Error())
		return nil, err
	}

	return &connect.Response[powerservicev1.GetStudentDataResponse]{
		Msg: data,
	}, nil
}
