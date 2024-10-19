package main

import (
	"net/http"
	"vcassist-backend/lib/util/sqliteutil"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"
	"vcassist-backend/services/auth"
	"vcassist-backend/services/auth/db"
	"vcassist-backend/services/auth/verifier"
)

type AuthSmtpConfig struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"`
}

type AuthConfig struct {
	Smtp           AuthSmtpConfig `json:"smtp"`
	Database       string         `json:"database"`
	AllowedDomains []string       `json:"allowed_domains"`
	// this is an email address that will have a verification code bypass
	// for app reviewers and testers
	TestEmail            string `json:"test_email"`
	TestVerificationCode string `json:"test_verification_code"`
}

func InitAuth(mux *http.ServeMux, cfg AuthConfig) (verifier.Verifier, error) {
	database, err := sqliteutil.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return verifier.Verifier{}, err
	}

	service := auth.NewService(database, auth.Options{
		AllowedDomains:       cfg.AllowedDomains,
		Smtp:                 auth.SmtpConfig(cfg.Smtp),
		TestEmail:            cfg.TestEmail,
		TestVerificationCode: cfg.TestVerificationCode,
	})

	mux.Handle(authv1connect.NewAuthServiceHandler(service))

	return verifier.NewVerifier(database), nil
}
