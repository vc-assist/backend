package powerschoolapi_test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"
	powerschoolapi "vcassist-backend/cmd/powerschool_api"
	"vcassist-backend/cmd/powerschool_api/api"
	"vcassist-backend/lib/configuration"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/telemetry"

	_ "embed"

	"connectrpc.com/connect"
	"github.com/lqr471814/protocolreg"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

func createPSProtocolHandler(t testing.TB, tokenpath string) func(t testing.TB) {
	switch runtime.GOOS {
	case "linux":
		err := protocolreg.RegisterLinux("powerschool_authenticator", protocolreg.LinuxOptions{
			Exec:      fmt.Sprintf(`sh -c "echo %%u > %s"`, tokenpath),
			Protocols: []string{"com.powerschool.portal"},
			Metadata: protocolreg.LinuxMetadataOptions{
				Name: "Powerschool",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return func(t testing.TB) {
		switch runtime.GOOS {
		case "linux":
			err := protocolreg.UnregisterLinux("powerschool_authenticator")
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func getOAuthFlow(t testing.TB, service powerschoolapi.PowerschoolService) *api.OAuthFlow {
	authFlow, err := service.GetAuthFlow(
		context.Background(),
		&connect.Request[api.GetAuthFlowRequest]{Msg: &api.GetAuthFlowRequest{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	oauthFlow := authFlow.Msg.GetOauth()
	return oauthFlow
}

func getLoginUrl(t testing.TB, oauthFlow *api.OAuthFlow) string {
	loginUrl, err := oauth.GetLoginUrl(
		context.Background(),
		oauth.AuthCodeRequest{
			AccessType:   oauthFlow.GetAccessType(),
			Scope:        oauthFlow.GetScope(),
			RedirectUri:  oauthFlow.GetRedirectUri(),
			CodeVerifier: oauthFlow.GetCodeVerifier(),
			ClientId:     oauthFlow.GetClientId(),
		},
		oauthFlow.GetBaseLoginUrl(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return loginUrl
}

func tokenFromCallbackUrl(t testing.TB, oauthFlow *api.OAuthFlow, callbackUrl string) string {
	parsed, err := url.Parse(strings.Trim(string(callbackUrl), " \n\t"))
	if err != nil {
		t.Fatal("failed to parse callback url", callbackUrl, err)
	}

	authcode := parsed.Query().Get("code")
	if authcode == "" {
		t.Fatal("could not get auth code", callbackUrl)
	}

	token, _, err := oauth.GetToken(
		context.Background(),
		oauth.TokenRequest{
			ClientId:     oauthFlow.GetClientId(),
			CodeVerifier: oauthFlow.GetCodeVerifier(),
			Scope:        oauthFlow.GetScope(),
			RedirectUri:  oauthFlow.GetRedirectUri(),
			AuthCode:     authcode,
		},
		oauthFlow.GetTokenRequestUrl(),
	)
	if err != nil {
		t.Fatal("failed to fetch token", callbackUrl, err)
	}

	return token
}

func promptForToken(t testing.TB, service powerschoolapi.PowerschoolService) (string, func(t testing.TB)) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	callbackFilepath := path.Join(cwd, "callback_url")
	os.Remove(callbackFilepath)

	cleanupProtocol := createPSProtocolHandler(t, callbackFilepath)
	oauthFlow := getOAuthFlow(t, service)
	loginUrl := getLoginUrl(t, oauthFlow)

	slog.Info("login to your powerschool account:")
	fmt.Println(loginUrl)

	for {
		callbackUrl, err := os.ReadFile(callbackFilepath)
		if os.IsNotExist(err) {
			time.Sleep(2 * time.Second)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}

		token := tokenFromCallbackUrl(t, oauthFlow, string(callbackUrl))
		return token, cleanupProtocol
	}
}

//go:embed db/schema.sql
var schemaSql string

func setupService(t testing.TB, dbname string) (powerschoolapi.PowerschoolService, func(t testing.TB)) {
	sqlite, err := sql.Open("sqlite", dbname)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.Exec(schemaSql)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatal(err)
	}

	config, err := configuration.ReadConfig[powerschoolapi.Config]("config.json5")
	if err != nil {
		t.Fatal("failed to load configuration", err)
	}

	oauthd, err := powerschoolapi.NewOAuthDaemon(sqlite, config.OAuth)
	if err != nil {
		t.Fatal(err)
	}
	oauthdCtx, cancelOAuthd := context.WithCancel(context.Background())
	oauthd.Start(oauthdCtx)

	service := powerschoolapi.NewPowerschoolService(sqlite, config)

	return service, func(t testing.TB) {
		cancelOAuthd()
	}
}

func setupTelemetry(t testing.TB) func(t testing.TB) {
	tel, err := telemetry.SetupFromEnv(context.Background(), "test:powerschool_api")
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

func provideNewToken(t testing.TB, service powerschoolapi.PowerschoolService, id string) {
	token, cleanup := promptForToken(t, service)
	defer cleanup(t)

	fmt.Println("====== TEST TOKEN =====")
	fmt.Println(token)
	fmt.Println("=======================")

	providedOAuth, err := service.ProvideOAuth(
		context.Background(),
		&connect.Request[api.ProvideOAuthRequest]{
			Msg: &api.ProvideOAuthRequest{
				StudentId: id,
				Token:     token,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	slog.Info("provide auth message", "msg", providedOAuth.Msg.GetMessage())
	require.True(t, providedOAuth.Msg.GetSuccess())

	foundToken, err := service.GetAuthStatus(
		context.Background(),
		&connect.Request[api.GetAuthStatusRequest]{
			Msg: &api.GetAuthStatusRequest{
				StudentId: id,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	require.True(t, foundToken.Msg.GetIsAuthenticated())
}

func TestOAuth(t *testing.T) {
	cleanup := setupTelemetry(t)
	defer cleanup(t)
	service, cleanup := setupService(t, "oauth_test_state.db")
	defer cleanup(t)

	studentId := "student_id"

	hasAuth, err := service.GetAuthStatus(
		context.Background(),
		&connect.Request[api.GetAuthStatusRequest]{
			Msg: &api.GetAuthStatusRequest{
				StudentId: studentId,
			},
		},
	)
	if err != nil || !hasAuth.Msg.GetIsAuthenticated() {
		provideNewToken(t, service, studentId)
	}

	foundStudentData, err := service.GetStudentData(
		context.Background(),
		&connect.Request[api.GetStudentDataRequest]{
			Msg: &api.GetStudentDataRequest{
				StudentId: studentId,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	foundStudentDataJson, err := protojson.Marshal(foundStudentData.Msg)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(foundStudentDataJson))

	require.NotNil(t, foundStudentData.Msg.GetProfile())
	require.NotEmpty(t, foundStudentData.Msg.GetProfile().GetGuid())
	require.NotEmpty(t, foundStudentData.Msg.GetProfile().GetFirstName())
	require.NotEmpty(t, foundStudentData.Msg.GetProfile().GetLastName())

	require.Greater(t, len(foundStudentData.Msg.GetProfile().GetSchools()), 0)
	for _, school := range foundStudentData.Msg.GetProfile().GetSchools() {
		require.NotEmpty(t, school.GetName())
	}
}

func TestBasicNotFound(t *testing.T) {
	cleanup := setupTelemetry(t)
	defer cleanup(t)
	service, cleanup := setupService(t, ":memory:")
	defer cleanup(t)

	id := "any_student_id"

	res, err := service.GetAuthStatus(
		context.Background(),
		&connect.Request[api.GetAuthStatusRequest]{
			Msg: &api.GetAuthStatusRequest{
				StudentId: id,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	require.False(t, res.Msg.GetIsAuthenticated())

	_, err = service.GetStudentData(
		context.Background(),
		&connect.Request[api.GetStudentDataRequest]{
			Msg: &api.GetStudentDataRequest{
				StudentId: id,
			},
		},
	)
	require.NotNil(t, err)
}
