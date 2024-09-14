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
	go func() {
		<-ctx.Done()
		telemetry.Shutdown(context.Background())
	}()
	telemetry.InstrumentPerfStats(ctx)

	if !verbose {
		return
	}

	vcsis.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput(".dev/resty/vcsis"),
	)
	keychain.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput(".dev/resty/keychain"),
	)
	core.SetRestyInstrumentOutput(
		restyutil.NewFilesystemOutput(".dev/resty/moodle_core"),
	)
}
