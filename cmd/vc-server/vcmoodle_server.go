package main

import (
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/vcmoodle/v1/vcmoodlev1connect"
	"vcassist-backend/services/vcmoodle/server"
)

type VCMoodleServerConfig struct {
	Database configlibsql.Struct `json:"database"`
}

func InitVCMoodleServer(
	mux *http.ServeMux,
	cfg VCMoodleServerConfig,
	keychain keychainv1connect.KeychainServiceClient,
) error {
	database, err := cfg.Database.OpenDB()
	if err != nil {
		return err
	}

	vcmoodlev1connect.MoodleServiceTracer = telemetry.Tracer("vcmoodle_server")
	mux.Handle(vcmoodlev1connect.NewMoodleServiceHandler(
		vcmoodlev1connect.NewInstrumentedMoodleServiceClient(
			server.NewService(keychain, database),
		),
	))

	return nil
}
