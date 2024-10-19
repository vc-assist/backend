package main

import (
	"net/http"
	"vcassist-backend/lib/util/serviceutil"
	"vcassist-backend/lib/util/sqliteutil"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/services/linker"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
)

type LinkerConfig struct {
	Database    string `json:"database"`
	AccessToken string `json:"access_token"`
}

func InitLinker(mux *http.ServeMux, cfg LinkerConfig) (linker.Service, error) {
	db, err := sqliteutil.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return linker.Service{}, err
	}

	service := linker.NewService(db)
	mux.Handle(linkerv1connect.NewLinkerServiceHandler(
		service,
		connect.WithInterceptors(
			serviceutil.VerifyAccessTokenInterceptor(cfg.AccessToken),
		),
	))
	return service, nil
}
