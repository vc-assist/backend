package main

import (
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/sis/v1/sisv1connect"
	"vcassist-backend/services/vcsis"
)

type VCSisOAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type VCSisConfig struct {
	Database           configlibsql.Struct `json:"database"`
	PowerschoolBaseUrl string              `json:"powerschool_base_url"`
	PowerschoolOAuth   VCSisOAuthConfig    `json:"powerschool_oauth"`
}

func InitVCSis(
	mux *http.ServeMux,
	cfg VCSisConfig,
	keychain keychainv1connect.KeychainServiceClient,
) {
	sisv1connect.SIServiceTracer = telemetry.Tracer("vcsis")
	mux.Handle(sisv1connect.NewSIServiceHandler(
		sisv1connect.NewInstrumentedSIServiceClient(
			vcsis.NewService(
				vcsis.ServiceOptions{
					// TODO: fix this
					Database: nil,
					Keychain: keychain,
					BaseUrl:  cfg.PowerschoolBaseUrl,
					OAuth:    vcsis.OAuthConfig(cfg.PowerschoolOAuth),
				},
			),
		),
	))
}
