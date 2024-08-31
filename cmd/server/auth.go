package main

import (
	"net/http"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"
	"vcassist-backend/services/auth"
	"vcassist-backend/services/vcs/db"
)

type AuthSmtpConfig struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"`
}

type AuthConfig struct {
	Smtp           AuthSmtpConfig      `json:"smtp"`
	Database       configlibsql.Struct `json:"database"`
	AllowedDomains []string            `json:"allowed_domains"`
	// this is an email address that will have a verification code bypass
	// for app reviewers and testers
	TestEmail            string `json:"test_email"`
	TestVerificationCode string `json:"test_verification_code"`
}

func InitAuth(mux *http.ServeMux, cfg AuthConfig) error {
	db, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		return err
	}

	service := auth.NewService(db, auth.Options{
		AllowedDomains:       cfg.AllowedDomains,
		Smtp:                 auth.SmtpConfig(cfg.Smtp),
		TestEmail:            cfg.TestEmail,
		TestVerificationCode: cfg.TestVerificationCode,
	})
	mux.Handle(authv1connect.NewAuthServiceHandler(
		authv1connect.NewInstrumentedAuthServiceClient(
			service,
		),
	))
	return nil
}
