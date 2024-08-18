package main

import (
	"context"
	"net/http"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/services/linker"
	linkerdb "vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
)

type Config struct {
	Database    configlibsql.Struct `json:"database"`
	AccessToken string              `json:"access_token"`
}

func main() {
	ctx := serviceutil.SignalContext()

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}

	db, err := config.Database.OpenDB(linkerdb.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open database", err)
	}

	t, err := telemetry.SetupFromEnv(ctx, "linker")
	if err != nil {
		serviceutil.Fatal("failed to setup telemetry", err)
	}
	defer t.Shutdown(context.Background())

	mux := http.NewServeMux()
	mux.Handle(linkerv1connect.NewLinkerServiceHandler(
		linkerv1connect.NewInstrumentedLinkerServiceClient(
			linker.NewService(db),
		),
		connect.WithInterceptors(serviceutil.VerifyAccessTokenInterceptor(config.AccessToken)),
	))
	go serviceutil.StartHttpServer(8222, mux)

	<-ctx.Done()
}
