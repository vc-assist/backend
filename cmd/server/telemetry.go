package main

import (
	"context"
	"log/slog"
	"os"
	"time"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/powerservice"

	"github.com/lmittmann/tint"
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
