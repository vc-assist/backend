package keychain

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

//no idea if this is bad or not @Shengzhi please double check

type Service struct {
	db     *sql.DB
	qry    *db.Queries
	client *resty.Client
}

type googleUserInfo struct {
	Email string `json:"email"`
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

func getEmailFromOAuth(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return "", fmt.Errorf("getEmailFromOAuth: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getEmailFromOAuth: %w", err)
	}
	buff, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("getEmail has been wrong")
	}

	var result googleUserInfo
	err = json.Unmarshal(buff, &result)
	if err != nil {
		slog.Debug("umarshal went wrong")
	}
	return result.Email, nil
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
	toke := req.Msg.GetKey().Token
	email, err := getEmailFromOAuth(ctx, toke)
	if err != nil {
		slog.Debug("email from token is not working")
		return nil, err
	}
	err = s.qry.CreateOAuth(ctx, db.CreateOAuthParams{
		Namespace:  req.Msg.GetNamespace(),
		ID:         0,
		Token:      toke,
		Email:      email,
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
		ID:        0,
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
		ID:        0,
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
		ID:        0,
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

func GenerateRandomBytes(length int) ([]byte, error) {
	token := make([]byte, length)
	_, err := rand.Read(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// generate token function
func GenerateBase64Token(length int) (string, error) {
	bytes, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// this function does not need to return an error
// either the token
func (s Service) CheckSessionToken(ctx context.Context, token string) (b bool) {
	_, err := s.qry.FindSessionToken(ctx, token)
	return err == sql.ErrNoRows
}
func (s Service) SetSessionToken(ctx context.Context, req *connect.Request[keychainv1.SetSessionTokenRequest]) (res *connect.Response[keychainv1.SetSessionTokenResponse], err error) {
	//get oauth token if it exists, insert it into the db(if it does) grab the id - if can
	//check useranme password passed in if exists then then grab id if it doesnt (either paremeters are null or new user - handle this) insert into db or just do nothing
	//finally create the random token, check if either one exists and insert in to the session token db
	mainReq := req.Msg
	var OAuthId int64
	var UsernamePasswordId int64
	if err != nil {
		slog.Debug("failed to properly generate token, apologies from Justin Shi")
	}
	if mainReq.OauthFeilds == nil {
		slog.Debug("OauthFeilds are nil")
	} else {
		s.qry.CreateOAuth(
			ctx,
			db.CreateOAuthParams{
				Namespace:  "powerschool",
				ID:         0, //this is autoincreamented in the db 0 is tem parameter
				Token:      mainReq.OauthFeilds.Token,
				RefreshUrl: mainReq.OauthFeilds.RefreshUrl,
				ClientID:   mainReq.OauthFeilds.ClientId,
				ExpiresAt:  mainReq.OauthFeilds.ExpiresAt,
			},
		)
		OAuthId, err = s.qry.FindIdFromOAuthToken(ctx, mainReq.OauthFeilds.Token)
		if err != nil {
			slog.Debug("cannot find the token in outh db insertion went wrong")
		}
	}

	if mainReq.UsernamePassword == nil {
		slog.Debug("Username Password feilds are nil")
	} else {
		s.qry.CreateUsernamePassword(
			ctx,
			db.CreateUsernamePasswordParams{
				Namespace: "moodle",
				ID:        0,
				Username:  mainReq.UsernamePassword.Username,
				Password:  mainReq.UsernamePassword.Password,
			},
		)
		UsernamePasswordId, err = s.qry.FindIdFromUsername(ctx, mainReq.UsernamePassword.Username)
		if err != nil {
			slog.Debug("insertion into the usernamepassword table went wrong")
		}
	}
	mainToken, err := GenerateBase64Token(32)
	sqlOauthId := sql.NullInt64{
		Int64: OAuthId,
		Valid: true,
	}
	sqlUsernamePasswordId := sql.NullInt64{
		Int64: UsernamePasswordId,
		Valid: true,
	}
	err = s.qry.CreateSessionToken(
		ctx,
		db.CreateSessionTokenParams{
			Token:              mainToken,
			Oauthid:            sqlOauthId,
			Usernamepasswordid: sqlUsernamePasswordId,
		},
	)
	if err != nil {
		slog.Debug("Session token failed to create")
	}
	return &connect.Response[keychainv1.SetSessionTokenResponse]{
		Msg: &keychainv1.SetSessionTokenResponse{
			SessionToken: mainToken,
		},
	}, nil
}
