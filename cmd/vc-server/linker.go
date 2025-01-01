package main

import (
	"net/http"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/pkg/migrations"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/services/linker"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
)

type LinkerConfig struct {
	Database    string `json:"database"`
	AccessToken string `json:"access_token"`
}

func InitLinker(mux *http.ServeMux, cfg LinkerConfig) (linkerv1connect.InstrumentedLinkerServiceClient, error) {
	db, err := migrations.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return linkerv1connect.NewInstrumentedLinkerServiceClient(nil), err
	}
	linkerv1connect.LinkerServiceTracer = telemetry.Tracer("linker")

	service := linkerv1connect.NewInstrumentedLinkerServiceClient(
		linker.NewService(db),
	)
	mux.Handle(linkerv1connect.NewLinkerServiceHandler(
		service,
		connect.WithInterceptors(
			serviceutil.VerifyAccessTokenInterceptor(cfg.AccessToken),
		),
	))
	return service, nil
}
