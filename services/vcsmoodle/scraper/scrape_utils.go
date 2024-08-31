package scraper

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/timezone"

	"github.com/PuerkitoBio/goquery"
)

// NOTE: much of this is legacy scraping code, it may be used later so it is kept here

func scrapeThroughWorkaroundLink(ctx context.Context, client view.Client, link string) (string, error) {
	if !strings.Contains(link, client.Core.Http.BaseURL) ||
		!(strings.Contains(link, "/mod/url") || strings.Contains(link, "/mod/resource")) {
		return link, nil
	}

	res, err := client.Core.Http.R().
		SetContext(ctx).
		Get(link)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
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
	return "", err
}

type datedChapter struct {
	chapter view.Chapter
	dates   []time.Time
}

func scrapeChapters(ctx context.Context, client view.Client, resource view.Resource) (lessonPlan string, err error) {
	chapters, err := client.Chapters(ctx, resource)
	if err != nil {
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
	return "", err
}
