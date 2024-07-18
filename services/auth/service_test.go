package auth

import (
	"context"
	"io"
	"log"
	"regexp"
	"testing"
	"vcassist-backend/lib/testutil"
	authv1 "vcassist-backend/proto/vcassist/services/auth/v1"

	"connectrpc.com/connect"
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

//go:embed db/schema.sql
var schemaSql string

func setup(t testing.TB) (Service, func()) {
	res, cleanup := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "services/auth",
		DbSchema: schemaSql,
	})

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

	service := NewService(res.DB, EmailConfig{
		Server:       "localhost",
		Port:         1025,
		EmailAddress: "alice@email.com",
		Password:     "default",
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
		Get("http://127.0.0.1:1080/messages/1.plain")
	if err != nil {
		t.Fatal(err)
	}
	code := codeRegex.FindString(res.String())
	return code
}

func TestLoginFlow(t *testing.T) {
	service, cleanup := setup(t)
	defer cleanup()

	tracer := otel.Tracer("service_test")
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
