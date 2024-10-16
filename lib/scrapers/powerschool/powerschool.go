package powerschool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/timezone"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
)

var tracer = telemetry.Tracer("vcassist.lib.scrapers.powerschool")

type Client struct {
	http *resty.Client
}

func NewClient(baseUrl string) (*Client, error) {
	client := resty.New()
	client.SetTimeout(time.Minute)
	client.SetBaseURL(baseUrl)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.SetCookieJar(jar)
	client.SetHeader("user-agent", "okhttp/4.9.1")

	// 2 requests max per second
	// max burst >= 2 just means that no requests will be dropped
	rateLimiter := rate.NewLimiter(2, 2)
	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		err = rateLimiter.Wait(req.Context())
		if err != nil {
			return err
		}
		return nil
	})

	return &Client{http: client}, nil
}

func (c *Client) SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	restyutil.InstrumentClient(c.http, tracer, out)
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

	now := timezone.Now()
	expiresAt = now.Add(time.Second * time.Duration(openidToken.ExpiresIn))

	span.SetAttributes(
		attribute.String("now", now.Format(time.ANSIC)),
		attribute.String("expiresAt", expiresAt.Format(time.ANSIC)),
	)

	return expiresAt, nil
}

func DecodeBulletinTimestamp(tstr string) (time.Time, error) {
	return time.Parse("2024-08-13", tstr)
}

func DecodeTimestamp(tstr string) (time.Time, error) {
	// aka. parse by ISO timestamp
	return time.Parse(time.RFC3339, tstr)
}
