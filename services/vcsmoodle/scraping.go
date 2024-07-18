package vcsmoodle

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/timezone"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func scrapeThroughWorkaroundLink(ctx context.Context, client view.Client, href string) (string, error) {
	ctx, span := tracer.Start(ctx, "scrapeThroughWorkaroundLink")
	defer span.End()

	if !strings.Contains(href, client.Core.Http.BaseURL) {
		return href, nil
	}

	res, err := client.Core.Http.R().
		SetContext(ctx).
		Get(href)
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

	proxied, ok := doc.Find("div.urlworkaround a").Attr("href")
	if !ok {
		err := fmt.Errorf("failed to get div.urlworkaround href")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return proxied, nil
}

func scrapeZoomLink(ctx context.Context, client view.Client, section view.Section) (string, error) {
	ctx, span := tracer.Start(ctx, "scrapeZoomLink")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "section_name",
			Value: attribute.StringValue(section.Name),
		},
		attribute.KeyValue{
			Key:   "section_url",
			Value: attribute.StringValue(section.Href),
		},
	)

	resources, err := client.Resources(ctx, section)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	for _, r := range resources {
		if MatchName(r.Name, zoomKeywords) {
			link, err := scrapeThroughWorkaroundLink(ctx, client, r.Href)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				continue
			}
			return link, nil
		}
	}

	err = fmt.Errorf("could not find zoom link")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return "", err
}

func scrapeChapters(ctx context.Context, client view.Client, resource view.Resource) (lessonPlan string, err error) {
	ctx, span := tracer.Start(ctx, "scrapeChapters")
	defer span.End()

	chapters, err := client.Chapters(ctx, resource)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	now := timezone.Now()
	for _, c := range chapters {
		dates, err := parseTOCDate(ctx, c.Name)
		if err != nil {
			continue
		}
		for _, d := range dates {
			if d.Month() == now.Month() && d.Day() == now.Day() {
				return client.ChapterContent(ctx, c)
			}
		}
	}

	err = fmt.Errorf("could not find lesson plan")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return "", err
}

func scrapeLessonPlanSection(ctx context.Context, client view.Client, section view.Section) (lessonPlan string, err error) {
	ctx, span := tracer.Start(ctx, "scrapeQuarter")
	defer span.End()

	resources, err := client.Resources(ctx, section)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// the following heuristic for selecting quarters is dumb but reliable,
	// we don't care too much about speed so this is okay

	slices.SortFunc(resources, func(a, b view.Resource) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	for _, r := range resources {
		lessonPlan, err := scrapeChapters(ctx, client, r)
		if err != nil {
			continue
		}
		return lessonPlan, nil
	}

	err = fmt.Errorf("could not find today's lesson plan")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return "", err
}

func scrapeCourseData(ctx context.Context, client view.Client, course view.Course) (*vcsmoodlev1.Course, error) {
	ctx, span := tracer.Start(ctx, "scrapeCourseData")
	defer span.End()

	result := &vcsmoodlev1.Course{
		Name: course.Name,
	}

	sections, err := client.Sections(ctx, course)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	wg := sync.WaitGroup{}
	for _, s := range sections {
		if MatchName(s.Name, zoomKeywords) {
			wg.Add(1)
			go func() {
				defer wg.Done()

				zoomlink, err := scrapeZoomLink(ctx, client, s)
				if err != nil {
					return
				}
				result.ZoomLink = zoomlink
			}()
		}

		if MatchName(s.Name, lessonPlanKeywords) {
			wg.Add(1)
			go func() {
				defer wg.Done()

				lessonplan, err := scrapeLessonPlanSection(ctx, client, s)
				if err != nil {
					return
				}
				result.LessonPlan = lessonplan
			}()
		}
	}
	wg.Wait()

	return result, nil
}

func scrapeCourses(ctx context.Context, client view.Client) ([]*vcsmoodlev1.Course, error) {
	ctx, span := tracer.Start(ctx, "scrapeCourses")
	defer span.End()

	courses, err := client.Courses(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var result []*vcsmoodlev1.Course

	wg := sync.WaitGroup{}
	for _, c := range courses {
		if MatchName(c.Name, blacklistCourseKeywords) {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			course, err := scrapeCourseData(ctx, client, c)
			if err != nil {
				// the reason why this error is not being recorded
				// is because it has already been recorded in the
				// child span, so there is no need to repeat the error
				return
			}
			result = append(result, course)
		}()
	}
	wg.Wait()

	return result, nil
}
