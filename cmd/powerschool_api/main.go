package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/cmd/powerschool_api/api/apiconnect"
	"vcassist-backend/cmd/powerschool_api/db"
	"vcassist-backend/lib/telemetry"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	cfg := flag.String("config", "config.json5", "specify the path to a config file")
	flag.Parse()

	slog.Info("loading config...")
	config := MustLoadConfig(context.Background(), *cfg)

	slog.Info("setting up telemetry...")
	t, err := telemetry.Setup(context.Background(), config.Telemetry)
	if err != nil {
		slog.Error("failed to setup telemetry", "err", err.Error())
		os.Exit(1)
	}
	defer func() {
		err := t.Shutdown(context.Background())
		if err != nil {
			slog.Error("failed to shutdown telemetry", "err", err.Error())
		}
	}()

	slog.Info("setting up database...")

	database, err := OpenDB(config.Database)
	if err != nil {
		slog.Error("failed to open libsql connector", "err", err.Error())
		os.Exit(1)
	}
	qry := db.New(database)

	slog.Info("running db migrations...")

	slog.Info("setting up telemetry objects...")

	oauthdMeter := t.MeterProvider.Meter("oauthd")
	refreshCounter, err := oauthdMeter.Int64Counter("refresh_token")
	if err != nil {
		slog.Error("failed to create counter for refresh_token", "err", err.Error())
		os.Exit(1)
	}

	slog.Info("setting up oauth daemon...")

	oauthd := OAuthDaemon{
		qry:            qry,
		db:             database,
		config:         config.OAuth,
		tracer:         t.TracerProvider.Tracer("oauthd"),
		refreshCounter: refreshCounter,
	}
	oauthd.Start(context.Background())

	slog.Info("setting up grpc service handler...")

	service := PowerschoolService{
		baseUrl: config.BaseUrl,
		qry:     qry,
		db:      database,
		tracer:  t.TracerProvider.Tracer("service"),
		meter:   t.MeterProvider.Meter("service"),
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
