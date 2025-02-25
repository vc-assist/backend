package service

import (
	"context"
	powerschoolv1 "vcassist-backend/api/vcassist/powerschool/v1"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"

	"connectrpc.com/connect"
)

const (
	report_ps_user_count  = "powerschool.user-count"
	report_ps_get_email   = "powerschool.get-email"
	report_ps_scrape_user = "powerschool.scrape-user"
	report_ps_query_data  = "powerschool.query-data"
)

// PowerschoolAPI describes all the powerschool scraping methods (no scraping logic is in the service so the
// service's logic can be tested individually)
//
// note: there should not be any cron jobs running in here, cron jobs should only exist on the very top level
type PowerschoolAPI interface {
	// ScrapeUser scrapes a specific user and updates its cache.
	ScrapeUser(ctx context.Context, accountId int64) error

	// QueryData reads the cached data for a given user.
	QueryData(ctx context.Context, accountId int64) (*powerschoolv1.DataResponse, error)
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

// NewPowerschoolAuthInterceptor creates a struct that implements [connect.Interceptor] which will check the Authorization
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
