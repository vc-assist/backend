// scraper.go contains the extra logic for scraping valley's information moodle outside of client.go

package moodle

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"

	"github.com/PuerkitoBio/goquery"
)

const (
	report_postprocess_scrape_user             = "postprocess.scrape-user"
	report_postprocess_scrape_chapter          = "postprocess.scrape-chapter"
	report_postprocess_scrape_book             = "postprocess.scrape-book"
	report_postprocess_scrape_section          = "postprocess.scrape-section"
	report_postprocess_scrape_course           = "postprocess.scrape-course"
	report_postprocess_scrape_dashboard        = "postprocess.scrape-dashboard"
	report_postprocess_resolve_workaround_link = "postprocess.resolve-workaround-link"
	report_postprocess_handle_resource         = "postprocess.handle-resource"
)

func (s Scraper) createMoodleClient(username, password string) (*client, error) {
	moodleClient, err := newClient("https://learn.vcs.net", s.tel)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = moodleClient.LoginUsernamePassword(ctx, username, password)
	if err != nil {
		return nil, err
	}

	return moodleClient, nil
}

func (s Scraper) scrapeAllMoodle(ctx context.Context) error {
	client, err := s.createMoodleClient(s.adminUser, s.adminPass)
	if err != nil {
		return err
	}

	tx, discard, commit, err := s.makeTx()
	if err != nil {
		s.tel.ReportBroken(
			report_db_query,
			fmt.Errorf("make tx: %w", err),
		)
		return err
	}
	defer discard()

	err = tx.DeleteAllMoodleChapters(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleChapters")
		return err
	}
	err = tx.DeleteAllMoodleResources(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleResources")
		return err
	}
	err = tx.DeleteAllMoodleSections(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleSections")
		return err
	}
	err = tx.DeleteAllMoodleCourses(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "DeleteAllMoodleCourses")
		return err
	}

	r := scrapeReq{
		tel:    s.tel,
		tx:     tx,
		client: client,
		wg:     &sync.WaitGroup{},
	}
	r.Start(ctx)
	r.wg.Wait()

	return commit()
}

func (s Scraper) scrapeUser(ctx context.Context, accountId int64, username, password string) error {
	client, err := s.createMoodleClient(username, password)
	if err != nil {
		s.tel.ReportBroken(
			report_postprocess_scrape_user,
			fmt.Errorf("login: %w", err),
			username,
			password,
		)
		return err
	}
	courses, err := client.Courses(ctx)
	if err != nil {
		s.tel.ReportBroken(report_postprocess_scrape_user, err, username, password)
		return err
	}

	tx, discard, commit, err := s.makeTx()
	if err != nil {
		s.tel.ReportBroken(
			report_db_query,
			fmt.Errorf("make tx: %w", err),
		)
		return err
	}
	defer discard()

	for _, c := range courses {
		courseId, err := c.Id()
		if err != nil {
			s.tel.ReportBroken(
				report_postprocess_scrape_user,
				fmt.Errorf("parse course id: %w", err),
				c.Url,
			)
			continue
		}
		err = tx.AddMoodleUserCourse(ctx, db.AddMoodleUserCourseParams{
			AccountID: accountId,
			CourseID:  courseId,
		})
		if err != nil {
			s.tel.ReportBroken(
				report_db_query,
				err,
				"AddUserCourse",
				accountId,
				courseId,
			)
			return err
		}
	}

	commit()
	return nil
}

func (s Scraper) scrapeAllMoodleUsers(ctx context.Context) error {
	accounts, err := s.db.GetAllMoodleAccounts(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetAllMoodleAccounts")
		return err
	}
	wg := sync.WaitGroup{}
	for _, acc := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scrapeUser(ctx, acc.ID, acc.Username, acc.Password)
		}()
	}
	wg.Wait()

	return nil
}

// ScrapeUser implements the interface method.
func (s Scraper) ScrapeUser(ctx context.Context, accountId int64) error {
	user, err := s.db.GetMoodleAccountFromId(ctx, accountId)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetMoodleAccountFromId")
		return err
	}
	err = s.scrapeUser(ctx, accountId, user.Username, user.Password)
	if err != nil {
		return err
	}
	return nil
}

// ScrapeAll uses the admin account to scrape all the courses, but also
// updates all user specific information like moodle_user_course
func (s Scraper) ScrapeAll(ctx context.Context) error {
	err := s.scrapeAllMoodle(ctx)
	if err != nil {
		return err
	}
	return s.scrapeAllMoodleUsers(ctx)
}

type scrapeReq struct {
	tel    telemetry.API
	tx     *db.Queries
	client *client
	wg     *sync.WaitGroup
}

func (r scrapeReq) resolveWorkaroundLink(ctx context.Context, client *client, link string) (string, error) {
	if !strings.Contains(link, client.Http.BaseURL) ||
		!(strings.Contains(link, "/mod/url") || strings.Contains(link, "/mod/resource")) {
		r.tel.ReportDebug("skipped workaround link resolution", link)
		return link, nil
	}

	res, err := client.Http.R().
		SetContext(ctx).
		Get(link)
	if err != nil {
		r.tel.ReportBroken(report_postprocess_resolve_workaround_link, fmt.Errorf("fetch: %w", err))
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		r.tel.ReportBroken(report_postprocess_resolve_workaround_link, fmt.Errorf("parse: %w", err))
		return "", err
	}

	proxied, ok := doc.Find("div.resourceworkaround a").Attr("href")
	if ok {
		r.tel.ReportDebug("resolved workaround link", proxied)
		return proxied, nil
	}
	proxied, ok = doc.Find("div.urlworkaround a").Attr("href")
	if ok {
		r.tel.ReportDebug("resolved workaround link", proxied)
		return proxied, nil
	}

	err = fmt.Errorf("resolve workaround link: could not find target anchor for '%s'", link)
	r.tel.ReportWarning(report_postprocess_resolve_workaround_link, err, link)

	return "", err
}

func (r scrapeReq) handleResource(ctx context.Context, resource Resource, resourceIdx, sectionIdx, courseId int64) {
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
	case RESOURCE_GENERIC:
		realLink, err := r.resolveWorkaroundLink(ctx, r.client, urlStr)
		if err == nil {
			params.Url = realLink
		}

		r.tel.ReportDebug(
			"generic resource",
			courseId,
			sectionIdx,
			resource.Name,
		)
		params.Type = int64(db.MOODLE_RESOURCE_GENERIC)
	case RESOURCE_FILE:
		realLink, err := r.resolveWorkaroundLink(ctx, r.client, urlStr)
		if err == nil {
			params.Url = realLink
		}

		r.tel.ReportDebug(
			"file resource",
			courseId,
			sectionIdx,
			resource.Name,
		)
		params.Type = int64(db.MOODLE_RESOURCE_FILE)
	case RESOURCE_BOOK:
		r.tel.ReportDebug(
			"book resource",
			courseId,
			sectionIdx,
			resource.Name,
		)
		params.Type = int64(db.MOODLE_RESOURCE_BOOK)

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeBook(ctx, resource, courseId, sectionIdx, resourceIdx)
		}()
	case RESOURCE_HTML_AREA:
		r.tel.ReportDebug(
			"html area resource",
			courseId,
			sectionIdx,
			fmt.Sprintf("name_len=%d", len(resource.Name)),
		)
		params.Type = int64(db.MOODLE_RESOURCE_HTML_AREA)
	default:
		r.tel.ReportBroken(
			report_postprocess_handle_resource,
			fmt.Errorf("unknown resource type"),
			resource.Type,
		)
		return
	}

	err = r.tx.AddMoodleResource(ctx, params)
	if err != nil {
		r.tel.ReportBroken(
			report_db_query,
			err,
			"AddMoodleResource",
		)
	}
}

func (r scrapeReq) scrapeChapter(ctx context.Context, chapter Chapter, courseId, sectionIdx, resourceIdx int64) {
	r.tel.ReportDebug(report_postprocess_scrape_chapter, chapter.Name, chapter.Url)

	content, err := r.client.ChapterContent(ctx, chapter)
	if err != nil || content == "" {
		r.tel.ReportBroken(
			report_postprocess_scrape_chapter,
			fmt.Errorf("get content: %w", err),
			chapter.Url,
		)
		return
	}

	id, err := chapter.Id()
	if err != nil {
		r.tel.ReportBroken(
			report_postprocess_scrape_chapter,
			fmt.Errorf("parse id: %w", err),
			chapter.Url,
		)
		return
	}

	err = r.tx.AddMoodleChapter(ctx, db.AddMoodleChapterParams{
		CourseID:    courseId,
		SectionIdx:  sectionIdx,
		ResourceIdx: resourceIdx,
		ID:          id,
		Name:        chapter.Name,
		ContentHtml: content,
	})
	if err != nil {
		r.tel.ReportBroken(
			report_db_query,
			err,
			"AddMoodleChapter",
		)
	}
}

func (r scrapeReq) scrapeBook(ctx context.Context, resource Resource, courseId, sectionIdx, resourceIdx int64) {
	r.tel.ReportDebug("scraping book", resource.Name, resource.Url)

	chapterList, err := r.client.Chapters(ctx, resource)
	if err != nil {
		r.tel.ReportBroken(report_postprocess_scrape_book, err)
		return
	}
	if len(chapterList) == 0 {
		r.tel.ReportWarning(
			report_client_get_chapters,
			fmt.Errorf("get chapters: no chapters found in '%s' (%s)", resource.Name, resource.Url),
		)
	}

	for _, chapter := range chapterList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeChapter(ctx, chapter, courseId, sectionIdx, resourceIdx)
		}()
	}
}

func (r scrapeReq) scrapeSection(ctx context.Context, section Section, sectionIdx, courseId int64) error {
	r.tel.ReportDebug("scraping section", section.Name, section.Url)

	err := r.tx.AddMoodleSection(ctx, db.AddMoodleSectionParams{
		CourseID: courseId,
		Idx:      sectionIdx,
		Name:     section.Name,
	})
	if err != nil {
		r.tel.ReportBroken(report_db_query, "AddMoodleSection", err)
		return err
	}

	resourceList, err := r.client.Resources(ctx, section)
	if err != nil {
		r.tel.ReportBroken(report_postprocess_scrape_section, err)
		return err
	}
	if len(resourceList) == 0 {
		r.tel.ReportWarning(
			report_client_get_resources,
			fmt.Errorf("get resources: no resources found in '%s' (%s)", section.Name, section.Url),
		)
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

func (r scrapeReq) scrapeCourse(ctx context.Context, course Course) {
	id, err := course.Id()
	if err != nil {
		r.tel.ReportBroken(report_postprocess_scrape_course, course.Url, course.Name)
		return
	}

	r.tel.ReportDebug("scraping course", id, course.Name)

	err = r.tx.AddMoodleCourse(ctx, db.AddMoodleCourseParams{
		ID:   id,
		Name: course.Name,
	})
	if err != nil {
		r.tel.ReportBroken(report_db_query, err, "AddMoodleCourse")
		return
	}

	sectionList, err := r.client.Sections(ctx, course)
	if err != nil {
		r.tel.ReportBroken(report_postprocess_scrape_course, err)
		return
	}
	if len(sectionList) == 0 {
		r.tel.ReportWarning(
			report_client_get_sections,
			fmt.Errorf("get sections: no sections found in '%s' (%d)", course.Name, id),
		)
	}

	for i, section := range sectionList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeSection(ctx, section, int64(i), id)
		}()
	}
}

func (r scrapeReq) Start(ctx context.Context) {
	r.tel.ReportDebug("scraping dashboard")

	courseList, err := r.client.Courses(ctx)
	if err != nil {
		r.tel.ReportBroken(report_postprocess_scrape_dashboard, err)
		return
	}
	if len(courseList) == 0 {
		r.tel.ReportBroken(
			report_client_get_courses,
			fmt.Errorf("get courses: no courses found"),
		)
	}

	for _, course := range courseList {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.scrapeCourse(ctx, course)
		}()
	}
}
