package powerschool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/timezone"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("scrapers/powerschool")

type Client struct {
	http *resty.Client
}

func NewClient(baseUrl string) (*Client, error) {
	client := resty.New()
	client.SetBaseURL(baseUrl)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.SetCookieJar(jar)
	client.SetHeader("user-agent", "okhttp/4.9.1")

	telemetry.InstrumentResty(client, "scrapers/powerschool/http")

	return &Client{http: client}, nil
}

func (c *Client) LoginOAuth(ctx context.Context, token string) (expiresAt time.Time, err error) {
	ctx, span := tracer.Start(ctx, "LoginOAuth")
	defer span.End()

	var openidToken oauth.OpenIdToken
	err = json.Unmarshal([]byte(token), &openidToken)
	if err != nil {
		return timezone.Now(), err
	}

	c.http.
		SetHeader("Authorization", fmt.Sprintf(
			"%s %s",
			openidToken.TokenType,
			openidToken.AccessToken,
		)).
		SetHeader("profileUri", openidToken.IdToken).
		SetHeader("ServerURL", c.http.BaseURL)

	expiresAt = timezone.Now().Add(time.Second * time.Duration(openidToken.ExpiresIn))
	return expiresAt, nil
}

func DecodeAssignmentTime(tstr string) (time.Time, error) {
	// aka. parse by ISO timestamp
	return time.Parse(time.RFC3339, tstr)
}

func DecodeCourseTermTime(tstr string) (time.Time, error) {
	// aka. parse by ISO timestamp
	return time.Parse(time.RFC3339, tstr)
}

func DecodeSectionMeetingTimestamp(tstr string) (time.Time, error) {
	// aka. parse by ISO timestamp
	return time.Parse(time.RFC3339, tstr)
}
