package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"vcassist-backend/internal/assert"
	servicedb "vcassist-backend/internal/service/db"
)

// RandomAPI is an abstraction over any code that potentially generates random values.
// This makes mocking/simulation testing much easier.
//
// note: fault injection point
type RandomAPI interface {
	GenerateToken() (string, error)
}

type defaultRandomAPI struct{}

func (defaultRandomAPI) GenerateToken() (string, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(nonce), nil
}

const (
	REPORT_MOODLE_SCRAPING_LOGIN        = "moodle-scraping.login"
	REPORT_MOODLE_SCRAPING_REMOVE_USER  = "moodle-scraping.remove-user"
	REPORT_MOODLE_SCRAPING_SCRAPE_USER  = "moodle-scraping.scrape-user"
	REPORT_MOODLE_SCRAPING_SCRAPE_ALL   = "moodle-scraping.scrape-all"
	REPORT_MOODLE_QUERY_USER_COURSE_IDS = "moodle-query.query-user-course-ids"
	REPORT_MOODLE_QUERY_LESSON_PLANS    = "moodle-query.query-lesson-plans"
	REPORT_MOODLE_QUERY_CHAPTER_CONTENT = "moodle-query.query-chapter-content"

	REPORT_PS_SCRAPING_LOGIN       = "powerschool-scraping.login"
	REPORT_PS_SCRAPING_SCRAPE_ALL  = "powerschool-scraping.scrape-all"
	REPORT_PS_SCRAPING_SCRAPE_USER = "powerschool-scraping.scrape-user"
	REPORT_PS_QUERY_DATA           = "powerschool-query.query-data"

	REPORT_RAND_TOKEN_GENERATION = "rand.token-generation"
	REPORT_DB_QUERY              = "db.query"
)

// TelemetryAPI is an abstraction over logging/metrics.
// This allows for assertions and tests for working logging/metrics to exist.
//
// note: fault injection point
type TelemetryAPI interface {
	// this reports a component that has broken in a way that should be addressed
	ReportBroken(id string, params ...any)
}

func NamedParam(key string, param any) string {
	return fmt.Sprintf("%s: %v", key, param)
}

type defaultTelemetryAPI struct{}

func (defaultTelemetryAPI) formatParams(out *[]any, params []any) {
	for i, p := range params {
		*out = append(
			*out,
			fmt.Sprintf("params.%d", i),
			p,
		)
	}
}

func (t defaultTelemetryAPI) ReportBroken(id string, params ...any) {
	remainingPairs := []any{"id", id}
	t.formatParams(&remainingPairs, params)
	slog.Error("broken component (should address)", remainingPairs...)
}

type MakeTx = func() (tx *servicedb.Queries, discard, commit func())

type coreAPIs struct {
	db     *servicedb.Queries
	makeTx MakeTx
	rand   RandomAPI
	tel    TelemetryAPI
}

// NewCoreAPIs initializes a collection of common APIs all services need to run.
func NewCoreAPIs(
	db *servicedb.Queries,
	makeTx MakeTx,
	options ...CoreAPIsOption,
) coreAPIs {
	assert.NotNil(db, "db")
	assert.NotNil(makeTx, "makeTx")

	cfg := coreAPIsConfig{}
	for _, opt := range options {
		opt(&cfg)
	}

	apis := coreAPIs{
		db:     db,
		makeTx: makeTx,
		rand:   defaultRandomAPI{},
		tel:    defaultTelemetryAPI{},
	}
	if cfg.rand != nil {
		apis.rand = cfg.rand
	}
	if cfg.tel != nil {
		apis.tel = cfg.tel
	}

	return apis
}

type coreAPIsConfig struct {
	rand RandomAPI
	tel  TelemetryAPI
}

type CoreAPIsOption func(cfg *coreAPIsConfig)

func WithCustomRandomAPI(rand RandomAPI) CoreAPIsOption {
	return func(cfg *coreAPIsConfig) {
		cfg.rand = rand
	}
}

func WithCustomTelemetryAPI(tel TelemetryAPI) CoreAPIsOption {
	return func(cfg *coreAPIsConfig) {
		cfg.tel = tel
	}
}

// MoodleService implements vcassist.moodle.v1.MoodleService
type MoodleService struct {
	coreAPIs

	scraping MoodleScrapingAPI
	query    MoodleQueryAPI
	ctxKey   any
}

// PowerschoolService implements vcassist.powerschool.v1.PowerschoolService
type PowerschoolService struct {
	coreAPIs

	scraping PowerschoolScrapingAPI
	query    PowerschoolQueryAPI
	ctxKey   any
}

// PublicService vcassist.public.v1.PublicService
type PublicService struct {
	coreAPIs
}

func NewMoodleService(
	coreAPIs coreAPIs,
	scraping MoodleScrapingAPI,
	query MoodleQueryAPI,
	ctxKey any,
) MoodleService {
	assert.NotNil(scraping, "moodle scraping API implementation")
	assert.NotNil(query, "moodle query API implementation")
	assert.NotNil(ctxKey, "moodle ctx key")

	return MoodleService{
		coreAPIs: coreAPIs,
		scraping: scraping,
		query:    query,
		ctxKey:   ctxKey,
	}
}

func NewPowerschoolService(
	coreAPIs coreAPIs,
	scraping PowerschoolScrapingAPI,
	query PowerschoolQueryAPI,
	ctxKey any,
) PowerschoolService {
	assert.NotNil(scraping, "powerschool scraping API implementation")
	assert.NotNil(query, "powerschool query API implementation")
	assert.NotNil(ctxKey, "powerschool ctx key")

	return PowerschoolService{
		coreAPIs: coreAPIs,
		scraping: scraping,
		query:    query,
		ctxKey:   ctxKey,
	}
}
