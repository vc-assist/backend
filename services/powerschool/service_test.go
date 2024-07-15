package powerschoold

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
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/powerschool/api"

	_ "embed"

	"connectrpc.com/connect"
	"github.com/lqr471814/protocolreg"
	"github.com/stretchr/testify/require"
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

func getOAuthFlow(t testing.TB, ctx context.Context, service Service) *api.GetOAuthFlowResponse {
	authFlow, err := service.GetOAuthFlow(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return authFlow.Msg
}

func getLoginUrl(t testing.TB, ctx context.Context, oauthFlow *api.GetOAuthFlowResponse) string {
	loginUrl, err := oauth.GetLoginUrl(
		ctx,
		oauth.AuthCodeRequest{
			AccessType:   oauthFlow.GetFlow().GetAccessType(),
			Scope:        oauthFlow.GetFlow().GetScope(),
			RedirectUri:  oauthFlow.GetFlow().GetRedirectUri(),
			CodeVerifier: oauthFlow.GetFlow().GetCodeVerifier(),
			ClientId:     oauthFlow.GetFlow().GetClientId(),
		},
		oauthFlow.GetFlow().GetBaseLoginUrl(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return loginUrl
}

func tokenFromCallbackUrl(t testing.TB, ctx context.Context, oauthFlow *api.GetOAuthFlowResponse, callbackUrl string) string {
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
			ClientId:     oauthFlow.GetFlow().GetClientId(),
			CodeVerifier: oauthFlow.GetFlow().GetCodeVerifier(),
			Scope:        oauthFlow.GetFlow().GetScope(),
			RedirectUri:  oauthFlow.GetFlow().GetRedirectUri(),
			AuthCode:     authcode,
		},
		oauthFlow.GetFlow().GetTokenRequestUrl(),
	)
	if err != nil {
		t.Fatal("failed to fetch token", callbackUrl, err)
	}

	return token
}

func promptForToken(t testing.TB, ctx context.Context, service Service) (string, func(t testing.TB)) {
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

func setup(t testing.TB, dbname string) (Service, func()) {
	cleanupTel := telemetry.SetupForTesting(t, "test:powerschoold")

	sqlite, err := sql.Open("sqlite", dbname)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.Exec(schemaSql)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatal(err)
	}

	oauthConfig := OAuthConfig{
		BaseLoginUrl: "https://accounts.google.com/o/oauth2/v2/auth",
		RefreshUrl:   "https://oauth2.googleapis.com/token",
		ClientId:     "162669419438-egansm7coo8n7h301o7042kad9t9uao9.apps.googleusercontent.com",
	}

	oauthd, err := NewOAuthDaemon(sqlite, oauthConfig)
	if err != nil {
		t.Fatal(err)
	}
	oauthdCtx, cancelOAuthd := context.WithCancel(context.Background())
	oauthd.Start(oauthdCtx)

	service := NewService(sqlite, "https://vcsnet.powerschool.com", oauthConfig)

	return service, func() {
		cancelOAuthd()
		cleanupTel()
	}
}

func provideNewToken(t testing.TB, ctx context.Context, service Service, id string) {
	token, cleanup := promptForToken(t, ctx, service)
	defer cleanup(t)

	fmt.Println("====== TEST TOKEN =====")
	fmt.Println(token)
	fmt.Println("=======================")

	_, err := service.ProvideOAuth(ctx, &connect.Request[api.ProvideOAuthRequest]{
		Msg: &api.ProvideOAuthRequest{
			StudentId: id,
			Token:     token,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

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

	studentDataRes, err := service.GetStudentData(ctx, &connect.Request[api.GetStudentDataRequest]{
		Msg: &api.GetStudentDataRequest{
			StudentId: studentId,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundStudentData := studentDataRes.Msg

	require.NotNil(t, foundStudentData.GetProfile())
	require.NotEmpty(t, foundStudentData.GetProfile().GetGuid())
	require.NotEmpty(t, foundStudentData.GetProfile().GetFirstName())
	require.NotEmpty(t, foundStudentData.GetProfile().GetLastName())

	require.Greater(t, len(foundStudentData.GetProfile().GetSchools()), 0, "provided powerschool account must be a part of at least one school")
	for _, school := range foundStudentData.GetProfile().GetSchools() {
		require.NotEmpty(t, school.GetName())
	}

	courses := foundStudentData.GetCourseData()
	if len(courses) > 0 {
		for _, course := range courses {
			require.NotEmpty(t, course.GetGuid())
			require.NotEmpty(t, course.GetName())
			require.NotEmpty(t, course.GetPeriod())
		}
	}

	meetings := foundStudentData.GetMeetings().GetSectionMeetings()
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

	_, err = service.GetStudentData(ctx, &connect.Request[api.GetStudentDataRequest]{
		Msg: &api.GetStudentDataRequest{
			StudentId: id,
		},
	})
	require.NotNil(t, err)
}
