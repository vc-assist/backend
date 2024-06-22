package powerschoolapi

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/cmd/powerschool_api/api/apiconnect"
	"vcassist-backend/cmd/powerschool_api/db"
	"vcassist-backend/lib/configuration"
	"vcassist-backend/lib/telemetry"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	cfg := flag.String("config", "config.json5", "specify the path to a config file")
	flag.Parse()

	slog.Info("loading config...")

	config, err := configuration.ReadConfig[Config](*cfg)
	if err != nil {
		slog.Error("failed to load configuration", "err", err.Error())
		os.Exit(1)
	}

	slog.Info("setting up telemetry...")

	tel, err := telemetry.SetupFromEnv(context.Background(), "powerschool_api")
	if err != nil {
		slog.Error("failed to setup telemetry", "err", err.Error())
		os.Exit(1)
	}
	defer func() {
		err := tel.Shutdown(context.Background())
		if err != nil {
			slog.Error("failed to shutdown telemetry", "err", err.Error())
		}
	}()

	slog.Info("setting up database...")

	sqlite, err := OpenDB(config.Database)
	if err != nil {
		slog.Error("failed to open libsql connector", "err", err.Error())
		os.Exit(1)
	}
	qry := db.New(sqlite)

	slog.Info("setting up telemetry objects...")

	oauthdMeter := tel.MeterProvider.Meter("oauthd")
	refreshCounter, err := oauthdMeter.Int64Counter("refresh_token")
	if err != nil {
		slog.Error("failed to create counter for refresh_token", "err", err.Error())
		os.Exit(1)
	}

	slog.Info("setting up oauth daemon...")

	oauthd := OAuthDaemon{
		qry:            qry,
		db:             sqlite,
		config:         config.OAuth,
		tracer:         tel.TracerProvider.Tracer("oauthd"),
		refreshCounter: refreshCounter,
	}
	oauthd.Start(context.Background())

	slog.Info("setting up grpc service handler...")

	service := PowerschoolService{
		baseUrl: config.BaseUrl,
		qry:     qry,
		db:      sqlite,
		tracer:  tel.TracerProvider.Tracer("service"),
		meter:   tel.MeterProvider.Meter("service"),
	}

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
		slog.Error("failed to listen to port 9000", "err", err.Error())
		os.Exit(1)
	}
}
