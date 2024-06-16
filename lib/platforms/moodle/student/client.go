package moodlestudent

import (
	"context"
	"errors"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
	"vcassist-backend/lib/htmlutil"

	"github.com/PuerkitoBio/goquery"
	"github.com/dgraph-io/badger/v4"
	"github.com/dubonzi/otelresty"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("platform/moodle/student")

var InvalidCredentials = errors.New("Incorrect username or password.")

type Client struct {
	ClientId string
	BaseUrl  *url.URL
	http     *resty.Client
	cache    webpageCache
}

type ClientOptions struct {
	BaseUrl  string
	Username string
	Password string
	Cache    *badger.DB
	// a unique id for this client, used for cache
	ClientId string
}

func authenticate(ctx context.Context, client *resty.Client, opts ClientOptions) error {
	ctx, span := tracer.Start(ctx, "new_client:authenticate")
	defer span.End()

	res, err := client.R().Get("/login/index.php")
	if err != nil {
		span.SetStatus(codes.Error, "failed to fetch (1)")
		return err
	}
	doc, err := goquery.NewDocumentFromReader(res.RawBody())
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
		"username":   {opts.Username},
		"password":   {opts.Password},
	}

	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, err = client.R().
		SetBody(values.Encode()).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		Post("/login/index.php")
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))
	if err != nil {
		span.SetStatus(codes.Error, "failed to post login")
		return err
	}

	location := res.Header().Get("location")
	if res.StatusCode() != 303 || location == "" {
		span.SetStatus(codes.Error, "Something went wrong, response didn't redirect (is cloudflare adapting?")
		return fmt.Errorf("Something went wrong, response didn't redirect (is cloudflare adapting?")
	}
	if !strings.Contains(location, "testsession") {
		span.SetStatus(codes.Error, InvalidCredentials.Error())
		return InvalidCredentials
	}

	return nil
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

	otelresty.TraceClient(
		client,
		otelresty.WithTracerName("moodle-http"),
	)

	err = authenticate(ctx, client, opts)
	if err != nil {
		return nil, err
	}

	cache := webpageCache{
		db:      opts.Cache,
		baseUrl: baseUrl,
	}

	return &Client{
		ClientId: opts.ClientId,
		BaseUrl:  baseUrl,
		http:     client,
		cache:    cache,
	}, nil
}

type Course = htmlutil.Anchor

const COURSE_LIST_LIFETIME = int64((time.Hour / time.Second) * 24 * 30 * 6)

func (c *Client) Courses(ctx context.Context) ([]Course, error) {
	ctx, span := tracer.Start(ctx, "client:getCourses")
	defer span.End()

	page, err := c.cache.get(ctx, c.ClientId, "/index.php")
	if err == nil {
		span.SetStatus(codes.Ok, "CACHE HIT")
		return page.anchors, nil
	}

	if err != errWebpageNotFound {
		span.RecordError(err)
		span.AddEvent("CACHE ERROR", trace.WithAttributes(attribute.KeyValue{
			Key:   "log.severity",
			Value: attribute.StringValue("WARN"),
		}))
	}

	res, err := c.http.R().Get("/index.php")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(res.RawBody())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	anchors := htmlutil.GetAnchors(ctx, doc.Find("ul.unlist a"))

	err = c.cache.set(ctx, c.ClientId, "/index.php", webpage{
		contents:  res.Body(),
		anchors:   anchors,
		createdAt: time.Now().Unix(),
		lifetime:  COURSE_LIST_LIFETIME,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache request")
	}

	return anchors, nil
}

type Section = htmlutil.Anchor

func (c *Client) Sections(ctx context.Context, course Course) ([]Section, error) {
	ctx, span := tracer.Start(ctx, "client:getSections")
	defer span.End()

	endpoint := course.Url.String()

	page, err := c.cache.get(ctx, c.ClientId, endpoint)
	if err == nil {
		span.SetStatus(codes.Ok, "CACHE HIT")
		return page.anchors, nil
	}

	res, err := c.http.R().Get(endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(res.RawBody())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	anchors := htmlutil.GetAnchors(ctx, doc.Filter(".course-content a.nav-link"))

	err = c.cache.set(ctx, c.ClientId, endpoint, webpage{
		contents:  res.Body(),
		anchors:   anchors,
		createdAt: time.Now().Unix(),
		lifetime:  COURSE_LIST_LIFETIME,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache request")
	}

	return anchors, nil
}

type Resource = htmlutil.Anchor

func (c *Client) Resources() {}

