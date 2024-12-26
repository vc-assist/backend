package service

import (
	"context"
	"database/sql"
	"strings"
	powerschoolv1 "vcassist-backend/api/vcassist/powerschool/v1"
	publicv1 "vcassist-backend/api/vcassist/public/v1"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"

	"connectrpc.com/connect"
)

// PowerschoolAPI describes all the powerschool scraping methods (no scraping logic is in the service so the
// service's logic can be tested individually)
//
// note: there should not be any cron jobs running in here, cron jobs should only exist on the very top level
type PowerschoolAPI interface {
	// ScrapeUser scrapes a specific user and updates its cache.
	ScrapeUser(ctx context.Context, accountId int64) error

	// GetEmail gets the email associated with a token (if this succeeds this implies the token is valid).
	GetEmail(ctx context.Context, token string) (email string, err error)

	// QueryData reads the cached data for a given user.
	QueryData(ctx context.Context, accountId int64) (*powerschoolv1.DataResponse, error)
}

func normalizePSEmail(email string) string {
	email = strings.Trim(email, " \t\n")
	email = strings.ToLower(email)
	return email
}

// LoginPowerschool implements the protobuf method.
func (s PowerschoolService) LoginPowerschool(ctx context.Context, req *connect.Request[publicv1.LoginPowerschoolRequest]) (*connect.Response[publicv1.LoginPowerschoolResponse], error) {
	email, err := s.api.GetEmail(ctx, req.Msg.GetToken())
	if err != nil {
		s.tel.ReportBroken(report_ps_get_email, err, req.Msg.GetToken())
		return nil, err
	}
	email = normalizePSEmail(email)

	tx, discard, commit := s.makeTx()
	defer discard()

	psAccountId, err := tx.SetPSAccount(ctx, email)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "SetPSCred")
		return nil, err
	}

	token, err := s.rand.GenerateToken()
	if err != nil {
		s.tel.ReportBroken(report_rand_token_generation, err)
		return nil, err
	}

	err = tx.CreatePSToken(ctx, db.CreatePSTokenParams{
		Token: token,
		PowerschoolAccountID: sql.NullInt64{
			Int64: psAccountId,
			Valid: true,
		},
	})
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "CreatePSToken", psAccountId, token)
		return nil, err
	}

	commit()

	userCount, err := s.db.GetPSUserCount(ctx)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetPSUserCount")
	} else {
		s.tel.ReportCount(report_ps_user_count, userCount)
	}

	return &connect.Response[publicv1.LoginPowerschoolResponse]{
		Msg: &publicv1.LoginPowerschoolResponse{
			Token: token,
		},
	}, nil
}

// Refresh implements the protobuf method.
func (s PowerschoolService) Refresh(ctx context.Context, req *connect.Request[powerschoolv1.RefreshRequest]) (*connect.Response[powerschoolv1.RefreshResponse], error) {
	acc, ok := ctx.Value(s.ctxKey).(db.GetPSAccountFromTokenRow)
	if !ok {
		return nil, unauthorizedError
	}

	err := s.api.ScrapeUser(ctx, acc.ID)
	if err != nil {
		s.tel.ReportBroken(report_ps_scrape_user)
		return nil, err
	}

	return &connect.Response[powerschoolv1.RefreshResponse]{
		Msg: &powerschoolv1.RefreshResponse{},
	}, nil
}

// Data implements the protobuf method.
func (s PowerschoolService) Data(ctx context.Context, req *connect.Request[powerschoolv1.DataRequest]) (*connect.Response[powerschoolv1.DataResponse], error) {
	acc, ok := ctx.Value(s.ctxKey).(db.GetPSAccountFromTokenRow)
	if !ok {
		return nil, unauthorizedError
	}

	res, err := s.api.QueryData(ctx, acc.ID)
	if err != nil {
		s.tel.ReportBroken(report_ps_query_data)
		return nil, err
	}

	return &connect.Response[powerschoolv1.DataResponse]{
		Msg: res,
	}, nil
}

// NewPowerschoolAuthInterceptor creates a struct that implements connect.Interceptor which will check the Authorization
// header if a valid token has been provided and return the powerschool account associated with the token.
func NewPowerschoolAuthInterceptor(ctxKey any, db *db.Queries, tel telemetry.API) genericAuthInterceptor {
	return newGenericAuthInterceptor(func(ctx context.Context, token string) (context.Context, error) {
		acc, err := db.GetPSAccountFromToken(ctx, token)
		if err != nil {
			tel.ReportBroken(report_db_query, err, "GetPSAccountFromToken", token)
			return nil, err
		}
		return context.WithValue(ctx, ctxKey, acc), nil
	})
}
