package vcsmoodle

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/textutil"
	"vcassist-backend/lib/timezone"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func scrapeThroughWorkaroundLink(ctx context.Context, client view.Client, href string) (string, error) {
	ctx, span := tracer.Start(ctx, "scrapeThroughWorkaroundLink")
	defer span.End()

	span.SetAttributes(attribute.String("href", href))

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
		if textutil.MatchName(r.Name, zoomKeywords) {
			link, err := scrapeThroughWorkaroundLink(ctx, client, r.Href)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				continue
			}
			span.SetAttributes(attribute.String("zoom_link", link))
			return link, nil
		}
	}

	err = fmt.Errorf("could not find zoom link")
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
	span.SetAttributes(attribute.String("href", resource.Href))

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

func scrapeLessonPlanSection(ctx context.Context, client view.Client, section view.Section) (lessonPlan string, err error) {
	ctx, span := tracer.Start(ctx, "scrapeLessonPlanSection")
	defer span.End()

	span.SetAttributes(attribute.String("name", section.Name))
	span.SetAttributes(attribute.String("href", section.Href))

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
		if !strings.HasPrefix(strings.ToLower(r.Name), "quarter") {
			span.AddEvent("skip lesson plan resource", trace.WithAttributes(
				attribute.String("name", r.Name),
			))
			continue
		}

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
		Url:  course.Href,
	}

	span.SetAttributes(attribute.String("name", course.Name))

	sections, err := client.Sections(ctx, course)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, s := range sections {
			zoomlink, err := scrapeZoomLink(ctx, client, s)
			if err != nil {
				return
			}
			result.ZoomLink = zoomlink
			break
		}
	}()

	for _, s := range sections {
		if textutil.MatchName(s.Name, lessonPlanKeywords) {
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

	span.SetAttributes(attribute.Int("lesson_plan_length", len(result.GetLessonPlan())))
	span.SetAttributes(attribute.String("zoom_link", result.GetZoomLink()))

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

	var resultLock sync.Mutex
	var result []*vcsmoodlev1.Course

	courseChan := make(chan view.Course)

	for i := 0; i < 3; i++ {
		go func() {
			for c := range courseChan {
				course, err := scrapeCourseData(ctx, client, c)
				if err != nil {
					// the reason why this error is not being recorded
					// is because it has already been recorded in the
					// child span, so there is no need to repeat the error
					return
				}

				resultLock.Lock()
				result = append(result, course)
				resultLock.Unlock()

				slog.Info("scraped course", "current", len(result), "total", len(courses))
			}
		}()
	}

	for _, c := range courses {
		if textutil.MatchName(c.Name, blacklistCourseKeywords) {
			continue
		}
		courseChan <- c
	}

	return result, nil
}
