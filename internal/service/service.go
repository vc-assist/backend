package service

import (
	"crypto/rand"
	"encoding/base64"
	"vcassist-backend/internal/assert"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
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
	report_moodle_user_count            = "moodle.user-count"
	report_moodle_login                 = "moodle.login"
	report_moodle_scrape_user           = "moodle.scrape-user"
	report_moodle_query_user_course_ids = "moodle.query-user-course-ids"
	report_moodle_query_lesson_plans    = "moodle.query-lesson-plans"
	report_moodle_query_chapter_content = "moodle.query-chapter-content"

	report_ps_user_count  = "powerschool.user-count"
	report_ps_get_email   = "powerschool.get-email"
	report_ps_scrape_user = "powerschool.scrape-user"
	report_ps_query_data  = "powerschool.query-data"

	report_db_query              = "db.query"
	report_rand_token_generation = "rand.token-generation"
)

type MakeTx = func() (tx *db.Queries, discard, commit func())

type coreAPIs struct {
	db     *db.Queries
	makeTx MakeTx
	rand   RandomAPI
	tel    telemetry.API
}

// NewCoreAPIs initializes a collection of common APIs all services need to run.
func NewCoreAPIs(db *db.Queries, makeTx MakeTx, options ...CoreAPIsOption) coreAPIs {
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
		tel:    telemetry.SlogAPI{},
	}
	if cfg.rand != nil {
		apis.rand = cfg.rand
	}
	if cfg.tel != nil {
		apis.tel = cfg.tel
	}

	apis.tel = telemetry.NewScopedAPI("service", apis.tel)

	return apis
}

type coreAPIsConfig struct {
	rand RandomAPI
	tel  telemetry.API
}

type CoreAPIsOption func(cfg *coreAPIsConfig)

func WithCustomRandomAPI(rand RandomAPI) CoreAPIsOption {
	return func(cfg *coreAPIsConfig) {
		cfg.rand = rand
	}
}

func WithCustomTelemetryAPI(tel telemetry.API) CoreAPIsOption {
	return func(cfg *coreAPIsConfig) {
		cfg.tel = tel
	}
}

// MoodleService implements vcassist.moodle.v1.MoodleService
type MoodleService struct {
	coreAPIs

	api    MoodleAPI
	ctxKey any
}

// PowerschoolService implements vcassist.powerschool.v1.PowerschoolService
type PowerschoolService struct {
	coreAPIs

	api    PowerschoolAPI
	ctxKey any
}

// PublicService vcassist.public.v1.PublicService
type PublicService struct {
	coreAPIs
}

// NewPublicService creates a PublicService
func NewPublicService(coreAPIs coreAPIs) PublicService {
	return PublicService{coreAPIs: coreAPIs}
}

// NewMoodleService creates a MoodleService
func NewMoodleService(coreAPIs coreAPIs, api MoodleAPI, ctxKey any) MoodleService {
	assert.NotNil(api, "moodle scraping API implementation")
	assert.NotNil(ctxKey, "moodle ctx key")

	return MoodleService{
		coreAPIs: coreAPIs,
		api:      api,
		ctxKey:   ctxKey,
	}
}

// NewPowerschoolService creates a PowerschoolService
func NewPowerschoolService(coreAPIs coreAPIs, api PowerschoolAPI, ctxKey any) PowerschoolService {
	assert.NotNil(api, "powerschool scraping API implementation")
	assert.NotNil(ctxKey, "powerschool ctx key")

	return PowerschoolService{
		coreAPIs: coreAPIs,
		api:      api,
		ctxKey:   ctxKey,
	}
}
