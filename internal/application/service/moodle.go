package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	moodlev1 "vcassist-backend/api/vcassist/moodle/v1"
	publicv1 "vcassist-backend/api/vcassist/public/v1"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"

	"connectrpc.com/connect"
)

// MoodleAPI describes all the moodle scraping methods (no scraping logic is in the service so the
// service's logic can be tested individually)
//
// note: there should not be any cron jobs running in here, cron jobs should only exist on the very top level
type MoodleAPI interface {
	// ScrapeUser updates the "user" information for a specific user.
	ScrapeUser(ctx context.Context, accountId int64) error

	// TestLogin tests if the user login information is correct.
	TestLogin(ctx context.Context, username, password string) error

	// QueryLessonPlans transforms the cached moodle data into a *moodlev1.LessonPlansResponse
	// given a list of course ids.
	QueryLessonPlans(ctx context.Context, courseIds []int64) (*moodlev1.LessonPlansResponse, error)

	// QueryChapterContent returns the html content for a given chapter id.
	QueryChapterContent(ctx context.Context, chapterId int64) (string, error)

	// QueryUserCourseIds returns the ids for the moodle courses available to a given user's account.
	QueryUserCourseIds(ctx context.Context, accountId int64) ([]int64, error)
}

const email_suffix = "@warriorlife.net"

// this removes potential formatting inconsistencies from user input (extra spaces,
// capitalization, adding @warriorlife.net to the end of the username)
func normalizeMoodleUsername(moodleUsername string) string {
	username := moodleUsername
	username = strings.Trim(username, " \n\t")
	username = strings.ToLower(username)
	if strings.HasSuffix(username, email_suffix) {
		username = username[:len(username)-len(email_suffix)]
	}
	return username
}

// LoginMoodle implements the protobuf method.
func (s MoodleService) LoginMoodle(ctx context.Context, req *connect.Request[publicv1.LoginMoodleRequest]) (*connect.Response[publicv1.LoginMoodleResponse], error) {
	username := normalizeMoodleUsername(req.Msg.GetUsername())
	password := req.Msg.GetPassword()

	err := s.api.TestLogin(ctx, username, password)
	if err != nil {
		if !strings.Contains(err.Error(), "invalid username or password") {
			s.tel.ReportBroken(report_moodle_login, err, username, password)
		}
		return nil, err
	}

	tx, discard, commit := s.makeTx()
	defer discard()

	moodleAccountId, err := tx.AddMoodleAccount(ctx, username)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "AddMoodleAccount", username, password)
		return nil, err
	}

	token, err := s.rand.GenerateToken()
	if err != nil {
		s.tel.ReportBroken(report_rand_token_generation, err)
		return nil, err
	}

	err = tx.CreateMoodleToken(ctx, db.CreateMoodleTokenParams{
		Token: token,
		MoodleAccountID: sql.NullInt64{
			Int64: moodleAccountId,
			Valid: true,
		},
	})
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "CreateMoodleToken", moodleAccountId, token)
		return nil, err
	}

	commit()

	err = s.api.ScrapeUser(ctx, moodleAccountId)
	if err != nil {
		s.tel.ReportBroken(report_moodle_scrape_user, err, username, password)
		return nil, err
	}

	userCount, err := s.db.GetMoodleUserCount(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetMoodleUserCount")
	} else {
		s.tel.ReportCount(report_moodle_user_count, userCount)
	}

	return &connect.Response[publicv1.LoginMoodleResponse]{
		Msg: &publicv1.LoginMoodleResponse{
			Token: token,
		},
	}, nil
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
