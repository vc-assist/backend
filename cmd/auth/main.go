package main

import (
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/lib/configuration"
	configlibsql "vcassist-backend/lib/configuration/libsql"
	"vcassist-backend/lib/osutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"
	"vcassist-backend/services/auth"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatalerr(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

type Config struct {
	Email  auth.EmailConfig    `json:"email"`
	Libsql configlibsql.Struct `json:"database"`
}

func main() {
	config, err := configuration.ReadConfig[Config]("config.json5")
	if err != nil {
		fatalerr("failed to read config", err)
	}
	slog.Info("opening database...")
	sqlite, err := config.Libsql.OpenDB()
	if err != nil {
		fatalerr("failed to open libsql connector", err)
	}

	service := auth.NewService(sqlite, config.Email)
	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewAuthServiceHandler(service))

	ctx := osutil.SignalContext()

	go func() {
		slog.Info("listening to gRPC...", "port", 8111)
		err = http.ListenAndServe(
			"0.0.0.0:8111",
			h2c.NewHandler(mux, &http2.Server{}),
		)
		if err != nil {
			fatalerr("failed to listen to port 8111", err)
		}
	}()

	telemetry.SetupFromEnv(ctx, "cmd/auth")

	<-ctx.Done()
}
