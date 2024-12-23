package main

import (
	"net/http"
	"vcassist-backend/lib/sqliteutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/vcmoodle/v1/vcmoodlev1connect"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/vcmoodle/db"
	"vcassist-backend/services/vcmoodle/server"

	"connectrpc.com/connect"
)

type VCMoodleServerConfig struct {
	Database string `json:"database"`
}

func InitVCMoodleServer(
	mux *http.ServeMux,
	keychainIntercepter keychain.AuthInterceptor,
	cfg VCMoodleServerConfig,
	keych keychainv1connect.KeychainServiceClient,
) error {
	database, err := sqliteutil.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return err
	}

	vcmoodlev1connect.MoodleServiceTracer = telemetry.Tracer("vcmoodle_server")
	mux.Handle(vcmoodlev1connect.NewMoodleServiceHandler(
		server.NewService(keych, database),
		connect.WithInterceptors(
			keychain.NewAuthInterceptor(database),
		),
	))

	return nil
}
