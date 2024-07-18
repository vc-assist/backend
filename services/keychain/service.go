package keychain

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/timezone"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("services/keychain")

type Service struct {
	db  *sql.DB
	qry *db.Queries
}

func NewService(database *sql.DB) Service {
	return Service{
		db:  database,
		qry: db.New(database),
	}
}

func (s Service) refreshOAuthKey(ctx context.Context, original db.OAuth) error {
	ctx, span := tracer.Start(ctx, "refreshOAuthToken")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "expires_at",
			Value: attribute.Int64Value(original.ExpiresAt),
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
		ctx, originalToken, original.RefreshUrl, original.ClientID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to refresh oauth token")
		return err
	}

	expiresAt := timezone.Now().Add(time.Duration(newTokenObject.ExpiresIn))

	err = s.qry.CreateOAuth(ctx, db.CreateOAuthParams{
		ID:         original.ID,
		Namespace:  original.Namespace,
		RefreshUrl: original.RefreshUrl,
		ClientID:   original.ClientID,
		Token:      newToken,
		ExpiresAt:  expiresAt.Unix(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update db with refreshed token")
		return err
	}

	return nil
}
func (s Service) refreshAllOAuthKeys(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "refreshOAuthKeys")
	defer span.End()

	now := timezone.Now().Add(5 * time.Minute)
	almostExpired, err := s.qry.GetOAuthBefore(ctx, now.Unix())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, row := range almostExpired {
		s.refreshOAuthKey(ctx, row)
	}
	return nil
}

func (s Service) refreshOAuthDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.refreshAllOAuthKeys(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s Service) deleteOAuthDaemon(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, span := tracer.Start(ctx, "deleteExpiredOAuth")

			err := s.qry.DeleteOAuthBefore(ctx, timezone.Now().Unix())
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			span.End()
		case <-ctx.Done():
			return
		}
	}
}

func (s Service) StartOAuthDaemon(ctx context.Context) {
	go s.refreshOAuthDaemon(ctx)
	go s.deleteOAuthDaemon(ctx)
}

func (s Service) SetOAuth(ctx context.Context, req *connect.Request[keychainv1.SetOAuthRequest]) (*connect.Response[keychainv1.SetOAuthResponse], error) {
	ctx, span := tracer.Start(ctx, "SetOAuth")
	defer span.End()

	err := s.qry.CreateOAuth(ctx, db.CreateOAuthParams{
		Namespace:  req.Msg.GetNamespace(),
		ID:         req.Msg.GetId(),
		Token:      req.Msg.GetKey().GetToken(),
		RefreshUrl: req.Msg.GetKey().GetRefreshUrl(),
		ClientID:   req.Msg.GetKey().GetClientId(),
		ExpiresAt:  req.Msg.GetKey().GetExpiresAt(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[keychainv1.SetOAuthResponse]{
		Msg: &keychainv1.SetOAuthResponse{},
	}, nil
}

func (s Service) GetOAuth(ctx context.Context, req *connect.Request[keychainv1.GetOAuthRequest]) (*connect.Response[keychainv1.GetOAuthResponse], error) {
	ctx, span := tracer.Start(ctx, "GetOAuth")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", req.Msg.GetNamespace()),
		attribute.String("id", req.Msg.GetId()),
	)

	row, err := s.qry.GetOAuth(ctx, db.GetOAuthParams{
		Namespace: req.Msg.GetNamespace(),
		ID:        req.Msg.GetId(),
	})
	if err == sql.ErrNoRows || row.ExpiresAt < timezone.Now().Unix() {
		span.SetStatus(codes.Error, "key not found")
		return &connect.Response[keychainv1.GetOAuthResponse]{
			Msg: &keychainv1.GetOAuthResponse{
				Key: nil,
			},
		}, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[keychainv1.GetOAuthResponse]{
		Msg: &keychainv1.GetOAuthResponse{
			Key: &keychainv1.OAuthKey{
				Token:      row.Token,
				RefreshUrl: row.RefreshUrl,
				ClientId:   row.ClientID,
			},
		},
	}, nil
}

func (s Service) SetUsernamePassword(ctx context.Context, req *connect.Request[keychainv1.SetUsernamePasswordRequest]) (*connect.Response[keychainv1.SetUsernamePasswordResponse], error) {
	ctx, span := tracer.Start(ctx, "SetUsernamePassword")
	defer span.End()

	err := s.qry.CreateUsernamePassword(ctx, db.CreateUsernamePasswordParams{
		Namespace: req.Msg.GetNamespace(),
		ID:        req.Msg.GetId(),
		Username:  req.Msg.GetKey().GetUsername(),
		Password:  req.Msg.GetKey().GetPassword(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[keychainv1.SetUsernamePasswordResponse]{
		Msg: &keychainv1.SetUsernamePasswordResponse{},
	}, nil
}

func (s Service) GetUsernamePassword(ctx context.Context, req *connect.Request[keychainv1.GetUsernamePasswordRequest]) (*connect.Response[keychainv1.GetUsernamePasswordResponse], error) {
	ctx, span := tracer.Start(ctx, "GetUsernamePassword")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", req.Msg.GetNamespace()),
		attribute.String("id", req.Msg.GetId()),
	)

	row, err := s.qry.GetUsernamePassword(ctx, db.GetUsernamePasswordParams{
		Namespace: req.Msg.GetNamespace(),
		ID:        req.Msg.GetId(),
	})
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Error, "key not found")
		return &connect.Response[keychainv1.GetUsernamePasswordResponse]{
			Msg: &keychainv1.GetUsernamePasswordResponse{
				Key: nil,
			},
		}, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[keychainv1.GetUsernamePasswordResponse]{
		Msg: &keychainv1.GetUsernamePasswordResponse{
			Key: &keychainv1.UsernamePasswordKey{
				Username: row.Username,
				Password: row.Password,
			},
		},
	}, nil
}
