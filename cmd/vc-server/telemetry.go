package main

import (
	"context"
	"log/slog"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/vcsis"
)

func InitTelemetry(ctx context.Context, verbose bool) {
	telemetry.InitSlog(verbose)

	if verbose {
		slog.DebugContext(ctx, "verbose logging enabled")
	}

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
