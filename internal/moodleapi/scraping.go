package moodleapi

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
	"vcassist-backend/internal/db"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"

	"github.com/PuerkitoBio/goquery"
)

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

type scrapeAllReq struct {
	client view.Client
	db     *db.Queries
	wg     *sync.WaitGroup
}

func (r scrapeAllReq) scrapeChapter(ctx context.Context, chapter view.Chapter, courseId, sectionIdx, resourceIdx int64) {
	slog.DebugContext(ctx, "scraping chapter", "name", chapter.Name, "url", chapter.Url)

	content, err := r.client.ChapterContent(ctx, chapter)
	if err != nil || content == "" {
		slog.WarnContext(ctx, "failed to get chapter content", "url", chapter.Url, "err", err)
		return
	}

	id, err := chapter.Id()
	if err != nil {
		slog.WarnContext(ctx, "failed to parse chapter id", "id", id, "name", chapter.Name)
		return
	}

	err = r.db.AddMoodleChapter(ctx, db.AddMoodleChapterParams{
		CourseID:    courseId,
		SectionIdx:  sectionIdx,
		ResourceIdx: resourceIdx,
		ID:          id,
		Name:        chapter.Name,
		ContentHtml: content,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to note chapter", "err", err)
	}
}

func (r scrapeAllReq) scrapeBook(ctx context.Context, resource view.Resource, courseId, sectionIdx, resourceIdx int64) {
	slog.DebugContext(ctx, "scraping book", "name", resource.Name, "url", resource.Url)

	chapterList, err := r.client.Chapters(ctx, resource)
	if err != nil {
		slog.WarnContext(ctx, "failed to get chapters", "err", err)
		return
	}

	for _, chapter := range chapterList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeChapter(ctx, chapter, courseId, sectionIdx, resourceIdx)
		}()
	}
}

func (r scrapeAllReq) handleResource(ctx context.Context, resource view.Resource, resourceIdx, sectionIdx, courseId int64) {
	var id int64
	var urlStr string
	var err error
	if resource.Url != nil {
		urlStr = resource.Url.String()
		idStr := resource.Url.Query().Get("id")
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	params := db.AddMoodleResourceParams{
		CourseID:   courseId,
		SectionIdx: sectionIdx,
		ID: sql.NullInt64{
			Int64: id,
			Valid: err == nil,
		},
		Idx:            resourceIdx,
		DisplayContent: resource.Name,
		Url:            urlStr,
	}

	switch resource.Type {
	case view.RESOURCE_GENERIC:
		realLink, err := scrapeThroughWorkaroundLink(ctx, r.client, urlStr)
		if err == nil {
			slog.DebugContext(ctx, "scraped through workaround link", "workaround_url", urlStr, "real_url", realLink)
			params.Url = realLink
		} else {
			slog.WarnContext(ctx, "failed to scrape through workaround link", "url", urlStr, "err", err)
		}

		slog.DebugContext(ctx, "adding generic resource", "idx", sectionIdx, "course_id", courseId, "name", resource.Name)
		params.Type = int64(db.MOODLE_RESOURCE_GENERIC)
	case view.RESOURCE_FILE:
		realLink, err := scrapeThroughWorkaroundLink(ctx, r.client, urlStr)
		if err == nil {
			slog.DebugContext(ctx, "scraped through workaround link", "workaround_url", urlStr, "real_url", realLink)
			params.Url = realLink
		} else {
			slog.WarnContext(ctx, "failed to scrape through workaround link", "url", urlStr, "err", err)
		}

		slog.DebugContext(ctx, "adding file resource", "idx", sectionIdx, "course_id", courseId, "name", resource.Name)
		params.Type = int64(db.MOODLE_RESOURCE_FILE)
	case view.RESOURCE_BOOK:
		slog.DebugContext(ctx, "adding book resource", "idx", sectionIdx, "course_id", courseId, "name", resource.Name)
		params.Type = int64(db.MOODLE_RESOURCE_BOOK)

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeBook(ctx, resource, courseId, sectionIdx, resourceIdx)
		}()
	case view.RESOURCE_HTML_AREA:
		slog.DebugContext(ctx, "adding html area resource", "idx", sectionIdx, "course_id", courseId, "length", len(resource.Name))
		params.Type = int64(db.MOODLE_RESOURCE_HTML_AREA)
	default:
		slog.WarnContext(ctx, "unknown resource type", "type", resource.Type)
		return
	}

	err = r.db.AddMoodleResource(ctx, params)
	if err != nil {
		slog.WarnContext(ctx, "failed to note resource", "err", err)
	}
}

func (r scrapeAllReq) scrapeSection(ctx context.Context, section view.Section, sectionIdx, courseId int64) error {
	slog.DebugContext(ctx, "scraping section", "idx", sectionIdx, "course_id", courseId)

	err := r.db.AddMoodleSection(ctx, db.AddMoodleSectionParams{
		CourseID: courseId,
		Idx:      sectionIdx,
		Name:     section.Name,
	})
	if err != nil {
		return err
	}

	resourceList, err := r.client.Resources(ctx, section)
	if err != nil {
		return err
	}
	for i, resource := range resourceList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.handleResource(ctx, resource, int64(i), sectionIdx, courseId)
		}()
	}

	return nil
}

func (r scrapeAllReq) scrapeCourse(ctx context.Context, course view.Course) {
	id, err := course.Id()
	if err != nil {
		slog.WarnContext(ctx, "failed to parse course id", "id", id, "name", course.Name)
		return
	}
	slog.DebugContext(ctx, "scraping course", "id", id, "name", course.Name)

	err = r.db.AddMoodleCourse(ctx, db.AddMoodleCourseParams{
		ID:   id,
		Name: course.Name,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to note course", "err", err)
		return
	}

	sectionList, err := r.client.Sections(ctx, course)
	if err != nil {
		slog.WarnContext(ctx, "failed to get course sections", "err", err)
		return
	}
	for i, section := range sectionList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeSection(ctx, section, int64(i), id)
		}()
	}
}

func (r scrapeAllReq) scrapeDashboard(ctx context.Context) {
	slog.DebugContext(ctx, "scraping dashboard")

	courseList, err := r.client.Courses(ctx)
	if err != nil {
		slog.WarnContext(ctx, "failed to get courses", "err", err)
		return
	}
	for _, course := range courseList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeCourse(ctx, course)
		}()
	}
}

func createMoodleClient(username, password string) (view.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: "https://learn.vcs.net",
	})
	if err != nil {
		return view.Client{}, err
	}
	err = coreClient.LoginUsernamePassword(ctx, username, password)
	if err != nil {
		return view.Client{}, err
	}
	client, err := view.NewClient(ctx, coreClient)
	if err != nil {
		return view.Client{}, err
	}

	return client, nil
}

func (impl Implementation) scrapeAllMoodle(ctx context.Context) error {
	client, err := createMoodleClient(impl.adminUser, impl.adminPass)
	if err != nil {
		return err
	}

	tx, discard, commit := impl.makeTx()
	defer discard()

	err = tx.DeleteAllMoodleChapters(ctx)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleChapters")
		return err
	}
	err = tx.DeleteAllMoodleResources(ctx)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleResources")
		return err
	}
	err = tx.DeleteAllMoodleSections(ctx)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleSections")
		return err
	}
	err = tx.DeleteAllMoodleCourses(ctx)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleCourses")
		return err
	}

	r := scrapeAllReq{
		client: client,
		db:     tx,
		wg:     &sync.WaitGroup{},
	}
	r.scrapeDashboard(ctx)
	r.wg.Wait()

	commit()
	return nil
}

func (impl Implementation) scrapeAllMoodleUsers(ctx context.Context) error {
	tx, discard, commit := impl.makeTx()
	defer discard()

	accounts, err := tx.GetAllMoodleAccounts(ctx)
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	for _, acc := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			impl.scrapeUser(ctx, acc.ID, acc.Username, acc.Password)
		}()
	}
	wg.Wait()

	commit()

	return nil
}

// ScrapeAll uses the admin account to scrape all the courses, but also
// updates all user specific information like moodle_user_course
func (impl Implementation) ScrapeAll(ctx context.Context) error {
	err := impl.scrapeAllMoodle(ctx)
	if err != nil {
		return err
	}
	return impl.scrapeAllMoodleUsers(ctx)
}

func (impl Implementation) scrapeUser(ctx context.Context, accountId int64, username, password string) error {
	client, err := createMoodleClient(username, password)
	if err != nil {
		impl.tel.ReportBroken(report_impl_user_login, err, username, password)
		return err
	}
	courses, err := client.Courses(ctx)
	if err != nil {
		impl.tel.ReportBroken(report_impl_scrape_user_courses, err, username, password)
		return err
	}

	tx, discard, commit := impl.makeTx()
	defer discard()

	for _, c := range courses {
		courseId, err := c.Id()
		if err != nil {
			impl.tel.ReportBroken(report_impl_courseid_parse, err, c.Url)
			continue
		}
		err = tx.AddMoodleUserCourse(ctx, db.AddMoodleUserCourseParams{
			AccountID: accountId,
			CourseID:  courseId,
		})
		if err != nil {
			impl.tel.ReportBroken(report_db_query, err, "AddUserCourse", accountId, courseId)
			return err
		}
	}

	commit()
	return nil
}

// ScrapeUser implements the interface method.
func (impl Implementation) ScrapeUser(ctx context.Context, accountId int64) error {
	user, err := impl.db.GetMoodleAccountFromId(ctx, accountId)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "GetMoodleAccountFromUsername")
		return err
	}
	err = impl.scrapeUser(ctx, accountId, user.Username, user.Password)
	if err != nil {
		return err
	}
	return nil
}
