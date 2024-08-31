package scraper

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/services/vcsmoodle/db"
)

type scraper struct {
	client view.Client
	qry    *db.Queries
	wg     *sync.WaitGroup
}

func (s scraper) scrapeChapter(ctx context.Context, chapter view.Chapter, courseId, sectionIdx, resourceIdx int64) {
	defer s.wg.Done()

	slog.DebugContext(ctx, "scraping chapter", "name", chapter.Name, "url", chapter.Url)

	content, err := s.client.ChapterContent(ctx, chapter)
	if err != nil || content == "" {
		slog.WarnContext(ctx, "failed to get chapter content", "err", err)
		return
	}

	id, err := chapter.Id()
	if err != nil {
		slog.WarnContext(ctx, "failed to parse chapter id", "id", id, "name", chapter.Name)
		return
	}

	err = s.qry.NoteChapter(ctx, db.NoteChapterParams{
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

func (s scraper) scrapeBook(ctx context.Context, resource view.Resource, courseId, sectionIdx, resourceIdx int64) {
	defer s.wg.Done()

	slog.DebugContext(ctx, "scraping book", "name", resource.Name, "url", resource.Url)

	chapterList, err := s.client.Chapters(ctx, resource)
	if err != nil {
		slog.WarnContext(ctx, "failed to get chapters", "err", err)
		return
	}

	for _, chapter := range chapterList {
		s.wg.Add(1)
		go s.scrapeChapter(ctx, chapter, courseId, sectionIdx, resourceIdx)
	}
}

func (s scraper) handleResource(ctx context.Context, resource view.Resource, resourceIdx, sectionIdx, courseId int64) {
	defer s.wg.Done()

	urlStr := ""
	if resource.Url != nil {
		urlStr = resource.Url.String()
	}

	params := db.NoteResourceParams{
		CourseID:       courseId,
		SectionIdx:     sectionIdx,
		Idx:            resourceIdx,
		DisplayContent: resource.Name,
		Url:            urlStr,
	}

	switch resource.Type {
	case view.RESOURCE_GENERIC:
		slog.DebugContext(ctx, "scraping through potential workaround link", "url", urlStr)
		realLink, err := scrapeThroughWorkaroundLink(ctx, s.client, urlStr)
		if err == nil {
			params.Url = realLink
		} else {
			slog.WarnContext(ctx, "failed to scrape through workaround link", "url", urlStr, "err", err)
		}

		slog.DebugContext(ctx, "noting generic resource", "idx", sectionIdx, "course_id", courseId, "name", resource.Name)
		params.Type = 0
	case view.RESOURCE_BOOK:
		slog.DebugContext(ctx, "noting book resource", "idx", sectionIdx, "course_id", courseId, "name", resource.Name)
		params.Type = 1

		go s.scrapeBook(ctx, resource, courseId, sectionIdx, resourceIdx)
	case view.RESOURCE_HTML_AREA:
		slog.DebugContext(ctx, "noting html area resource", "idx", sectionIdx, "course_id", courseId, "length", len(resource.Name))
		params.Type = 2
	default:
		slog.WarnContext(ctx, "unknown resource type", "type", resource.Type)
		return
	}

	err := s.qry.NoteResource(ctx, params)
	if err != nil {
		slog.WarnContext(ctx, "failed to note resource", "err", err)
	}
}

func (s scraper) scrapeSection(ctx context.Context, section view.Section, sectionIdx, courseId int64) error {
	defer s.wg.Done()

	slog.DebugContext(ctx, "scraping section", "idx", sectionIdx, "course_id", courseId)

	err := s.qry.NoteSection(ctx, db.NoteSectionParams{
		CourseID: courseId,
		Idx:      sectionIdx,
		Name:     section.Name,
	})
	if err != nil {
		return err
	}

	resourceList, err := s.client.Resources(ctx, section)
	if err != nil {
		return err
	}
	for i, resource := range resourceList {
		s.wg.Add(1)
		go s.handleResource(ctx, resource, int64(i), sectionIdx, courseId)
	}

	return nil
}

func (s scraper) scrapeCourse(ctx context.Context, course view.Course) {
	defer s.wg.Done()

	id, err := course.Id()
	if err != nil {
		slog.WarnContext(ctx, "failed to parse course id", "id", id, "name", course.Name)
		return
	}
	slog.DebugContext(ctx, "scraping course", "id", id, "name", course.Name)

	err = s.qry.NoteCourse(ctx, db.NoteCourseParams{
		ID:   id,
		Name: course.Name,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to note course", "err", err)
		return
	}

	sectionList, err := s.client.Sections(ctx, course)
	if err != nil {
		slog.WarnContext(ctx, "failed to get course sections", "err", err)
		return
	}
	for i, section := range sectionList {
		s.wg.Add(1)
		go s.scrapeSection(ctx, section, int64(i), id)
	}
}

func (s scraper) scrapeDashboard(ctx context.Context) {
	slog.DebugContext(ctx, "scraping dashboard")

	courseList, err := s.client.Courses(ctx)
	if err != nil {
		slog.WarnContext(ctx, "failed to get courses", "err", err)
		return
	}
	for _, course := range courseList {
		s.wg.Add(1)
		go s.scrapeCourse(ctx, course)
	}
}

func Scrape(ctx context.Context, out *sql.DB, client view.Client) {
	qry := db.New(out)
	tx, err := out.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create transaction", "err", err)
		return
	}
	defer tx.Commit()

	txqry := qry.WithTx(tx)

	err = txqry.DeleteAllChapters(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete all chapters", "err", err)
		return
	}
	err = txqry.DeleteAllResources(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete all resources", "err", err)
		return
	}
	err = txqry.DeleteAllSections(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete all sections", "err", err)
		return
	}
	err = txqry.DeleteAllCourses(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete all courses", "err", err)
		return
	}

	s := scraper{
		client: client,
		qry:    txqry,
		wg:     &sync.WaitGroup{},
	}
	s.scrapeDashboard(ctx)
	s.wg.Wait()
}
