package vcsmoodle

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/timezone"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// NOTE: much of this is legacy scraping code, it may be used later so it is kept here

func scrapeThroughWorkaroundLink(ctx context.Context, client view.Client, link string) (string, error) {
	ctx, span := tracer.Start(ctx, "scrapeThroughWorkaroundLink")
	defer span.End()

	span.SetAttributes(attribute.String("url", link))

	if !strings.Contains(link, client.Core.Http.BaseURL) ||
		!(strings.Contains(link, "/mod/url") || strings.Contains(link, "/mod/resource")) {
		return link, nil
	}

	res, err := client.Core.Http.R().
		SetContext(ctx).
		Get(link)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	proxied, ok := doc.Find("div.resourceworkaround a").Attr("href")
	if ok {
		return proxied, nil
	}
	proxied, ok = doc.Find("div.urlworkaround a").Attr("href")
	if ok {
		return proxied, nil
	}

	err = fmt.Errorf("failed to get find workaround target anchor for '%s'", link)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return "", err
}

type datedChapter struct {
	chapter view.Chapter
	dates   []time.Time
}

func scrapeChapters(ctx context.Context, client view.Client, resource view.Resource) (lessonPlan string, err error) {
	ctx, span := tracer.Start(ctx, "scrapeChapters")
	defer span.End()

	span.SetAttributes(attribute.String("name", resource.Name))
	span.SetAttributes(attribute.String("href", resource.Url.String()))

	chapters, err := client.Chapters(ctx, resource)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	datedChapters := []datedChapter{}

	now := timezone.Now()
	for _, c := range chapters {
		dates, err := parseTOCDate(ctx, c.Name)
		if err != nil {
			continue
		}
		for _, d := range dates {
			if d.Month() != now.Month() {
				continue
			}
			datedChapters = append(datedChapters, datedChapter{
				chapter: c,
				dates:   dates,
			})
			break
		}
	}

	// 3 for loops here because it is in order of priority
	// 1. exact match month and day
	// 2. exact match month and yesterday
	// 3. exact match month and tommorow
	for _, dc := range datedChapters {
		for _, d := range dc.dates {
			if d.Month() == now.Month() && d.Day() == now.Day() {
				return client.ChapterContent(ctx, dc.chapter)
			}
		}
	}
	for _, dc := range datedChapters {
		for _, d := range dc.dates {
			if d.Month() == now.Month() && d.Day() == now.Day()-1 {
				return client.ChapterContent(ctx, dc.chapter)
			}
		}
	}
	for _, dc := range datedChapters {
		for _, d := range dc.dates {
			if d.Month() == now.Month() && d.Day() == now.Day()+1 {
				return client.ChapterContent(ctx, dc.chapter)
			}
		}
	}

	err = fmt.Errorf("could not find lesson plan")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return "", err
}
