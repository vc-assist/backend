package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	moodlev1 "vcassist-backend/api/vcassist/moodle/v1"
	publicv1 "vcassist-backend/api/vcassist/public/v1"
	servicedb "vcassist-backend/internal/service/db"

	"connectrpc.com/connect"
)

// MoodleScrapingAPI describes all the methods that affect the information moodle scrapes and stores. It is
// effectively the "write" API.
//
// note: there should not be any cron jobs running in here, cron jobs should only exist on the very top level
// (whatever uses Service)
type MoodleScrapingAPI interface {
	// ScrapeAll performs moodle scraping for all courses using the admin account, it also does updates
	// for the "user" information for all the "users".
	ScrapeAll(ctx context.Context) error

	// ScrapeUser updates the "user" information for a specific user.
	ScrapeUser(ctx context.Context, username string) error

	// AddUserAccount adds a "user" account (and tests if the credentials are valid)
	// for which, only the information available under "QueryUser..." methods need to be scraped.
	//
	// note: It should NOT automatically call "ScrapeUser", this allows for a user to exist
	// but their information not be populated, in that case, simply return the zero value.
	AddUserAccount(ctx context.Context, username, password string) error

	// RemoveUserAccount removes a "user" account from scraping.
	RemoveUserAccount(ctx context.Context, username string) error
}

// MoodleQueryAPI describes all the methods that query the information moodle stores. It is effectively the
// "read" API.
//
// This also implies that the scraping API has the responsibility of ensuring that QueryData will always return
// some data (or a fatal error).
//
// note: there should not be any cron jobs running in here, cron jobs should only exist on the very top level
// (whatever uses Service)
type MoodleQueryAPI interface {
	// QueryLessonPlans transforms the cached moodle data into a *moodlev1.LessonPlansResponse
	// given a list of course ids.
	QueryLessonPlans(ctx context.Context, courseIds []int64) (*moodlev1.LessonPlansResponse, error)

	// QueryChapterContent returns the html content for a given chapter id.
	QueryChapterContent(ctx context.Context, chapterId int64) (string, error)

	// QueryUserCourseIds returns the ids for the moodle courses available to a given user's account.
	QueryUserCourseIds(ctx context.Context, username string) ([]int64, error)
}

const email_suffix = "@warriorlife.net"

func NormalizeMoodleUsername(moodleUsername string) string {
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
	// this removes potential formatting inconsistencies from user input
	username := NormalizeMoodleUsername(req.Msg.GetUsername())
	password := req.Msg.GetPassword()

	err := s.scraping.AddUserAccount(ctx, username, password)
	if err != nil {
		if !strings.Contains(err.Error(), "invalid username or password") {
			s.tel.ReportBroken(REPORT_MOODLE_SCRAPING_LOGIN, err, username, password)
			return nil, err
		}
		return nil, err
	}

	failed := true
	defer func() {
		// this works because this is a closure
		// see: https://go.dev/tour/moretypes/25
		if !failed {
			return
		}
		err = s.scraping.RemoveUserAccount(ctx, username)
		if err != nil {
			s.tel.ReportBroken(REPORT_MOODLE_SCRAPING_REMOVE_USER, err, username)
		}
	}()

	err = s.scraping.ScrapeUser(ctx, username)
	if err != nil {
		s.tel.ReportBroken(REPORT_MOODLE_SCRAPING_SCRAPE_USER, err, username, password)
		return nil, err
	}

	tx, discard, commit := s.makeTx()
	defer discard()

	moodleAccountId, err := tx.SetMoodleAccount(ctx, username)
	if err != nil {
		s.tel.ReportBroken(REPORT_DB_QUERY, err, "SetMoodleAccount", username, password)
		return nil, err
	}

	token, err := s.rand.GenerateToken()
	if err != nil {
		s.tel.ReportBroken(REPORT_RAND_TOKEN_GENERATION, err)
		return nil, err
	}

	err = tx.CreateMoodleToken(ctx, servicedb.CreateMoodleTokenParams{
		Token: token,
		MoodleAccountID: sql.NullInt64{
			Int64: moodleAccountId,
			Valid: true,
		},
	})
	if err != nil {
		s.tel.ReportBroken(REPORT_DB_QUERY, err, "CreateMoodleToken", moodleAccountId, token)
		return nil, err
	}

	commit()
	failed = false

	return &connect.Response[publicv1.LoginMoodleResponse]{
		Msg: &publicv1.LoginMoodleResponse{
			Token: token,
		},
	}, nil
}

// ScrapeAllMoodle exposes a public method to update all the moodle data (including the users)
// which an be run on a cron job
//
// note: cron job point
func (s MoodleService) ScrapeAllMoodle(ctx context.Context) error {
	err := s.scraping.ScrapeAll(ctx)
	if err != nil {
		s.tel.ReportBroken(REPORT_MOODLE_SCRAPING_SCRAPE_ALL, err)
		return err
	}
	return nil
}

var unauthorizedError = fmt.Errorf("unauthorized")

// Refresh implements the protobuf method.
func (s MoodleService) Refresh(ctx context.Context, req *connect.Request[moodlev1.RefreshRequest]) (*connect.Response[moodlev1.RefreshResponse], error) {
	username, ok := ctx.Value(s.ctxKey).(string)
	if !ok {
		return nil, unauthorizedError
	}

	err := s.scraping.ScrapeUser(ctx, username)
	if err != nil {
		s.tel.ReportBroken(REPORT_MOODLE_SCRAPING_SCRAPE_USER, err, username)
		return nil, err
	}

	return &connect.Response[moodlev1.RefreshResponse]{
		Msg: &moodlev1.RefreshResponse{},
	}, nil
}

// LessonPlans implements the protobuf method.
func (s MoodleService) LessonPlans(ctx context.Context, req *connect.Request[moodlev1.LessonPlansRequest]) (*connect.Response[moodlev1.LessonPlansResponse], error) {
	username, ok := ctx.Value(s.ctxKey).(string)
	if !ok {
		return nil, unauthorizedError
	}

	courseIds, err := s.query.QueryUserCourseIds(ctx, username)
	if err != nil {
		s.tel.ReportBroken(REPORT_MOODLE_QUERY_USER_COURSE_IDS, err, username)
		return nil, err
	}

	lessonPlans, err := s.query.QueryLessonPlans(ctx, courseIds)
	if err != nil {
		s.tel.ReportBroken(REPORT_MOODLE_QUERY_LESSON_PLANS, err, username, courseIds)
		return nil, err
	}

	return &connect.Response[moodlev1.LessonPlansResponse]{
		Msg: lessonPlans,
	}, nil
}

// ChapterContent implements the protobuf method.
func (s MoodleService) ChapterContent(ctx context.Context, req *connect.Request[moodlev1.ChapterContentRequest]) (*connect.Response[moodlev1.ChapterContentResponse], error) {
	_, ok := ctx.Value(s.ctxKey).(string)
	if !ok {
		return nil, unauthorizedError
	}

	chapterId := req.Msg.GetId()
	content, err := s.query.QueryChapterContent(ctx, chapterId)
	if err != nil {
		s.tel.ReportBroken(REPORT_MOODLE_QUERY_CHAPTER_CONTENT, err, chapterId)
		return nil, err
	}

	return &connect.Response[moodlev1.ChapterContentResponse]{
		Msg: &moodlev1.ChapterContentResponse{
			Content: content,
		},
	}, nil
}
