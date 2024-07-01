package powerschoolapi_test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	powerschoolapi "vcassist-backend/cmd/powerschoold"
	"vcassist-backend/cmd/powerschoold/api"
	"vcassist-backend/lib/configuration"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/platforms/powerschool"
	"vcassist-backend/lib/telemetry"

	_ "embed"

	"connectrpc.com/connect"
	"github.com/lqr471814/protocolreg"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	_ "modernc.org/sqlite"
)

var tracer = otel.Tracer("service_test")

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

func getOAuthFlow(t testing.TB, ctx context.Context, service powerschoolapi.PowerschoolService) *api.OAuthFlow {
	authFlow, err := service.GetAuthFlow(
		ctx,
		&connect.Request[api.GetAuthFlowRequest]{Msg: &api.GetAuthFlowRequest{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	oauthFlow := authFlow.Msg.GetOauth()
	return oauthFlow
}

func getLoginUrl(t testing.TB, ctx context.Context, oauthFlow *api.OAuthFlow) string {
	loginUrl, err := oauth.GetLoginUrl(
		ctx,
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

func tokenFromCallbackUrl(t testing.TB, ctx context.Context, oauthFlow *api.OAuthFlow, callbackUrl string) string {
	parsed, err := url.Parse(strings.Trim(string(callbackUrl), " \n\t"))
	if err != nil {
		t.Fatal("failed to parse callback url", callbackUrl, err)
	}

	authcode := parsed.Query().Get("code")
	if authcode == "" {
		t.Fatal("could not get auth code", callbackUrl)
	}

	token, _, err := oauth.GetToken(
		ctx,
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

func promptForToken(t testing.TB, ctx context.Context, service powerschoolapi.PowerschoolService) (string, func(t testing.TB)) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	callbackFilepath := filepath.Join(cwd, "callback_url.tmp")
	os.Remove(callbackFilepath)

	cleanupProtocol := createPSProtocolHandler(t, callbackFilepath)
	oauthFlow := getOAuthFlow(t, ctx, service)
	loginUrl := getLoginUrl(t, ctx, oauthFlow)

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

		token := tokenFromCallbackUrl(t, ctx, oauthFlow, string(callbackUrl))
		return token, cleanupProtocol
	}
}

//go:embed db/schema.sql
var schemaSql string

func setup(t testing.TB, dbname string) (powerschoolapi.PowerschoolService, func()) {
	cleanupTel := telemetry.SetupForTesting(t, "test:powerschoold")

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

	return service, func() {
		cancelOAuthd()
		cleanupTel()
	}
}

func provideNewToken(t testing.TB, ctx context.Context, service powerschoolapi.PowerschoolService, id string) {
	token, cleanup := promptForToken(t, ctx, service)
	defer cleanup(t)

	fmt.Println("====== TEST TOKEN =====")
	fmt.Println(token)
	fmt.Println("=======================")

	providedOAuth, err := service.ProvideOAuth(
		ctx,
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
		ctx,
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
	service, cleanup := setup(t, "oauth_test_state.db")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestOAuth")
	defer span.End()

	studentId := "student_id"

	hasAuth, err := service.GetAuthStatus(
		ctx,
		&connect.Request[api.GetAuthStatusRequest]{
			Msg: &api.GetAuthStatusRequest{
				StudentId: studentId,
			},
		},
	)
	if err != nil || !hasAuth.Msg.GetIsAuthenticated() {
		provideNewToken(t, ctx, service, studentId)
	}

	foundStudentData, err := service.GetStudentData(
		ctx,
		&connect.Request[api.GetStudentDataRequest]{
			Msg: &api.GetStudentDataRequest{
				StudentId: studentId,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	require.NotNil(t, foundStudentData.Msg.GetProfile())
	require.NotEmpty(t, foundStudentData.Msg.GetProfile().GetGuid())
	require.NotEmpty(t, foundStudentData.Msg.GetProfile().GetFirstName())
	require.NotEmpty(t, foundStudentData.Msg.GetProfile().GetLastName())

	require.Greater(t, len(foundStudentData.Msg.GetProfile().GetSchools()), 0, "provided powerschool account must be a part of at least one school")
	for _, school := range foundStudentData.Msg.GetProfile().GetSchools() {
		require.NotEmpty(t, school.GetName())
	}

	courses := foundStudentData.Msg.GetCourseData()
	if len(courses) > 0 {
		for _, course := range courses {
			require.NotEmpty(t, course.GetGuid())
			require.NotEmpty(t, course.GetName())
			require.NotEmpty(t, course.GetPeriod())
		}
	}

	meetings := foundStudentData.Msg.GetMeetings().SectionMeetings
	if len(meetings) > 0 {
		for _, meeting := range meetings {
			require.NotEmpty(t, meeting.GetSectionGuid())
			_, err = powerschool.DecodeSectionMeetingTimestamp(meeting.GetStart())
			require.Nil(t, err)
			_, err = powerschool.DecodeSectionMeetingTimestamp(meeting.GetStop())
			require.Nil(t, err)
		}
	}
}

func TestBasicNotFound(t *testing.T) {
	service, cleanup := setup(t, ":memory:")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestBasicNotFound")
	defer span.End()

	id := "any_student_id"

	res, err := service.GetAuthStatus(
		ctx,
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
		ctx,
		&connect.Request[api.GetStudentDataRequest]{
			Msg: &api.GetStudentDataRequest{
				StudentId: id,
			},
		},
	)
	require.NotNil(t, err)
}
