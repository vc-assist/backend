package main

import (
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/services/linker"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
)

type LinkerConfig struct {
	Database    configlibsql.Struct `json:"database"`
	AccessToken string              `json:"access_token"`
}

func InitLinker(mux *http.ServeMux, cfg LinkerConfig) error {
	db, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		return err
	}
	mux.Handle(linkerv1connect.NewLinkerServiceHandler(
		linkerv1connect.NewInstrumentedLinkerServiceClient(
			linker.NewService(db),
		),
		connect.WithInterceptors(
			serviceutil.VerifyAccessTokenInterceptor(cfg.AccessToken),
		),
	))
	return nil
}
