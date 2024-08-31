package main

import (
	"context"
	"log/slog"
	"os"
	"time"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/auth"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/powerservice"

	"github.com/lmittmann/tint"
	"go.opentelemetry.io/otel/sdk/trace"
)

func initSlog(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
		return
	}
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	}))
	slog.SetDefault(logger)
}

func InitTelemetry(ctx context.Context, verbose bool) {
	t, err := telemetry.SetupFromEnv(ctx, "server")
	if err != nil {
		serviceutil.Fatal("setup telemetry", err)
	}
	defer t.Shutdown(context.Background())
	telemetry.InstrumentPerfStats(ctx)

	initSlog(verbose)
	if !verbose {
		return
	}

	authResource, err := telemetry.NewResource("auth")
	if err != nil {
		serviceutil.Fatal("setup telemetry", err)
	}
	authTraceProvider := trace.NewTracerProvider(
		trace.WithBatcher(t.SpanExporter),
		trace.WithResource(authResource),
	)
	auth.SetTracerProvider(authTraceProvider)

	vcsmoodleResource, err := telemetry.NewResource("vcsmoodle_scraper")
	if err != nil {
		serviceutil.Fatal("setup telemetry", err)
	}
	vcsmoodleTraceProvider := trace.NewTracerProvider(
		trace.WithBatcher(t.SpanExporter),
		trace.WithResource(vcsmoodleResource),
	)
	core.SetTracerProvider(vcsmoodleTraceProvider)
	view.SetTracerProvider(vcsmoodleTraceProvider)

	powerservice.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput("<dev_state>/resty_telemetry/powerservice"),
	)
	keychain.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput("<dev_state>/resty_telemetry/keychain"),
	)
	core.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput("<dev_state>/resty_telemetry/moodle_core"),
	)
}
