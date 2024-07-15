package powerservice

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/timezone"
	"vcassist-backend/services/powerservice/db"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("services/powerschoold")

type OAuthDaemon struct {
	qry            *db.Queries
	db             *sql.DB
	config         OAuthConfig
	refreshCounter metric.Int64Counter
}

func NewOAuthDaemon(database *sql.DB, config OAuthConfig) (OAuthDaemon, error) {
	refreshCounter, err := meter.Int64Counter(
		"oauthd_refresh_total",
		metric.WithDescription("The total amount of times a token has been refreshed."),
	)
	if err != nil {
		return OAuthDaemon{}, err
	}
	return OAuthDaemon{
		db:             database,
		qry:            db.New(database),
		config:         config,
		refreshCounter: refreshCounter,
	}, nil
}

func (d OAuthDaemon) refreshToken(ctx context.Context, original db.OAuthToken) error {
	ctx, span := tracer.Start(ctx, "oauth_daemon:refreshToken")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "student_id",
			Value: attribute.StringValue(original.Studentid),
		},
		attribute.KeyValue{
			Key:   "expires_at",
			Value: attribute.Int64Value(original.Expiresat),
		},
		attribute.KeyValue{
			Key:   "token",
			Value: attribute.StringValue(original.Token),
		},
	)

	var originalToken oauth.OpenIdToken
	err := json.Unmarshal([]byte(original.Token), &originalToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deserialize original token")
		return err
	}

	newToken, newTokenObject, err := oauth.Refresh(
		ctx, originalToken, d.config.RefreshUrl, d.config.ClientId,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to refresh oauth token")
		return err
	}

	expiresAt := timezone.Now().Add(time.Duration(newTokenObject.ExpiresIn))

	err = d.qry.CreateOrUpdateOAuthToken(ctx, db.CreateOrUpdateOAuthTokenParams{
		Studentid: original.Studentid,
		Expiresat: expiresAt.Unix(),
		Token:     newToken,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update db with refreshed token")
		return err
	}

	return nil
}

func (d OAuthDaemon) refreshAllTokens(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "oauth_daemon:refreshAllTokens")
	defer span.End()

	fiveMinutesFromNow := timezone.Now().Add(time.Minute * 5).Unix()
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

func (d OAuthDaemon) deleteExpiredTokens(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "oauth_daemon:deleteExpiredTokens")
	defer span.End()

	err := d.qry.DeleteExpiredOAuthTokens(ctx, timezone.Now().Unix())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete expired oauth tokens")
	}
}

func (d OAuthDaemon) refreshDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 1)
	d.refreshAllTokens(ctx)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			d.refreshAllTokens(ctx)
		}
	}
}

func (d OAuthDaemon) deletionDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 30)
	d.deleteExpiredTokens(ctx)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			d.deleteExpiredTokens(ctx)
		}
	}
}

func (d OAuthDaemon) Start(ctx context.Context) {
	go d.refreshDaemon(ctx)
	go d.deletionDaemon(ctx)
}
