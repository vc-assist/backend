package powerschoolapi

import (
	"context"
	"database/sql"
	"time"
	"vcassist-backend/cmd/powerschool_api/db"
	"vcassist-backend/lib/platforms/powerschool"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type OAuthDaemon struct {
	qry    *db.Queries
	db     *sql.DB
	config powerschool.OAuthConfig

	tracer         trace.Tracer
	refreshCounter metric.Int64Counter
}

func (d *OAuthDaemon) refreshToken(ctx context.Context, token db.OAuthToken) error {
	ctx, span := d.tracer.Start(ctx, "daemon:refreshToken")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "student_id",
			Value: attribute.StringValue(token.Studentid),
		},
		attribute.KeyValue{
			Key:   "expires_at",
			Value: attribute.Int64Value(token.Expiresat),
		},
		attribute.KeyValue{
			Key:   "token",
			Value: attribute.StringValue(token.Token),
		},
	)

	refreshed, expiresAt, err := d.config.Refresh(ctx, token.Token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to refresh oauth token")
		return err
	}

	err = d.qry.CreateOrUpdateOAuthToken(ctx, db.CreateOrUpdateOAuthTokenParams{
		Studentid: token.Studentid,
		Expiresat: expiresAt.Unix(),
		Token:     refreshed,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update db with refreshed token")
		return err
	}

	return nil
}

func (d *OAuthDaemon) refreshAllTokens(ctx context.Context) {
	ctx, span := d.tracer.Start(ctx, "oauth_daemon:refreshAllTokens")
	defer span.End()

	fiveMinutesFromNow := time.Now().Add(time.Minute * 5).Unix()
	almostExpired, err := d.qry.GetExpiredTokens(ctx, fiveMinutesFromNow)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get almost expired oauth tokens")
		return
	}

	for _, token := range almostExpired {
		err := d.refreshToken(ctx, token)
		if err != nil {
			continue
		}
		d.refreshCounter.Add(ctx, 1)
	}
}

func (d *OAuthDaemon) deleteExpiredTokens(ctx context.Context) {
	ctx, span := d.tracer.Start(ctx, "oauth_daemon:deleteExpiredTokens")
	defer span.End()

	err := d.qry.DeleteExpiredOAuthTokens(ctx, time.Now().Unix())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete expired oauth tokens")
	}
}

func (d *OAuthDaemon) refreshDaemon(ctx context.Context) {
	timer := time.NewTimer(time.Minute * 1)
	d.refreshAllTokens(ctx)
	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			d.refreshAllTokens(ctx)
		}
	}
}

func (d *OAuthDaemon) deletionDaemon(ctx context.Context) {
	timer := time.NewTimer(time.Minute * 30)
	d.deleteExpiredTokens(ctx)
	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			d.deleteExpiredTokens(ctx)
		}
	}
}

func (d *OAuthDaemon) Start(ctx context.Context) {
	go d.refreshDaemon(ctx)
	go d.deletionDaemon(ctx)
}
