package main

import (
	"context"
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/keychain/db"
)

type KeychainConfig struct {
	Database configlibsql.Struct `json:"database"`
}

func InitKeychain(ctx context.Context, mux *http.ServeMux, cfg KeychainConfig) (keychainv1connect.InstrumentedKeychainServiceClient, error) {
	db, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		return keychainv1connect.NewInstrumentedKeychainServiceClient(nil), err
	}

	keychainv1connect.KeychainServiceTracer = telemetry.Tracer("keychain")
	service := keychain.NewService(ctx, db)
	instrumented := keychainv1connect.NewInstrumentedKeychainServiceClient(service)
	mux.Handle(keychainv1connect.NewKeychainServiceHandler(
		instrumented,
	))

	return instrumented, nil
}
