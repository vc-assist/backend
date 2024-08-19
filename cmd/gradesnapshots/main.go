package main

import (
	"context"
	"net/http"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/gradesnapshots/v1/gradesnapshotsv1connect"
	"vcassist-backend/services/gradesnapshots"
	gradesnapshotsdb "vcassist-backend/services/gradesnapshots/db"

	"connectrpc.com/connect"
)

type Config struct {
	Database configlibsql.Struct `json:"database"`
}

func main() {
	ctx := serviceutil.SignalContext()

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}

	db, err := config.Database.OpenDB(gradesnapshotsdb.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open database", err)
	}

	t, err := telemetry.SetupFromEnv(ctx, "gradesnapshots")
	if err != nil {
		serviceutil.Fatal("failed to setup telemetry", err)
	}
	defer t.Shutdown(context.Background())
	telemetry.InstrumentPerfStats(ctx)

	otelIntercept := serviceutil.NewConnectOtelInterceptor()

	mux := http.NewServeMux()
	mux.Handle(gradesnapshotsv1connect.NewGradeSnapshotsServiceHandler(
		gradesnapshotsv1connect.NewInstrumentedGradeSnapshotsServiceClient(
			gradesnapshots.NewService(db),
		),
		connect.WithInterceptors(otelIntercept),
	))
	go serviceutil.StartHttpServer(8444, mux)

	<-ctx.Done()
}
