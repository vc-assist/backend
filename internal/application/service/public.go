package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	openidv1 "vcassist-backend/api/openid/v1"
	publicv1 "vcassist-backend/api/vcassist/public/v1"
	"vcassist-backend/internal/components/db"

	"connectrpc.com/connect"
)

type PublicAPI interface {
	// GetEmail gets the email associated with a token (if this succeeds this implies the token is valid).
	GetEmail(ctx context.Context, token *openidv1.Token) (email string, err error)

	// RefreshToken refreshes a powerschool access token.
	RefreshToken(ctx context.Context, token *openidv1.Token) (string, error)

	// TestMoodleLogin tests if the user login information is correct.
	TestMoodleLogin(ctx context.Context, username, password string) error
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
func (s PublicService) LoginMoodle(ctx context.Context, req *connect.Request[publicv1.LoginMoodleRequest]) (*connect.Response[publicv1.LoginMoodleResponse], error) {
	username := normalizeMoodleUsername(req.Msg.GetUsername())
	password := req.Msg.GetPassword()

	err := s.api.TestMoodleLogin(ctx, username, password)
	if err != nil {
		if !strings.Contains(err.Error(), "invalid username or password") {
			s.tel.ReportBroken(report_moodle_login, err, username, password)
		}
		return nil, err
	}

	tx, discard, commit, err := s.makeTx()
	if err != nil {
		s.tel.ReportBroken(report_db_query, fmt.Errorf("make tx: %w", err))
		return nil, err
	}
	defer discard()

	moodleAccountId, err := tx.AddMoodleAccount(ctx, db.AddMoodleAccountParams{
		Username: username,
		Password: password,
	})
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

func normalizePSEmail(email string) string {
	email = strings.Trim(email, " \t\n")
	email = strings.ToLower(email)
	return email
}

// LoginPowerschool implements the protobuf method.
func (s PublicService) LoginPowerschool(ctx context.Context, req *connect.Request[publicv1.LoginPowerschoolRequest]) (*connect.Response[publicv1.LoginPowerschoolResponse], error) {
	email, err := s.api.GetEmail(ctx, req.Msg.GetToken())
	if err != nil {
		s.tel.ReportBroken(report_ps_get_email, err, req.Msg.GetToken())
		return nil, err
	}
	email = normalizePSEmail(email)

	tx, discard, commit, err := s.makeTx()
	if err != nil {
		s.tel.ReportBroken(report_db_query, fmt.Errorf("make tx: %w", err))
		return nil, err
	}
	defer discard()

	psAccountId, err := tx.AddPSAccount(ctx, db.AddPSAccountParams{
		Email:        email,
		RefreshToken: req.Msg.GetToken().GetRefreshToken(),
		AccessToken:  req.Msg.GetToken().GetAccessToken(),
		IDToken:      req.Msg.GetToken().GetIdToken(),
		TokenType:    req.Msg.GetToken().GetTokenType(),
		Scope:        req.Msg.GetToken().GetScope(),
		ExpiresAt: s.time.Now().Add(
			time.Duration(req.Msg.GetToken().GetExpiresIn()) * time.Second,
		),
	})
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "AddPSAccount")
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
