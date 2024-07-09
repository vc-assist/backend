package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
	"vcassist-backend/lib/htmlutil"
	"vcassist-backend/lib/telemetry"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("scrapers/moodle/core")

var LoginFailed = fmt.Errorf("Failed to login to your account.")

type Client struct {
	BaseUrl *url.URL
	Http    *resty.Client
	Sesskey string
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
	client.GetClient().Transport = cloudflarebp.AddCloudFlareByPass(client.GetClient().Transport)

	client.SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	client.SetRedirectPolicy(resty.DomainCheckRedirectPolicy(baseUrl.Hostname()))
	client.SetTimeout(time.Second * 30)

	telemetry.InstrumentResty(client, "scrapers/moodle/http")

	c := &Client{
		BaseUrl: baseUrl,
		Http:    client,
	}
	return c, nil
}

var moodleConfigRegex = regexp.MustCompile(`(?m)M\.cfg *= *(.+?);`)

func getSesskey(ctx context.Context, doc *goquery.Document) string {
	ctx, span := tracer.Start(ctx, "getMoodleConfig")
	defer span.End()

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
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to unmarshal moodle config")
			return ""
		}
		return cfg.Sesskey
	}

	return ""
}

func (c *Client) DefaultRedirectPolicy() resty.RedirectPolicy {
	return resty.DomainCheckRedirectPolicy(c.BaseUrl.Hostname())
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

	res, err = c.Http.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"logintoken": logintoken,
			"username":   username,
			"password":   password,
		}).
		Post("/login/index.php")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make login request")
		return err
	}

	res, err = c.Http.R().
		SetContext(ctx).
		Get("/")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to request dashboard after login")
		return err
	}
	doc, err = goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse login page html")
		return err
	}

	if len(doc.Find("div.usermenu span.login").Nodes) > 0 {
		span.SetStatus(codes.Error, LoginFailed.Error())
		return LoginFailed
	}

	c.Sesskey = getSesskey(ctx, doc)
	return nil
}
