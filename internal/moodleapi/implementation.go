package moodleapi

import (
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

const (
	report_db_query = "db.query"

	report_impl_user_login            = "impl.user-login"
	report_impl_scrape_user_courses   = "impl.scrape-user-courses"
	report_impl_courseid_parse        = "impl.course-id-parse"
	report_impl_parsetocdate          = "impl.parse-toc-date"
	report_impl_resource_id_null      = "impl.resource-id-null"
	report_warning_impl_lessonplan_not_found = "impl.lessonplan-not-found"
)

// MakeTx is a function that creates a db transaction
type MakeTx = func() (tx *db.Queries, discard, commit func())

// Implementation implements service.MoodleAPI
type Implementation struct {
	db        *db.Queries
	makeTx    MakeTx
	tel       telemetry.API
	adminUser string
	adminPass string
}

func NewImplementation(
	db *db.Queries,
	makeTx MakeTx,
	adminUser, adminPass string,
	opts ...ImplementationOption,
) Implementation {
	var cfg implementationCfg
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.tel == nil {
		cfg.tel = telemetry.SlogAPI{}
	}

	cfg.tel = telemetry.NewScopedAPI("moodleapi_impl", cfg.tel)

	return Implementation{db: db, makeTx: makeTx, tel: cfg.tel}
}

type ImplementationOption func(cfg *implementationCfg)

type implementationCfg struct {
	tel telemetry.API
}

func WithCustomTelemetryAPI(tel telemetry.API) ImplementationOption {
	return func(cfg *implementationCfg) {
		cfg.tel = tel
	}
}
