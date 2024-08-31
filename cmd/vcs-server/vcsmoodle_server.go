package main

import (
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/vcsmoodle/v1/vcsmoodlev1connect"
	"vcassist-backend/services/vcsmoodle/db"
	"vcassist-backend/services/vcsmoodle/server"
)

type VcsmoodleServerConfig struct {
	Database configlibsql.Struct `json:"database"`
}

func InitVcsmoodleServer(
	mux *http.ServeMux,
	cfg VcsmoodleServerConfig,
	keychain keychainv1connect.KeychainServiceClient,
) error {
	database, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		return err
	}

	vcsmoodlev1connect.MoodleServiceTracer = telemetry.Tracer("vcsmoodle_server")
	mux.Handle(vcsmoodlev1connect.NewMoodleServiceHandler(
		vcsmoodlev1connect.NewInstrumentedMoodleServiceClient(
			server.NewService(keychain, database),
		),
	))

	return nil

}
