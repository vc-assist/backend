package main

import (
	"net/http"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/powerservice/v1/powerservicev1connect"
	"vcassist-backend/services/powerservice"
)

type PowerserviceOAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type PowerserviceConfig struct {
	BaseUrl string                  `json:"base_url"`
	OAuth   PowerserviceOAuthConfig `json:"oauth"`
}

func InitPowerservice(
	mux *http.ServeMux,
	cfg PowerserviceConfig,
	keychain keychainv1connect.KeychainServiceClient,
) {
	powerservicev1connect.PowerschoolServiceTracer = telemetry.Tracer("powerservice")
	mux.Handle(powerservicev1connect.NewPowerschoolServiceHandler(
		powerservicev1connect.NewInstrumentedPowerschoolServiceClient(
			powerservice.NewService(
				keychain,
				cfg.BaseUrl,
				powerservice.OAuthConfig(cfg.OAuth),
			),
		),
	))
}
