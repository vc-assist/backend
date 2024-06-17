package powerschool

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"
	"vcassist-backend/lib/oauth"

	"github.com/dubonzi/otelresty"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("platform/powerschool")

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

	otelresty.TraceClient(
		client,
		otelresty.WithTracerName("powerschool-http"),
	)

	return &Client{http: client}, nil
}

func (c *Client) LoginOAuth(ctx context.Context, token string) (time.Time, error) {
	ctx, span := tracer.Start(ctx, "client:loginOAuth2")
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

	_, err = c.GetAllStudents(ctx)
	if err != nil {
		return time.Now(), err
	}

	expiresAt := time.Now().Add(time.Second * time.Duration(openidToken.ExpiresIn))
	return expiresAt, nil
}

type OAuthConfig struct {
	BaseLoginUrl string
	RefreshUrl   string
	ClientId     string
}

func (o OAuthConfig) LoginUrl(ctx context.Context) (string, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}

	req := oauth.AuthCodeRequest{
		AccessType:   "offline",
		Scope:        "openid email profile",
		RedirectUri:  "com.powerschool.portal://",
		ClientId:     o.ClientId,
		CodeVerifier: hex.EncodeToString(nonce),
	}

	return req.GetLoginUrl(o.BaseLoginUrl)
}

func (o OAuthConfig) Refresh(ctx context.Context, token string) (time.Time, error) {
	ctx, span := tracer.Start(ctx, "refreshOAuth2")
	defer span.End()

	var openidToken oauth.OpenIdToken
	err := json.Unmarshal([]byte(token), &openidToken)
	if err != nil {
		return time.Now(), err
	}

	idToken, err := openidToken.Refresh(ctx, o.RefreshUrl, o.ClientId)
	if err != nil {
		return time.Now(), err
	}

	expiresAt := time.Now().Add(time.Duration(idToken.ExpiresIn) * time.Second)
	return expiresAt, nil
}
