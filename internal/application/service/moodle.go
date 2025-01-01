package service

import (
	"context"
	"fmt"
	moodlev1 "vcassist-backend/api/vcassist/moodle/v1"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"

	"connectrpc.com/connect"
)

const (
	report_moodle_user_count            = "moodle.user-count"
	report_moodle_login                 = "moodle.login"
	report_moodle_scrape_user           = "moodle.scrape-user"
	report_moodle_query_user_course_ids = "moodle.query-user-course-ids"
	report_moodle_query_lesson_plans    = "moodle.query-lesson-plans"
	report_moodle_query_chapter_content = "moodle.query-chapter-content"
)

// MoodleAPI describes all the moodle scraping methods (no scraping logic is in the service so the
// service's logic can be tested individually)
//
// note: there should not be any cron jobs running in here
type MoodleAPI interface {
	// ScrapeUser updates the "user" information for a specific user.
	ScrapeUser(ctx context.Context, accountId int64) error

	// QueryLessonPlans transforms the cached moodle data into a *moodlev1.LessonPlansResponse
	// given a list of course ids.
	QueryLessonPlans(ctx context.Context, courseIds []int64) (*moodlev1.LessonPlansResponse, error)

	// QueryChapterContent returns the html content for a given chapter id.
	QueryChapterContent(ctx context.Context, chapterId int64) (string, error)

	// QueryUserCourseIds returns the ids for the moodle courses available to a given user's account.
	QueryUserCourseIds(ctx context.Context, accountId int64) ([]int64, error)
}

var unauthorizedError = fmt.Errorf("unauthorized")

// Refresh implements the protobuf method.
func (s MoodleService) Refresh(ctx context.Context, req *connect.Request[moodlev1.RefreshRequest]) (*connect.Response[moodlev1.RefreshResponse], error) {
	acc, ok := ctx.Value(s.ctxKey).(db.GetMoodleAccountFromTokenRow)
	if !ok {
		return nil, unauthorizedError
	}

	err := s.api.ScrapeUser(ctx, acc.ID)
	if err != nil {
		s.tel.ReportBroken(report_moodle_scrape_user, err, acc.ID)
		return nil, err
	}

	return &connect.Response[moodlev1.RefreshResponse]{
		Msg: &moodlev1.RefreshResponse{},
	}, nil
}

// LessonPlans implements the protobuf method.
func (s MoodleService) LessonPlans(ctx context.Context, req *connect.Request[moodlev1.LessonPlansRequest]) (*connect.Response[moodlev1.LessonPlansResponse], error) {
	acc, ok := ctx.Value(s.ctxKey).(db.GetMoodleAccountFromTokenRow)
	if !ok {
		return nil, unauthorizedError
	}

	courseIds, err := s.api.QueryUserCourseIds(ctx, acc.ID)
	if err != nil {
		s.tel.ReportBroken(report_moodle_query_user_course_ids, err, acc.ID)
		return nil, err
	}

	lessonPlans, err := s.api.QueryLessonPlans(ctx, courseIds)
	if err != nil {
		s.tel.ReportBroken(report_moodle_query_lesson_plans, err, acc.ID, courseIds)
		return nil, err
	}

	return &connect.Response[moodlev1.LessonPlansResponse]{
		Msg: lessonPlans,
	}, nil
}

// ChapterContent implements the protobuf method.
func (s MoodleService) ChapterContent(ctx context.Context, req *connect.Request[moodlev1.ChapterContentRequest]) (*connect.Response[moodlev1.ChapterContentResponse], error) {
	_, ok := ctx.Value(s.ctxKey).(db.GetMoodleAccountFromTokenRow)
	if !ok {
		return nil, unauthorizedError
	}

	chapterId := req.Msg.GetId()
	content, err := s.api.QueryChapterContent(ctx, chapterId)
	if err != nil {
		s.tel.ReportBroken(report_moodle_query_chapter_content, err, chapterId)
		return nil, err
	}

	return &connect.Response[moodlev1.ChapterContentResponse]{
		Msg: &moodlev1.ChapterContentResponse{
			Content: content,
		},
	}, nil
}

// NewMoodleAuthInterceptor creates a struct that implements connect.Interceptor which will check the Authorization
// header if a valid token has been provided and return the moodle account associated with the token.
func NewMoodleAuthInterceptor(ctxKey any, db *db.Queries, tel telemetry.API) genericAuthInterceptor {
	return newGenericAuthInterceptor(func(ctx context.Context, token string) (context.Context, error) {
		acc, err := db.GetMoodleAccountFromToken(ctx, token)
		if err != nil {
			tel.ReportBroken(report_db_query, err, "GetMoodleAccountFromToken", token)
			return nil, err
		}
		return context.WithValue(ctx, ctxKey, acc), nil
	})
}
