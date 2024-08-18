package main

import (
	"context"
	"net/http"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/powerservice/v1/powerservicev1connect"
	"vcassist-backend/services/powerservice"
	powerservicedb "vcassist-backend/services/powerservice/db"
)

type OAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type Config struct {
	BaseUrl         string              `json:"base_url"`
	Database        configlibsql.Struct `json:"database"`
	OAuth           OAuthConfig         `json:"oauth"`
	KeychainBaseUrl string              `json:"keychain_service_baseurl"`
}

func main() {
	ctx := serviceutil.SignalContext()

	t, err := telemetry.SetupFromEnv(ctx, "powerservice")
	if err != nil {
		serviceutil.Fatal("failed to setup telemetry", err)
	}
	defer t.Shutdown(context.Background())

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}
	db, err := config.Database.OpenDB(powerservicedb.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open database", err)
	}

	mux := http.NewServeMux()
	mux.Handle(powerservicev1connect.NewPowerschoolServiceHandler(
		powerservicev1connect.NewInstrumentedPowerschoolServiceClient(
			powerservice.NewService(
				db,
				keychainv1connect.NewKeychainServiceClient(
					http.DefaultClient,
					config.KeychainBaseUrl,
				),
				config.BaseUrl,
				powerservice.OAuthConfig(config.OAuth),
			),
		),
	))
	go serviceutil.StartHttpServer(8555, mux)

	<-ctx.Done()
}
