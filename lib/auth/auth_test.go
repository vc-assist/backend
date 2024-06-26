package auth

import (
	"context"
	"database/sql"
	"io"
	"log"
	"regexp"
	"testing"
	"vcassist-backend/lib/telemetry"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	_ "modernc.org/sqlite"

	_ "embed"
)

var codeRegex = regexp.MustCompile(`[A-F0-9]{4}-[A-F0-9]{4}`)

func TestGenerateCode(t *testing.T) {
	code, err := generateVerificationCode()
	if err != nil {
		t.Fatal(err)
	}
	require.True(t, codeRegex.MatchString(code))
}

func setupTelemetry(t testing.TB) func(testing.TB) {
	tel, err := telemetry.SetupFromEnv(context.Background(), "test:auth")
	if err != nil {
		t.Fatal(err)
	}
	return func(t testing.TB) {
		err := tel.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}
}

//go:embed db/schema.sql
var schemaSql string

func setupService(t testing.TB) (AuthService, func(t testing.TB)) {
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
				ExposedPorts: []string{"1025:1025", "1080:1080"},
				WaitingFor:   wait.ForLog("smtp://0.0.0.0:1025"),
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	service := NewAuthService(sqlite, EmailConfig{
		Server:       "localhost",
		Port:         1025,
		EmailAddress: "alice@email.com",
		Password:     "default",
	})

	return service, func(t testing.TB) {
		err := smtp.Terminate(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}
}

var globalClient = resty.New()

func getVerificationCodeFromEmail(t testing.TB) string {
	res, err := globalClient.R().
		Get("http://127.0.0.1:1080/messages/1.plain")
	if err != nil {
		t.Fatal(err)
	}
	code := codeRegex.FindString(res.String())
	return code
}

func TestLoginFlow(t *testing.T) {
	cleanup := setupTelemetry(t)
	defer cleanup(t)
	service, cleanup := setupService(t)
	defer cleanup(t)

	tracer := otel.Tracer("service_test")
	ctx, span := tracer.Start(context.Background(), "TestLoginFlow")
	defer span.End()

	userEmail := "bob@email.com"

	err := service.StartLogin(ctx, userEmail)
	if err != nil {
		t.Fatal(err)
	}
	code := getVerificationCodeFromEmail(t)

	token, err := service.ConsumeVerificationCode(ctx, userEmail, code)
	if err != nil {
		t.Fatal(err)
	}
	span.AddEvent("login successful", trace.WithAttributes(attribute.KeyValue{
		Key:   "token",
		Value: attribute.StringValue(token),
	}))

	user, err := service.VerifyToken(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, userEmail, user.Email)
}
