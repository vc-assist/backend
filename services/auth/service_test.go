package auth

import (
	"context"
	"database/sql"
	"io"
	"log"
	"strings"
	"testing"
	"vcassist-backend/lib/telemetry"
	authv1 "vcassist-backend/proto/vcassist/services/auth/v1"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"

	"connectrpc.com/connect"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed db/schema.sql
var schemaSql string

func setup(t testing.TB) (authv1connect.AuthServiceClient, func()) {
	cleanup := telemetry.SetupForTesting("test:auth")
	sqlite, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.Exec(schemaSql)
	if err != nil {
		t.Fatal(err)
	}

	// suppress logging
	testcontainers.Logger = log.New(io.Discard, "", 0)

	smtp, err := testcontainers.GenericContainer(
		context.Background(),
		testcontainers.GenericContainerRequest{
			Started: true,
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "haravich/fake-smtp-server",
				ExposedPorts: []string{"1025:1025", "1090:1080"},
				WaitingFor:   wait.ForLog("smtp://0.0.0.0:1025"),
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(sqlite, Options{
		Smtp: SmtpConfig{
			Server:       "localhost",
			Port:         1025,
			EmailAddress: "alice@email.com",
			Password:     "default",
		},
	})

	return service, func() {
		cleanup()
		err := smtp.Terminate(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}
}

var globalClient = resty.New()

func getVerificationCodeFromEmail(t testing.TB) string {
	res, err := globalClient.R().
		Get("http://127.0.0.1:1090/messages/1.plain")
	if err != nil {
		t.Fatal(err)
	}
	contents := res.String()
	return strings.Split(contents, "\n\n")[1]
}

func TestLoginFlow(t *testing.T) {
	service, cleanup := setup(t)
	defer cleanup()

	tracer := telemetry.Tracer("vcassist.services.auth.service_test")
	ctx, span := tracer.Start(context.Background(), "TestLoginFlow")
	defer span.End()

	userEmail := "bob@email.com"

	_, err := service.StartLogin(ctx, &connect.Request[authv1.StartLoginRequest]{
		Msg: &authv1.StartLoginRequest{
			Email: userEmail,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	code := getVerificationCodeFromEmail(t)

	res, err := service.ConsumeVerificationCode(ctx, &connect.Request[authv1.ConsumeVerificationCodeRequest]{
		Msg: &authv1.ConsumeVerificationCodeRequest{
			Email:        userEmail,
			ProvidedCode: code,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	token := res.Msg.GetToken()
	span.AddEvent("login successful", trace.WithAttributes(attribute.KeyValue{
		Key:   "token",
		Value: attribute.StringValue(token),
	}))

	userRes, err := service.VerifyToken(ctx, &connect.Request[authv1.VerifyTokenRequest]{
		Msg: &authv1.VerifyTokenRequest{
			Token: token,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, userEmail, userRes.Msg.GetEmail())
}
