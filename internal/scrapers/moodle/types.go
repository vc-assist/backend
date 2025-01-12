package moodle

import (
	"net/url"
	"strconv"
	"strings"
	"vcassist-backend/internal/components/assert"
	"vcassist-backend/internal/components/chrono"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"
	"vcassist-backend/pkg/htmlutil"
)

const (
	report_db_query = "db.query"
)

// Scraper scrapes moodle.
type Scraper struct {
	db        *db.Queries
	makeTx    db.MakeTx
	tel       telemetry.API
	time      chrono.TimeAPI
	adminUser string
	adminPass string
}

func NewScraper(
	db *db.Queries,
	makeTx db.MakeTx,
	time chrono.TimeAPI,
	tel telemetry.API,
	adminUser, adminPass string,
) Scraper {
	assert.NotNil(db)
	assert.NotNil(makeTx)
	assert.NotNil(time)
	assert.NotNil(tel)
	assert.NotEmptyStr(adminUser)
	assert.NotEmptyStr(adminPass)

	tel = telemetry.NewScopedAPI("moodle_scraper", tel)

	return Scraper{
		db:        db,
		makeTx:    makeTx,
		time:      time,
		tel:       tel,
		adminUser: adminUser,
		adminPass: adminPass,
	}
}

type ResourceType int

const (
	RESOURCE_GENERIC ResourceType = iota
	RESOURCE_FILE
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
		switch {
		case strings.HasPrefix(a.Url.Path, "/mod/resource"):
			resourceType = RESOURCE_FILE
		case strings.HasPrefix(a.Url.Path, "/mod/book"):
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

func parseIdFromUrl(link *url.URL, key string) (int64, error) {
	str := link.Query().Get(key)
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return -1, err
	}
	return id, nil
}

type Course htmlutil.Anchor

func (c Course) Id() (int64, error) {
	return parseIdFromUrl(c.Url, "id")
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

type Chapter htmlutil.Anchor

func (c Chapter) Id() (int64, error) {
	return parseIdFromUrl(c.Url, "chapterid")
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
