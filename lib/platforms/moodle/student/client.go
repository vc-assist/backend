package moodlestudent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
	"vcassist-backend/lib/htmlutil"
	"vcassist-backend/lib/telemetry"

	"github.com/PuerkitoBio/goquery"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("platforms/moodle/student")

var InvalidCredentials = errors.New("Incorrect username or password.")

type Client struct {
	ClientId string
	BaseUrl  *url.URL

	http  *resty.Client
	cache webpageCache
}

type ClientOptions struct {
	BaseUrl  string
	Username string
	Password string
	Cache    *badger.DB
	// a unique id for this client, used for cache
	ClientId string
}

func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	if tracer == nil {
		panic("you must call SetupTracing before calling any library methods")
	}

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

	telemetry.InstrumentResty(client, "platform/moodle/http")

	cache := webpageCache{
		db:      opts.Cache,
		baseUrl: baseUrl,
	}

	return &Client{
		ClientId: opts.ClientId,
		BaseUrl:  baseUrl,

		http:  client,
		cache: cache,
	}, nil
}

func (c *Client) LoginUsernamePassword(ctx context.Context, username, password string) error {
	ctx, span := tracer.Start(ctx, "client:authenticate")
	defer span.End()

	res, err := c.http.R().
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

	c.http.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, err = c.http.R().
		SetContext(ctx).
		SetBody(values.Encode()).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		Post("/login/index.php")
	c.http.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))
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

type Course = htmlutil.Anchor

const COURSE_LIST_LIFETIME = int64((time.Hour / time.Second) * 24 * 30 * 6)

func (c *Client) Courses(ctx context.Context) ([]Course, error) {
	ctx, span := tracer.Start(ctx, "client:getCourses")
	defer span.End()

	page, err := c.cache.get(ctx, c.ClientId, "/index.php")
	if err == nil {
		span.SetStatus(codes.Ok, "CACHE HIT")
		return page.Anchors, nil
	}

	if err != errWebpageNotFound {
		span.RecordError(err)
		span.AddEvent("CACHE ERROR", trace.WithAttributes(attribute.KeyValue{
			Key:   "log.severity",
			Value: attribute.StringValue("WARN"),
		}))
	}

	res, err := c.http.R().
		SetContext(ctx).
		Get("/index.php")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	anchors := htmlutil.GetAnchors(ctx, doc.Find("ul.unlist a"))

	err = c.cache.set(ctx, c.ClientId, "/index.php", webpage{
		Contents:  res.Body(),
		Anchors:   anchors,
		ExpiresAt: time.Now().Unix() + COURSE_LIST_LIFETIME,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache request")
	}

	return anchors, nil
}

type Section = htmlutil.Anchor

const SECTION_LIST_LIFETIME = int64(time.Hour / time.Second * 24)

func (c *Client) Sections(ctx context.Context, course Course) ([]Section, error) {
	ctx, span := tracer.Start(ctx, "client:getSections")
	defer span.End()

	endpoint := course.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	page, err := c.cache.get(ctx, c.ClientId, endpoint)
	if err == nil {
		span.SetStatus(codes.Ok, "CACHE HIT")
		return page.Anchors, nil
	}

	res, err := c.http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	anchors := htmlutil.GetAnchors(ctx, doc.Filter(".course-content a.nav-link"))

	err = c.cache.set(ctx, c.ClientId, endpoint, webpage{
		Contents:  res.Body(),
		Anchors:   anchors,
		ExpiresAt: time.Now().Unix() + SECTION_LIST_LIFETIME,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache request")
	}

	return anchors, nil
}

type Resource = htmlutil.Anchor

const RESOURCE_LIST_LIFETIME = int64(time.Minute * 15 / time.Second)

func (c *Client) Resources(ctx context.Context, section Section) ([]Resource, error) {
	ctx, span := tracer.Start(ctx, "client:getResources")
	defer span.End()

	endpoint := section.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	page, err := c.cache.get(ctx, c.ClientId, endpoint)
	if err == nil {
		span.SetStatus(codes.Ok, "CACHE HIT")
		return page.Anchors, nil
	}

	res, err := c.http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	anchors := htmlutil.GetAnchors(ctx, doc.Filter("li.activity a"))

	err = c.cache.set(ctx, c.ClientId, endpoint, webpage{
		Contents:  res.Body(),
		Anchors:   anchors,
		ExpiresAt: time.Now().Unix() + RESOURCE_LIST_LIFETIME,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache request")
	}

	return anchors, nil
}

type Chapter = htmlutil.Anchor

const CHAPTER_LIST_LIFETIME = int64(time.Minute * 15 / time.Second)

func (c *Client) Chapters(ctx context.Context, resource Resource) ([]Chapter, error) {
	ctx, span := tracer.Start(ctx, "client:getBooks")
	defer span.End()

	endpoint := resource.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	page, err := c.cache.get(ctx, c.ClientId, endpoint)
	if err == nil {
		span.SetStatus(codes.Ok, "CACHE HIT")
		return page.Anchors, nil
	}

	res, err := c.http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	tableOfContents := htmlutil.GetAnchors(ctx, doc.Filter("div.columnleft li a"))

	currentContents, err := doc.Find("div[role=main] div.box").Html()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialized html content")
		return nil, err
	}
	currentChapter := doc.Find("div.columnleft li").Text()

	anchors := append(tableOfContents, htmlutil.Anchor{
		Href: resource.Href,
		Name: currentChapter,
	})

	err = c.cache.set(ctx, c.ClientId, endpoint, webpage{
		Contents:  []byte(currentContents),
		Anchors:   anchors,
		ExpiresAt: time.Now().Unix() + CHAPTER_LIST_LIFETIME,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache request")
	}

	return anchors, nil
}
