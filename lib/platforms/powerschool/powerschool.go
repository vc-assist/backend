package powerschool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/telemetry"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("platforms/powerschool")

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
	client.SetHeader("user-agent", "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.6478.110 Mobile Safari/537.36")

	telemetry.InstrumentResty(client, "platform/powerschool/http")

	return &Client{http: client}, nil
}

func (c *Client) LoginOAuth(ctx context.Context, token string) (time.Time, error) {
	ctx, span := tracer.Start(ctx, "client:LoginOAuth")
	defer span.End()

	var openidToken oauth.OpenIdToken
	err := json.Unmarshal([]byte(token), &openidToken)
	if err != nil {
		return time.Now(), err
	}

	c.http.
		SetHeader("Authorization", fmt.Sprintf(
			"%s %s",
			openidToken.TokenType,
			openidToken.AccessToken,
		)).
		SetHeader("profileUri", openidToken.IdToken).
		SetHeader("ServerURL", c.http.BaseURL)

	expiresAt := time.Now().Add(time.Second * time.Duration(openidToken.ExpiresIn))
	return expiresAt, nil
}

func DecodeSectionMeetingTimestamp(tstr string) (time.Time, error) {
	// aka. parse by ISO timestamp
	return time.Parse(time.RFC3339, tstr)
}
