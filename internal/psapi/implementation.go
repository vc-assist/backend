package psapi

import (
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

// WeightsAPI describes all the methods
type WeightsAPI interface {
	// GetWeights returns the weight values for a course and its categories
	GetWeights(courseName string, categories []string) []float32
}

// Implementation implements service.PowerschoolAPI
type Implementation struct {
	db      *db.Queries
	tel     telemetry.API
	weights WeightsAPI
}

func NewImplementation(db *db.Queries, weights WeightsAPI) Implementation {
	return Implementation{db: db, weights: weights}
}

const (
	report_impl_get_email = "impl.get-email"

	report_scraper_new_client  = "scraper.new-client"
	report_scraper_login_oauth = "scraper.login-oauth"
	report_scraper_ps_request  = "scraper.powerschool-request"
	report_scraper_postprocess = "scraper.post-process"

	report_db_query = "db.query"

	report_pb_unmarshal = "pb.unmarshal"
	report_pb_marshal   = "pb.marshal"
)
