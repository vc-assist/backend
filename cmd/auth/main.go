package main

import (
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/lib/configuration"
	configlibsql "vcassist-backend/lib/configuration/libsql"
	"vcassist-backend/lib/osutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"
	"vcassist-backend/services/auth"
	"vcassist-backend/services/auth/db"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatalerr(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

type smtpConfig struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"`
}

type config struct {
	Smtp           smtpConfig          `json:"smtp"`
	Libsql         configlibsql.Struct `json:"database"`
	AllowedDomains []string            `json:"allowed_domains"`
	// this is an email address that will have a verification code bypass
	// for app reviewers and testers
	TestEmail            string `json:"test_email"`
	TestVerificationCode string `json:"test_verification_code"`
}

func main() {
	config, err := configuration.ReadConfig[config]("config.json5")
	if err != nil {
		fatalerr("failed to read config", err)
	}
	slog.Info("opening database...")
	sqlite, err := config.Libsql.OpenDB(db.Schema)
	if err != nil {
		fatalerr("failed to open libsql connector", err)
	}

	service := auth.NewService(sqlite, auth.Config{
		AllowedDomains:       config.AllowedDomains,
		Smtp:                 auth.SmtpConfig(config.Smtp),
		TestEmail:            config.TestEmail,
		TestVerificationCode: config.TestVerificationCode,
	})
	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewAuthServiceHandler(service))

	ctx := osutil.SignalContext()

	go func() {
		slog.Info("listening to gRPC...", "port", 8111)
		err = http.ListenAndServe(
			"0.0.0.0:8111",
			h2c.NewHandler(mux, &http2.Server{}),
		)
		if err != nil {
			fatalerr("failed to listen to port 8111", err)
		}
	}()

	telemetry.SetupFromEnv(ctx, "cmd/auth")

	<-ctx.Done()
}
