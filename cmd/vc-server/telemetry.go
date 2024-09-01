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
	"vcassist-backend/services/vcsis"

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
	initSlog(verbose)

	err := telemetry.SetupFromEnv(ctx, "server")
	if err != nil {
		serviceutil.Fatal("setup telemetry", err)
	}
	defer telemetry.Shutdown(context.Background())
	telemetry.InstrumentPerfStats(ctx)

	if !verbose {
		return
	}

	vcsis.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput("<dev_state>/resty_telemetry/vcsis"),
	)
	keychain.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput("<dev_state>/resty_telemetry/keychain"),
	)
	core.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput("<dev_state>/resty_telemetry/moodle_core"),
	)
}
