package view

import (
	"bytes"
	"context"
	"net/url"
	"time"
	"vcassist-backend/lib/htmlutil"
	"vcassist-backend/lib/scrapers/moodle/core"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("vcassist.lib.scrapers.moodle.view")

type Client struct {
	ClientId string
	Core     *core.Client
}

type ClientOptions struct {
	// a unique id for this client, used for cache
	ClientId string
}

func NewClient(ctx context.Context, coreClient *core.Client, opts ClientOptions) (Client, error) {
	c := Client{
		ClientId: opts.ClientId,
		Core:     coreClient,
	}
	return c, nil
}

type Course htmlutil.Anchor

func (c Course) Id() string {
	href, err := url.Parse(c.Href)
	if err != nil {
		return ""
	}
	return href.Query().Get("id")
}

func coursesFromAnchors(anchors []htmlutil.Anchor) []Course {
	courses := make([]Course, len(anchors))
	for i := 0; i < len(anchors); i++ {
		a := anchors[i]
		if a == (htmlutil.Anchor{}) {
			continue
		}
		courses[i] = Course{
			Name: a.Name,
			Href: a.Href,
		}
	}
	return courses
}

const COURSE_LIST_LIFETIME = int64((time.Hour / time.Second) * 24 * 30 * 6)

func (c Client) Courses(ctx context.Context) ([]Course, error) {
	ctx, span := tracer.Start(ctx, "Courses")
	defer span.End()

	res, err := c.Core.Http.R().
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

	return coursesFromAnchors(anchors), nil
}

type Section htmlutil.Anchor

func sectionsFromAnchors(anchors []htmlutil.Anchor) []Section {
	sections := make([]Section, len(anchors))
	for i := 0; i < len(anchors); i++ {
		a := anchors[i]
		if a == (htmlutil.Anchor{}) {
			continue
		}
		sections[i] = Section{
			Name: a.Name,
			Href: a.Href,
		}
	}
	return sections
}

const SECTION_LIST_LIFETIME = int64(time.Hour / time.Second * 24)

func (c Client) Sections(ctx context.Context, course Course) ([]Section, error) {
	ctx, span := tracer.Start(ctx, "Sections")
	defer span.End()

	endpoint := course.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	res, err := c.Core.Http.R().
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

	anchors := htmlutil.GetAnchors(ctx, doc.Find(".course-content a.nav-link"))

	return sectionsFromAnchors(anchors), nil
}

type Resource htmlutil.Anchor

func resourcesFromAnchors(anchors []htmlutil.Anchor) []Resource {
	resources := make([]Resource, len(anchors))
	for i := 0; i < len(anchors); i++ {
		a := anchors[i]
		resources[i] = Resource{
			Name: a.Name,
			Href: a.Href,
		}
	}
	return resources
}

const RESOURCE_LIST_LIFETIME = int64(time.Minute * 15 / time.Second)

func (c Client) Resources(ctx context.Context, section Section) ([]Resource, error) {
	ctx, span := tracer.Start(ctx, "Resources")
	defer span.End()

	endpoint := section.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	res, err := c.Core.Http.R().
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

	anchors := htmlutil.GetAnchors(ctx, doc.Find("li.activity a"))

	return resourcesFromAnchors(anchors), nil
}

type Chapter htmlutil.Anchor

func chaptersFromAnchors(anchors []htmlutil.Anchor) []Chapter {
	chapters := make([]Chapter, len(anchors))
	for i := 0; i < len(anchors); i++ {
		a := anchors[i]
		if a == (htmlutil.Anchor{}) {
			continue
		}
		chapters[i] = Chapter{
			Name: a.Name,
			Href: a.Href,
		}
	}
	return chapters
}

const CHAPTER_LIST_LIFETIME = int64(time.Minute * 15 / time.Second)

func (c Client) Chapters(ctx context.Context, resource Resource) ([]Chapter, error) {
	ctx, span := tracer.Start(ctx, "Chapters")
	defer span.End()

	endpoint := resource.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	res, err := c.Core.Http.R().
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

	tableOfContents := htmlutil.GetAnchors(ctx, doc.Find("div.columnleft li a"))

	currentChapter := doc.Find("div.columnleft li").Text()

	anchors := append(tableOfContents, htmlutil.Anchor{
		Href: resource.Href,
		Name: currentChapter,
	})

	return chaptersFromAnchors(anchors), nil
}

func (c Client) ChapterContent(ctx context.Context, chapter Chapter) (string, error) {
	ctx, span := tracer.Start(ctx, "ChapterContent")
	defer span.End()

	endpoint := chapter.Href
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint),
	})

	res, err := c.Core.Http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return "", err
	}

	contents, err := doc.Find("div[role=main] div.box").Html()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return contents, nil
}
