package main

import (
	"context"
	"net/http"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"
	"vcassist-backend/services/auth"
	"vcassist-backend/services/auth/db"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
)

type SmtpConfig struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"`
}

type Config struct {
	Smtp           SmtpConfig          `json:"smtp"`
	Database       configlibsql.Struct `json:"database"`
	AllowedDomains []string            `json:"allowed_domains"`
	// this is an email address that will have a verification code bypass
	// for app reviewers and testers
	TestEmail            string `json:"test_email"`
	TestVerificationCode string `json:"test_verification_code"`
}

func main() {
	ctx := serviceutil.SignalContext()

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}
	db, err := config.Database.OpenDB(db.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open database", err)
	}

	t, err := telemetry.SetupFromEnv(ctx, "auth")
	if err != nil {
		serviceutil.Fatal("failed to setup telemetry", err)
	}
	defer t.Shutdown(context.Background())
	telemetry.InstrumentPerfStats(ctx)

	otelIntercept, err := otelconnect.NewInterceptor(
		otelconnect.WithTrustRemote(),
		otelconnect.WithoutServerPeerAttributes(),
	)
	if err != nil {
		serviceutil.Fatal("failed to initialize otel interceptor", err)
	}

	service := auth.NewService(db, auth.Options{
		AllowedDomains:       config.AllowedDomains,
		Smtp:                 auth.SmtpConfig(config.Smtp),
		TestEmail:            config.TestEmail,
		TestVerificationCode: config.TestVerificationCode,
	})
	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewAuthServiceHandler(
		authv1connect.NewInstrumentedAuthServiceClient(
			service,
		),
		connect.WithInterceptors(otelIntercept),
	))

	go serviceutil.StartHttpServer(8111, mux)

	<-ctx.Done()
}
