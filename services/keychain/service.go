package keychain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/timezone"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
	"github.com/go-resty/resty/v2"

	_ "modernc.org/sqlite"
)

type Service struct {
	db     *sql.DB
	qry    *db.Queries
	client *resty.Client
}

func NewService(ctx context.Context, database *sql.DB) keychainv1connect.KeychainServiceClient {
	client := resty.New()
	client.SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	client.SetTimeout(time.Second * 30)

	restyutil.InstrumentClient(client, nil, restyInstrumentOutput)

	s := Service{
		db:     database,
		qry:    db.New(database),
		client: client,
	}

	go s.refreshOAuthDaemon(ctx)
	go s.deleteOAuthDaemon(ctx)

	return s
}

func (s Service) refreshOAuthKey(ctx context.Context, originalRow db.OAuth) error {
	var originalToken oauth.OpenIdToken
	err := json.Unmarshal([]byte(originalRow.Token), &originalToken)
	if err != nil {
		return err
	}

	if originalToken.RefreshToken == "" {
		err := fmt.Errorf("token is not refreshable")
		return err
	}

	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("client_id", originalRow.ClientID)
	form.Add("scope", originalToken.Scope)
	form.Add("refresh_token", originalToken.RefreshToken)

	res, err := s.client.R().
		SetContext(ctx).
		SetBody(form.Encode()).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		Post(originalRow.RefreshUrl)
	if err != nil {
		return err
	}

	var newToken oauth.OpenIdToken
	err = json.Unmarshal(res.Body(), &newToken)
	if err != nil {
		return err
	}

	newToken.RefreshToken = originalToken.RefreshToken
	expiresAt := timezone.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)

	newTokenJson, err := json.Marshal(newToken)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "refreshed oauth token", "new_token", string(newTokenJson))

	err = s.qry.CreateOAuth(ctx, db.CreateOAuthParams{
		ID:         originalRow.ID,
		Namespace:  originalRow.Namespace,
		RefreshUrl: originalRow.RefreshUrl,
		ClientID:   originalRow.ClientID,
		Token:      string(newTokenJson),
		ExpiresAt:  expiresAt.Unix(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s Service) refreshAllOAuthKeys(ctx context.Context) error {
	now := timezone.Now().Add(5 * time.Minute)
	almostExpired, err := s.qry.GetOAuthBefore(ctx, now.Unix())
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, row := range almostExpired {
		wg.Add(1)
		go func(row db.OAuth) {
			err := s.refreshOAuthKey(ctx, row)
			if err != nil {
				slog.WarnContext(
					ctx, "failed to refresh oauth key",
					slog.Group("key",
						"namespace", row.Namespace,
						"id", row.ID,
					),
					"err", err,
				)
			}
			wg.Done()
		}(row)
	}
	wg.Wait()

	return nil
}

func (s Service) refreshOAuthDaemon(ctx context.Context) {
	slog.InfoContext(ctx, "start daemon", "task", "refresh oauth keys every 3 minutes")

	ticker := time.NewTicker(time.Minute * 3)
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
	slog.InfoContext(ctx, "start daemon", "task", "delete expired oauth keys every 30 minutes")

	ticker := time.NewTicker(time.Minute * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := s.qry.DeleteOAuthBefore(ctx, timezone.Now().Unix())
			if err != nil {
				slog.WarnContext(ctx, "failed to delete expired oauth keys", "err", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s Service) SetOAuth(ctx context.Context, req *connect.Request[keychainv1.SetOAuthRequest]) (*connect.Response[keychainv1.SetOAuthResponse], error) {
	err := s.qry.CreateOAuth(ctx, db.CreateOAuthParams{
		Namespace:  req.Msg.GetNamespace(),
		ID:         req.Msg.GetId(),
		Token:      req.Msg.GetKey().GetToken(),
		RefreshUrl: req.Msg.GetKey().GetRefreshUrl(),
		ClientID:   req.Msg.GetKey().GetClientId(),
		ExpiresAt:  req.Msg.GetKey().GetExpiresAt(),
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[keychainv1.SetOAuthResponse]{
		Msg: &keychainv1.SetOAuthResponse{},
	}, nil
}

func (s Service) GetOAuth(ctx context.Context, req *connect.Request[keychainv1.GetOAuthRequest]) (*connect.Response[keychainv1.GetOAuthResponse], error) {
	row, err := s.qry.GetOAuth(ctx, db.GetOAuthParams{
		Namespace: req.Msg.GetNamespace(),
		ID:        req.Msg.GetId(),
	})
	if err == sql.ErrNoRows || row.ExpiresAt < timezone.Now().Unix() {
		return &connect.Response[keychainv1.GetOAuthResponse]{
			Msg: &keychainv1.GetOAuthResponse{
				Key: nil,
			},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &connect.Response[keychainv1.GetOAuthResponse]{
		Msg: &keychainv1.GetOAuthResponse{
			Key: &keychainv1.OAuthKey{
				Token:      row.Token,
				RefreshUrl: row.RefreshUrl,
				ClientId:   row.ClientID,
				ExpiresAt:  row.ExpiresAt,
			},
		},
	}, nil
}

func (s Service) SetUsernamePassword(ctx context.Context, req *connect.Request[keychainv1.SetUsernamePasswordRequest]) (*connect.Response[keychainv1.SetUsernamePasswordResponse], error) {
	err := s.qry.CreateUsernamePassword(ctx, db.CreateUsernamePasswordParams{
		Namespace: req.Msg.GetNamespace(),
		ID:        req.Msg.GetId(),
		Username:  req.Msg.GetKey().GetUsername(),
		Password:  req.Msg.GetKey().GetPassword(),
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[keychainv1.SetUsernamePasswordResponse]{
		Msg: &keychainv1.SetUsernamePasswordResponse{},
	}, nil
}

func (s Service) GetUsernamePassword(ctx context.Context, req *connect.Request[keychainv1.GetUsernamePasswordRequest]) (*connect.Response[keychainv1.GetUsernamePasswordResponse], error) {
	row, err := s.qry.GetUsernamePassword(ctx, db.GetUsernamePasswordParams{
		Namespace: req.Msg.GetNamespace(),
		ID:        req.Msg.GetId(),
	})
	if err == sql.ErrNoRows {
		return &connect.Response[keychainv1.GetUsernamePasswordResponse]{
			Msg: &keychainv1.GetUsernamePasswordResponse{
				Key: nil,
			},
		}, nil
	}
	if err != nil {
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
