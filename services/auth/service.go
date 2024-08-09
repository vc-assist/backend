package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"strings"
	"time"
	"vcassist-backend/lib/timezone"
	authv1 "vcassist-backend/proto/vcassist/services/auth/v1"
	"vcassist-backend/proto/vcassist/services/auth/v1/authv1connect"
	"vcassist-backend/services/auth/db"
	"vcassist-backend/services/auth/verifier"

	"connectrpc.com/connect"
	"github.com/jordan-wright/email"
	"github.com/mazen160/go-random"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("vcassist.services.auth")

type EmailConfig struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"`
}

type Service struct {
	db       *sql.DB
	qry      *db.Queries
	email    EmailConfig
	verifier verifier.Verifier
}

func NewService(database *sql.DB, email EmailConfig) authv1connect.AuthServiceClient {
	return authv1connect.NewInstrumentedAuthServiceClient(
		Service{
			db:       database,
			qry:      db.New(database),
			email:    email,
			verifier: verifier.NewVerifier(database),
		},
	)
}

func (s Service) createVerificationCode(ctx context.Context, txqry *db.Queries, email string) (code string, err error) {
	ctx, span := tracer.Start(ctx, "createVerificationCode")
	defer span.End()

	code, err = random.String(8)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to generate verification code")
		return "", err
	}
	err = txqry.CreateVerificationCode(ctx, db.CreateVerificationCodeParams{
		Code:      code,
		Useremail: email,
		Expiresat: timezone.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to insert verification code row")
		return "", err
	}

	return code, nil
}

func (s Service) sendVerificationCode(ctx context.Context, userEmail, code string) error {
	ctx, span := tracer.Start(ctx, "sendVerificationCode")
	defer span.End()

	mail := email.NewEmail()
	mail.From = fmt.Sprintf("VC Assist <%s>", s.email.EmailAddress)
	mail.To = []string{userEmail}
	mail.Subject = "Verification Code"

	body := fmt.Sprintf(`Please enter the following verification code for you VC Assist account when prompted.

%s

If you don't recognize this account, please ignore this email.`, code)
	mail.Text = []byte(body)

	err := mail.Send(
		fmt.Sprintf("%s:%d", s.email.Server, s.email.Port),
		smtp.PlainAuth("", s.email.EmailAddress, s.email.Password, s.email.Server),
	)
	if err != nil && strings.Contains(err.Error(), "server doesn't support AUTH") {
		err = mail.Send(fmt.Sprintf("%s:%d", s.email.Server, s.email.Port), nil)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to send email")
			return err
		}
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send email")
		return err
	}

	return nil
}

func (s Service) StartLogin(ctx context.Context, req *connect.Request[authv1.StartLoginRequest]) (*connect.Response[authv1.StartLoginResponse], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	email := req.Msg.GetEmail()

	err = txqry.EnsureUserExists(ctx, email)
	if err != nil {
		return nil, err
	}
	code, err := s.createVerificationCode(ctx, txqry, email)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	err = s.sendVerificationCode(ctx, email, code)
	if err != nil {
		return nil, err
	}

	return &connect.Response[authv1.StartLoginResponse]{Msg: &authv1.StartLoginResponse{}}, nil
}

func (s Service) verifyAndDeleteCode(ctx context.Context, txqry *db.Queries, email, code string) error {
	ctx, span := tracer.Start(ctx, "verifyAndDeleteCode")
	defer span.End()

	email, err := txqry.GetUserFromCode(ctx, code)
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Error, "invalid verification code")
		return fmt.Errorf("invalid verification code")
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user from code")
		return err
	}
	err = txqry.DeleteVerificationCode(ctx, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not delete consumed verification code")
		return err
	}

	return nil
}

func (s Service) createToken(ctx context.Context, txqry *db.Queries, email string) (string, error) {
	ctx, span := tracer.Start(ctx, "createToken")
	defer span.End()

	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to generate token")
		return "", err
	}
	token := hex.EncodeToString(nonce)
	err = txqry.CreateToken(ctx, db.CreateTokenParams{
		Useremail: email,
		Token:     token,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "got unexpected error while creating user")
		return "", err
	}

	return token, nil
}

func (s Service) ConsumeVerificationCode(ctx context.Context, req *connect.Request[authv1.ConsumeVerificationCodeRequest]) (*connect.Response[authv1.ConsumeVerificationCodeResponse], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	email := req.Msg.GetEmail()
	providedCode := req.Msg.GetProvidedCode()

	err = s.verifyAndDeleteCode(ctx, txqry, email, providedCode)
	if err != nil {
		return nil, err
	}
	token, err := s.createToken(ctx, txqry, email)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &connect.Response[authv1.ConsumeVerificationCodeResponse]{
		Msg: &authv1.ConsumeVerificationCodeResponse{
			Token: token,
		},
	}, nil
}

func (s Service) VerifyToken(ctx context.Context, req *connect.Request[authv1.VerifyTokenRequest]) (*connect.Response[authv1.VerifyTokenResponse], error) {
	user, err := s.verifier.VerifyToken(ctx, req.Msg.GetToken())
	if err != nil {
		return nil, err
	}

	return &connect.Response[authv1.VerifyTokenResponse]{
		Msg: &authv1.VerifyTokenResponse{
			Email: user.Email,
		},
	}, nil
}
