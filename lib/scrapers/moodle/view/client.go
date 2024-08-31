package view

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"vcassist-backend/lib/htmlutil"
	"vcassist-backend/lib/scrapers/moodle/core"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Client struct {
	Core *core.Client
}

func NewClient(ctx context.Context, coreClient *core.Client) (Client, error) {
	c := Client{
		Core: coreClient,
	}
	return c, nil
}

func parseIdFromUrl(link *url.URL) (int64, error) {
	str := link.Query().Get("id")
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return -1, err
	}
	return id, nil
}

type Course htmlutil.Anchor

func (c Course) Id() (int64, error) {
	return parseIdFromUrl(c.Url)
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
			Url:  a.Url,
		}
	}
	return courses
}

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

	anchors := htmlutil.GetAnchors(res.Request.RawRequest.URL, doc.Find("ul.unlist a"))

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
			Url:  a.Url,
		}
	}
	return sections
}

func (c Client) Sections(ctx context.Context, course Course) ([]Section, error) {
	ctx, span := tracer.Start(ctx, "Sections")
	defer span.End()

	endpoint := course.Url.String()
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

	anchors := htmlutil.GetAnchors(course.Url, doc.Find(".course-content a.nav-link"))

	return sectionsFromAnchors(anchors), nil
}

type ResourceType int

const (
	RESOURCE_GENERIC ResourceType = iota
	RESOURCE_BOOK
	RESOURCE_HTML_AREA
)

type Resource struct {
	Type ResourceType
	Name string
	Url  *url.URL
}

func resourcesFromAnchors(anchors []htmlutil.Anchor) []Resource {
	resources := make([]Resource, len(anchors))
	for i := 0; i < len(anchors); i++ {
		a := anchors[i]

		resourceType := RESOURCE_GENERIC
		if strings.HasPrefix(a.Url.Path, "/mod/book") {
			resourceType = RESOURCE_BOOK
		}

		resources[i] = Resource{
			Type: resourceType,
			Name: a.Name,
			Url:  a.Url,
		}
	}
	return resources
}

func (c Client) Resources(ctx context.Context, section Section) ([]Resource, error) {
	ctx, span := tracer.Start(ctx, "SectionContent")
	defer span.End()

	if section.Url == nil {
		return nil, fmt.Errorf("section url is nil")
	}
	endpoint := section.Url.String()
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

	infoHtml, err := doc.Find("div[data-for=sectioninfo]").Html()
	if err != nil {
		slog.WarnContext(ctx, "failed to serialize html for section info", "err", err)
	}

	anchors := htmlutil.GetAnchors(section.Url, doc.Find("li.activity a"))
	resources := resourcesFromAnchors(anchors)
	if infoHtml != "" {
		resources = append([]Resource{{
			Type: RESOURCE_HTML_AREA,
			Name: infoHtml,
		}}, resources...)
	}

	return resources, nil
}

type Chapter htmlutil.Anchor

func (c Chapter) Id() (int64, error) {
	return parseIdFromUrl(c.Url)
}

func chaptersFromAnchors(anchors []htmlutil.Anchor) []Chapter {
	chapters := make([]Chapter, len(anchors))
	for i := 0; i < len(anchors); i++ {
		a := anchors[i]
		if a == (htmlutil.Anchor{}) {
			continue
		}
		chapters[i] = Chapter{
			Name: a.Name,
			Url:  a.Url,
		}
	}
	return chapters
}

func (c Client) Chapters(ctx context.Context, resource Resource) ([]Chapter, error) {
	ctx, span := tracer.Start(ctx, "Chapters")
	defer span.End()

	endpoint := resource.Url.String()
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

	tableOfContents := htmlutil.GetAnchors(resource.Url, doc.Find("div.columnleft li a"))

	currentChapter := doc.Find("div.columnleft li").Text()

	anchors := append(tableOfContents, htmlutil.Anchor{
		Url:  resource.Url,
		Name: currentChapter,
	})

	return chaptersFromAnchors(anchors), nil
}

func (c Client) ChapterContent(ctx context.Context, chapter Chapter) (string, error) {
	ctx, span := tracer.Start(ctx, "ChapterContent")
	defer span.End()

	endpoint := chapter.Url
	span.SetAttributes(attribute.KeyValue{
		Key:   "url",
		Value: attribute.StringValue(endpoint.String()),
	})

	res, err := c.Core.Http.R().
		SetContext(ctx).
		Get(endpoint.String())
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
