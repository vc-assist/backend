package service

import (
	"context"
	"database/sql"
	"strings"
	powerschoolv1 "vcassist-backend/api/vcassist/powerschool/v1"
	publicv1 "vcassist-backend/api/vcassist/public/v1"
	servicedb "vcassist-backend/internal/service/db"

	"connectrpc.com/connect"
)

// PowerschoolScrapingAPI is basically MoodleScrapingAPI but for powerschool.
type PowerschoolScrapingAPI interface {
	// ScrapeAll does scraping for all the powerschool user accounts stored by AddUserAccount and stores
	// it in cache.
	ScrapeAll(ctx context.Context) error

	// ScrapeUser scrapes a specific user and updates its cache.
	ScrapeUser(ctx context.Context, email string) error

	// AddUserAccount adds a user to the list of all user accounts that will be scraped by ScrapeAll.
	AddUserAccount(ctx context.Context, token string) (email string, err error)

	// RemoveUserAccount removes a user from the list of all user accounts.
	RemoveUserAccount(ctx context.Context, email string) error
}

// PowerschoolQueryAPI is basically MoodleQueryAPI but for powerschool.
type PowerschoolQueryAPI interface {
	// QueryData reads the cached data for a given user.
	QueryData(ctx context.Context, email string) (*powerschoolv1.DataResponse, error)
}

func NormalizePSEmail(email string) string {
	email = strings.Trim(email, " \t\n")
	email = strings.ToLower(email)
	return email
}

// LoginPowerschool implements the protobuf method.
func (s PowerschoolService) LoginPowerschool(ctx context.Context, req *connect.Request[publicv1.LoginPowerschoolRequest]) (*connect.Response[publicv1.LoginPowerschoolResponse], error) {
	email, err := s.scraping.AddUserAccount(ctx, req.Msg.GetToken())
	if err != nil {
		s.tel.ReportBroken(REPORT_PS_SCRAPING_LOGIN, err, req.Msg.GetToken())
		return nil, err
	}
	email = NormalizePSEmail(email)

	failed := true
	defer func() {
		// this works because this is a closure
		// see: https://go.dev/tour/moretypes/25
		if !failed {
			return
		}
		err = s.scraping.RemoveUserAccount(ctx, email)
		if err != nil {
			s.tel.ReportBroken(REPORT_MOODLE_SCRAPING_REMOVE_USER, err, email)
		}
	}()

	tx, discard, commit := s.makeTx()
	defer discard()

	psAccountId, err := tx.SetPSAccount(ctx, email)
	if err != nil {
		s.tel.ReportBroken(REPORT_DB_QUERY, err, "SetPSCred")
		return nil, err
	}

	token, err := s.rand.GenerateToken()
	if err != nil {
		s.tel.ReportBroken(REPORT_RAND_TOKEN_GENERATION, err)
		return nil, err
	}

	err = tx.CreatePSToken(ctx, servicedb.CreatePSTokenParams{
		Token: token,
		PowerschoolAccountID: sql.NullInt64{
			Int64: psAccountId,
			Valid: true,
		},
	})
	if err != nil {
		s.tel.ReportBroken(REPORT_DB_QUERY, err, "CreatePSToken", psAccountId, token)
		return nil, err
	}

	commit()
	failed = false

	return &connect.Response[publicv1.LoginPowerschoolResponse]{
		Msg: &publicv1.LoginPowerschoolResponse{
			Token: token,
		},
	}, nil
}

// ScrapeAllPowerschool exposes a public method to update all the powerschool data (including the users)
// which an be run on a cron job
//
// note: cron job point
func (s PowerschoolService) ScrapeAllPowerschool(ctx context.Context) error {
	err := s.scraping.ScrapeAll(ctx)
	if err != nil {
		s.tel.ReportBroken(REPORT_PS_SCRAPING_SCRAPE_ALL, err)
		return err
	}
	return nil
}

// Refresh implements the protobuf method.
func (s PowerschoolService) Refresh(ctx context.Context, req *connect.Request[powerschoolv1.RefreshRequest]) (*connect.Response[powerschoolv1.RefreshResponse], error) {
	email, ok := ctx.Value(s.ctxKey).(string)
	if !ok {
		return nil, unauthorizedError
	}

	err := s.scraping.ScrapeUser(ctx, email)
	if err != nil {
		s.tel.ReportBroken(REPORT_PS_SCRAPING_SCRAPE_USER)
		return nil, err
	}

	return &connect.Response[powerschoolv1.RefreshResponse]{
		Msg: &powerschoolv1.RefreshResponse{},
	}, nil
}

// Data implements the protobuf method.
func (s PowerschoolService) Data(ctx context.Context, req *connect.Request[powerschoolv1.DataRequest]) (*connect.Response[powerschoolv1.DataResponse], error) {
	email, ok := ctx.Value(s.ctxKey).(string)
	if !ok {
		return nil, unauthorizedError
	}

	res, err := s.query.QueryData(ctx, email)
	if err != nil {
		s.tel.ReportBroken(REPORT_PS_QUERY_DATA)
		return nil, err
	}

	return &connect.Response[powerschoolv1.DataResponse]{
		Msg: res,
	}, nil
}
