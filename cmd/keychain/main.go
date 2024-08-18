package main

import (
	"context"
	"net/http"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/services/keychain"
	keychaindb "vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
)

type Config struct {
	Database configlibsql.Struct `json:"database"`
}

func main() {
	ctx := serviceutil.SignalContext()

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}

	db, err := config.Database.OpenDB(keychaindb.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open database", err)
	}

	t, err := telemetry.SetupFromEnv(ctx, "keychain")
	if err != nil {
		serviceutil.Fatal("failed to setup telemetry", err)
	}
	defer t.Shutdown(context.Background())

	otelIntercept := serviceutil.NewConnectOtelInterceptor()

	mux := http.NewServeMux()
	mux.Handle(keychainv1connect.NewKeychainServiceHandler(
		keychainv1connect.NewInstrumentedKeychainServiceClient(
			keychain.NewService(ctx, db),
		),
		connect.WithInterceptors(otelIntercept),
	))
	go serviceutil.StartHttpServer(8333, mux)

	<-ctx.Done()
}
