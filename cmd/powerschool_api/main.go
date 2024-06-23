package powerschoolapi

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/cmd/powerschool_api/api/apiconnect"
	"vcassist-backend/lib/configuration"
	"vcassist-backend/lib/telemetry"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatalerr(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

func main() {
	slog.Info("loading config...")
	config, err := configuration.ReadConfig[Config]("config.json5")
	if err != nil {
		fatalerr("failed to load configuration", err)
	}

	slog.Info("setting up telemetry...")
	tel, err := telemetry.SetupFromEnv(context.Background(), "powerschool_api")
	if err != nil {
		fatalerr("failed to setup telemetry", err)
	}
	defer func() {
		err := tel.Shutdown(context.Background())
		if err != nil {
			slog.Error("failed to shutdown telemetry", "err", err.Error())
		}
	}()

	slog.Info("opening database...")
	sqlite, err := OpenDB(config.Database)
	if err != nil {
		fatalerr("failed to open libsql connector", err)
	}

	slog.Info("setting up oauth daemon...")
	oauthd, err := NewOAuthDaemon(sqlite, config.OAuth)
	if err != nil {
		fatalerr("failed to create oauth daemon", err)
	}
	oauthd.Start(context.Background())

	slog.Info("setting up grpc service handler...")
	service := NewPowerschoolService(sqlite, config)
	mux := http.NewServeMux()
	mux.Handle(apiconnect.NewPowerschoolServiceHandler(service))

	slog.Info("listening to gRPC...", "port", 9000)
	err = http.ListenAndServe(
		"127.0.0.1:9000",
		// for gRPC clients, it's convenient to support HTTP/2 without TLS. you can
		// avoid x/net/http2 by using http.ListenAndServeTLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		fatalerr("failed to listen to port 9000", err)
	}
}
