package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
	"vcassist-backend/lib/htmlutil"
	"vcassist-backend/lib/restyutil"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

var LoginFailed = fmt.Errorf("failed to login to your account")

type Client struct {
	BaseUrl *url.URL
	Http    *resty.Client
	Sesskey string
}

type ClientOptions struct {
	BaseUrl string
}

func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	baseUrl, err := url.Parse(opts.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := resty.New()
	client.SetTimeout(time.Minute * 2)
	client.SetBaseURL(opts.BaseUrl)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.SetCookieJar(jar)
	client.GetClient().Transport = cloudflarebp.AddCloudFlareByPass(client.GetClient().Transport)

	client.SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	client.SetRedirectPolicy(resty.DomainCheckRedirectPolicy(baseUrl.Hostname()))
	client.SetTimeout(time.Second * 30)

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

	restyutil.InstrumentClient(client, tracer, restyInstrumentOutput)

	c := &Client{
		BaseUrl: baseUrl,
		Http:    client,
	}
	return c, nil
}

var moodleConfigRegex = regexp.MustCompile(`(?m)M\.cfg *= *(.+?);`)

func getSesskey(ctx context.Context, doc *goquery.Document) string {
	for _, script := range doc.Find("script").Nodes {
		text := htmlutil.GetText(script)
		if !strings.HasPrefix(strings.Trim(text, " \t\n"), "//<![CDATA") {
			continue
		}
		groups := moodleConfigRegex.FindStringSubmatch(text)
		if len(groups) < 2 {
			continue
		}

		var cfg struct {
			Sesskey string `json:"sesskey"`
		}
		err := json.Unmarshal([]byte(groups[1]), &cfg)
		if err != nil {
			slog.ErrorContext(ctx, "getSesskey: failed to unmarshal moodle config", "err", err)
			return ""
		}
		return cfg.Sesskey
	}

	return ""
}

func (c *Client) DefaultRedirectPolicy() resty.RedirectPolicy {
	return resty.DomainCheckRedirectPolicy(c.BaseUrl.Hostname())
}

func wrapLoginError(err error) error {
	return fmt.Errorf("moodle login failed: %v", err)
}

func (c *Client) LoginUsernamePassword(ctx context.Context, username, password string) error {
	ctx, span := tracer.Start(ctx, "LoginUsernamePassword")
	defer span.End()

	res, err := c.Http.R().
		SetContext(ctx).
		Get("/login/index.php")
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch login page", "err", err)
		return wrapLoginError(err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse login page html", "err", err)
		return wrapLoginError(err)
	}

	logintoken := doc.Find("input[name=logintoken]").AttrOr("value", "")
	if logintoken == "" {
		slog.ErrorContext(ctx, "could not find login token")
		return wrapLoginError(fmt.Errorf("could not find login token"))
	}

	res, err = c.Http.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"logintoken": logintoken,
			"username":   username,
			"password":   password,
		}).
		Post("/login/index.php")
	if err != nil {
		slog.ErrorContext(ctx, "failed to make login request", "err", err)
		return wrapLoginError(err)
	}

	res, err = c.Http.R().
		SetContext(ctx).
		Get("/")
	if err != nil {
		slog.ErrorContext(ctx, "failed to request dashboard after login", "err", err)
		return wrapLoginError(err)
	}
	doc, err = goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse post-login html", "err", err)
		return wrapLoginError(err)
	}

	if len(doc.Find("span.avatar.current").Nodes) == 0 {
		slog.WarnContext(ctx, "login failed, likely due to invalid credentials")
		return LoginFailed
	}

	c.Sesskey = getSesskey(ctx, doc)
	return nil
}
