package main

import (
	"net/http"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/util/sqliteutil"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/vcmoodle/v1/vcmoodlev1connect"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcmoodle/db"
	"vcassist-backend/services/vcmoodle/server"

	"connectrpc.com/connect"
)

type VCMoodleServerConfig struct {
	Database string `json:"database"`
}

func InitVCMoodleServer(
	mux *http.ServeMux,
	verify verifier.Verifier,
	cfg VCMoodleServerConfig,
	keychain keychainv1connect.KeychainServiceClient,
) error {
	database, err := sqliteutil.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return err
	}

	vcmoodlev1connect.MoodleServiceTracer = telemetry.Tracer("vcmoodle_server")
	mux.Handle(vcmoodlev1connect.NewMoodleServiceHandler(
		vcmoodlev1connect.NewInstrumentedMoodleServiceClient(
			server.NewService(keychain, database),
		),
		connect.WithInterceptors(
			verifier.NewAuthInterceptor(verify),
		),
	))

	return nil
}
