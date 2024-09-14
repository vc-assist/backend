package main

import (
	"context"
	"vcassist-backend/lib/sqliteutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/keychain/db"
)

type KeychainConfig struct {
	Database string `json:"database"`
}

func InitKeychain(ctx context.Context, cfg KeychainConfig) (keychainv1connect.InstrumentedKeychainServiceClient, error) {
	db, err := sqliteutil.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return keychainv1connect.NewInstrumentedKeychainServiceClient(nil), err
	}

	keychainv1connect.KeychainServiceTracer = telemetry.Tracer("keychain")
	service := keychain.NewService(ctx, db)
	instrumented := keychainv1connect.NewInstrumentedKeychainServiceClient(service)
	return instrumented, nil
}
