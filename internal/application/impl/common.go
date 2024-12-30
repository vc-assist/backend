package impl

import (
	"vcassist-backend/internal/components/db"

	"github.com/go-resty/resty/v2"
)

const (
	report_db_query               = "db.query"
	report_pb_unmarshal           = "pb.unmarshal"
	report_pb_marshal             = "pb.marshal"
	report_snapshot_get_snapshots = "snapshot.get-snapshots"
	report_snapshot_make_snapshot = "snapshot.make-snapshot"

	report_moodle_user_login          = "moodle.user-login"
	report_moodle_scrape_user_courses = "moodle.scrape-user-courses"
	report_moodle_courseid_parse      = "moodle.course-id-parse"
	report_moodle_parse_toc_date      = "moodle.parse-toc-date"
	report_moodle_query_lesson_plans  = "moodle.query-lesson-plans"

	report_ps_get_email     = "powerschool.get-email"
	report_ps_new_client    = "powerschool.new-client"
	report_ps_login_oauth   = "powerschool.login-oauth"
	report_ps_request       = "powerschool.powerschool-request"
	report_ps_response_data = "powerschool.powerschool-response-data"
	report_ps_postprocess   = "powerschool.post-process"

	report_weights_implicit_category_resolution = "weights.implicit-category-resolution"
	report_weights_find_course                  = "weights.find-course"
)

var defaultClient = resty.New()

// MakeTx is a function that creates a db transaction
type MakeTx = func() (tx *db.Queries, discard, commit func())
