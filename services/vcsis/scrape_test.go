package vcsis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/util/oauthutil"
	"vcassist-backend/lib/util/restyutil"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"

	"github.com/go-resty/resty/v2"
	"github.com/lqr471814/protocolreg"
	"github.com/stretchr/testify/require"

	_ "embed"
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

func tokenFromCallbackUrl(t testing.TB, oauthFlow *keychainv1.OAuthFlow, callbackUrl string) string {
	parsed, err := url.Parse(strings.Trim(string(callbackUrl), " \n\t"))
	if err != nil {
		t.Fatal("failed to parse callback url", callbackUrl, err)
	}

	authcode := parsed.Query().Get("code")
	if authcode == "" {
		t.Fatal("could not get auth code", callbackUrl)
	}

	req := oauthutil.TokenRequest{
		GrantType:    "authorization_code",
		ClientId:     oauthFlow.GetClientId(),
		CodeVerifier: oauthFlow.GetCodeVerifier(),
		Scope:        oauthFlow.GetScope(),
		RedirectUri:  oauthFlow.GetRedirectUri(),
		AuthCode:     authcode,
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal("failed to serialize token request", err)
	}

	client := resty.New()
	restyutil.InstrumentClient(client, "vcsis_test", nil, restyInstrumentOutput)

	res, err := client.R().
		SetBody(body).
		Post(oauthFlow.GetTokenRequestUrl())
	if err != nil {
		t.Fatal("failed to request token", err)
	}

	return string(res.Body())
}

func promptForToken(t testing.TB, ctx context.Context, oauthConfig OAuthConfig) string {
	os.MkdirAll(".dev/test_vcsis", 0777)
	callbackPath := ".dev/test_vcsis/callback_url"
	os.Remove(callbackPath)

	cleanupProtocol := createPSProtocolHandler(t, callbackPath)
	defer cleanupProtocol(t)

	flow, err := oauthConfig.GetOAuthFlow()
	if err != nil {
		t.Fatal(err)
	}
	loginUrl, err := oauthutil.GetLoginUrl(
		ctx,
		oauthutil.AuthCodeRequest{
			AccessType:   flow.GetAccessType(),
			Scope:        flow.GetScope(),
			RedirectUri:  flow.GetRedirectUri(),
			CodeVerifier: flow.GetCodeVerifier(),
			ClientId:     flow.GetClientId(),
		},
		flow.GetBaseLoginUrl(),
	)
	if err != nil {
		t.Fatal(err)
	}

	slog.Info("login to your powerschool account:")
	fmt.Println(loginUrl)

	for {
		callbackUrl, err := os.ReadFile(callbackPath)
		if os.IsNotExist(err) {
			time.Sleep(2 * time.Second)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}

		token := tokenFromCallbackUrl(t, flow, string(callbackUrl))
		fmt.Println("---- TOKEN RECEIVED ----")
		fmt.Println(token)
		fmt.Println("------------------------")
		return token
	}
}

func getToken(t testing.TB, ctx context.Context, oauthConfig OAuthConfig) string {
	tokenPath := ".dev/test_vcsis/token"

	info, err := os.Stat(tokenPath)
	if err == nil && time.Now().Sub(info.ModTime()).Hours() < 0.5 {
		cached, err := os.ReadFile(tokenPath)
		if err == nil {
			return string(cached)
		} else {
			slog.Warn("failed to read cached token", "err", err)
		}
	}

	newToken := promptForToken(t, ctx, oauthConfig)
	err = os.WriteFile(tokenPath, []byte(newToken), 0600)
	if err != nil {
		slog.Warn("failed to write to token cache", "err", err)
	}
	return newToken
}

//go:embed weights.json
var weightsFile []byte

func TestScrape(t *testing.T) {
	cleanup := telemetry.SetupForTesting("test:vcsis")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	oauthConfig := OAuthConfig{
		BaseLoginUrl: "https://accounts.google.com/o/oauth2/v2/auth",
		RefreshUrl:   "https://oauth2.googleapis.com/token",
		ClientId:     "162669419438-egansm7coo8n7h301o7042kad9t9uao9.apps.googleusercontent.com",
	}
	token := getToken(t, ctx, oauthConfig)

	client, err := scraper.NewClient("https://vcsnet.powerschool.com")
	if err != nil {
		slog.ErrorContext(ctx, "failed to create powerschool client", "err", err)
		t.Fatal(err)
	}
	client.SetRestyInstrumentOutput(restyutil.NewFilesystemOutput(
		".dev/test_vcsis/resty/powerschool",
	))

	_, err = client.LoginOAuth(ctx, token)
	if err != nil {
		slog.ErrorContext(ctx, "failed to login to powerschool", "err", err)
		t.Fatal(err)
	}
	res, err := ScrapePowerschool(ctx, client)
	if err != nil {
		t.Fatal(err)
	}

	require.NotNil(t, res.GetProfile())
	require.NotEmpty(t, res.GetProfile().GetGuid())
	require.NotEmpty(t, res.GetProfile().GetName())
	require.Greater(t, len(res.GetSchools()), 0, "provided powerschool account must be a part of at least one school")
	for _, school := range res.GetSchools() {
		require.NotEmpty(t, school.GetName())
	}

	courses := res.GetCourses()
	require.NotEmpty(t, courses)
	for _, course := range courses {
		require.NotEmpty(t, course.GetGuid())
		require.NotEmpty(t, course.GetName())
		require.NotEmpty(t, course.GetPeriod())
		require.NotEmpty(t, course.GetTeacherEmail())
		require.NotEmpty(t, course.GetTeacher())

		slog.Debug(
			"course",
			"guid", course.GetGuid(),
			"name", course.GetName(),
			"period", course.GetPeriod(),
			"teacher", course.GetTeacher(),
			"teacher_email", course.GetTeacherEmail(),
			"room", course.GetRoom(),
		)

		if course.GetRoom() == "" {
			slog.Warn("no room found", "course", course.GetName())
		}

		if len(course.GetMeetings()) == 0 {
			slog.Warn("no meetings present", "course", course.GetName())
		}
		for _, meeting := range course.GetMeetings() {
			slog.Debug(
				"meeting",
				"start", time.Unix(meeting.GetStart(), 0).Format(time.RFC850),
				"stop", time.Unix(meeting.GetStop(), 0).Format(time.RFC850),
			)
			require.NotEmpty(t, meeting.GetStart())
			require.NotEmpty(t, meeting.GetStop())
		}

		if len(course.GetAssignments()) == 0 {
			slog.Warn("no assignments present", "course", course.GetName())
		}
		for _, assign := range course.GetAssignments() {
			require.NotEmpty(t, assign.GetTitle())
			require.NotEmpty(t, assign.GetCategory())
			require.NotEmpty(t, assign.GetDueDate())
		}
	}

	var weightData WeightData
	err = json.Unmarshal(weightsFile, &weightData)
	if err != nil {
		t.Fatal(err)
	}

	mapping := map[string]string{}
	for _, course := range courses {
		for weightCourse := range weightData {
			if course.GetName() == weightCourse {
				mapping[weightCourse] = course.GetName()
				break
			}
		}
	}
	AddWeights(ctx, courses, weightData, mapping)

	for _, course := range courses {
		if len(course.GetAssignmentCategories()) == 0 {
			slog.Warn("no assignment categories", "course", course.GetName())
		}
	}
}
