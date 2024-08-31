package main

import (
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/gradesnapshots/v1/gradesnapshotsv1connect"
	"vcassist-backend/services/gradesnapshots"
	"vcassist-backend/services/gradesnapshots/db"
)

type GradeSnapshotsConfig struct {
	Database configlibsql.Struct `json:"database"`
}

func InitGradeSnapshots(mux *http.ServeMux, cfg GradeSnapshotsConfig) error {
	database, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		return err
	}

	gradesnapshotsv1connect.GradeSnapshotsServiceTracer = telemetry.Tracer("gradesnapshots")
	mux.Handle(gradesnapshotsv1connect.NewGradeSnapshotsServiceHandler(
		gradesnapshotsv1connect.NewInstrumentedGradeSnapshotsServiceClient(
			gradesnapshots.NewService(database),
		),
	))

	return nil
}
