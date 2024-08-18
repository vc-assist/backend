package main

import (
	"context"
	"net/http"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/vcsmoodle/v1/vcsmoodlev1connect"
	"vcassist-backend/services/vcsmoodle"

	"connectrpc.com/connect"
)

type Config struct {
	KeychainBaseUrl string `json:"keychain_service_baseurl"`
}

func main() {
	ctx := serviceutil.SignalContext()

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to load configuration", err)
	}

	otelIntercept := serviceutil.NewConnectOtelInterceptor()

	t, err := telemetry.SetupFromEnv(ctx, "vcsmoodle")
	if err != nil {
		serviceutil.Fatal("failed to setup telemetry", err)
	}
	defer t.Shutdown(context.Background())

	mux := http.NewServeMux()
	mux.Handle(vcsmoodlev1connect.NewMoodleServiceHandler(
		vcsmoodlev1connect.NewInstrumentedMoodleServiceClient(
			vcsmoodle.NewService(keychainv1connect.NewKeychainServiceClient(
				http.DefaultClient,
				config.KeychainBaseUrl,
				connect.WithInterceptors(otelIntercept),
			)),
		),
		connect.WithInterceptors(otelIntercept),
	))
	go serviceutil.StartHttpServer(9222, mux)

	<-ctx.Done()
}
