package core

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"vcassist-backend/lib/telemetry"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("platforms/moodle/core")

var InvalidCredentials = fmt.Errorf("Incorrect username or password.")

type Client struct {
	BaseUrl *url.URL
	Http    *resty.Client
}

type ClientOptions struct {
	BaseUrl  string
	Username string
	Password string
}

func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	baseUrl, err := url.Parse(opts.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := resty.New()
	client.SetBaseURL(opts.BaseUrl)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.SetCookieJar(jar)
	client.SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	client.SetRedirectPolicy(resty.DomainCheckRedirectPolicy(baseUrl.Hostname()))

	telemetry.InstrumentResty(client, "platform/moodle/http")

	c := &Client{
		BaseUrl: baseUrl,
		Http:    client,
	}
	return c, nil
}

func (c *Client) LoginUsernamePassword(ctx context.Context, username, password string) error {
	ctx, span := tracer.Start(ctx, "client:LoginUsernamePassword")
	defer span.End()

	res, err := c.Http.R().
		SetContext(ctx).
		Get("/login/index.php")
	if err != nil {
		span.SetStatus(codes.Error, "failed to fetch (1)")
		return err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse html (1)")
		return err
	}
	logintoken := doc.Find("input[name=logintoken]").AttrOr("value", "")
	if logintoken == "" {
		span.SetStatus(codes.Error, "failed to find login token")
		return fmt.Errorf("could not find login token")
	}

	values := url.Values{
		"logintoken": {logintoken},
		"username":   {username},
		"password":   {password},
	}

	redirects := 0
	var loginSuccess = fmt.Errorf("login successful")
	c.Http.SetRedirectPolicy(
		resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
			redirects++
			if req.URL.Query().Get("testsession") != "" {
				return loginSuccess
			}
			return nil
		}),
	)
	defer c.Http.SetRedirectPolicy(resty.DomainCheckRedirectPolicy(c.BaseUrl.Hostname()))

	res, err = c.Http.R().
		SetContext(ctx).
		SetBody(values.Encode()).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		Post("/login/index.php")
	if err == nil {
		if redirects == 0 {
			span.SetStatus(codes.Error, "Something went wrong, response didn't redirect (is cloudflare adapting?")
			return fmt.Errorf("Something went wrong, response didn't redirect (is cloudflare adapting?")
		}
		span.SetStatus(codes.Error, InvalidCredentials.Error())
		return InvalidCredentials
	}

	if strings.Contains(err.Error(), "login successful") {
		return nil
	}
	span.SetStatus(codes.Error, "failed to post login request")
	return err
}
