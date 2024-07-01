package main

import (
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/cmd/authd/api/apiconnect"
	"vcassist-backend/lib/configuration"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatalerr(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

func main() {
	config, err := configuration.ReadConfig[AuthConfig]("config.json5")
	if err != nil {
		fatalerr("failed to read config", err)
	}

	slog.Info("opening database...")
	sqlite, err := config.Libsql.OpenDB()
	if err != nil {
		fatalerr("failed to open libsql connector", err)
	}

	service := GrpcService{
		service: NewService(sqlite, config.Email),
	}
	mux := http.NewServeMux()
	mux.Handle(apiconnect.NewAuthServiceHandler(service))

	slog.Info("listening to gRPC...", "port", 8111)
	err = http.ListenAndServe(
		"127.0.0.1:8111",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		fatalerr("failed to listen to port 8111", err)
	}
}
