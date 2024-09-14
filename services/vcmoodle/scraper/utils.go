package scraper

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"vcassist-backend/lib/scrapers/moodle/view"

	"github.com/PuerkitoBio/goquery"
)

func ScrapeThroughWorkaroundLink(ctx context.Context, client view.Client, link string) (string, error) {
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
