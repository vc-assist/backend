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
	"vcassist-backend/services/auth/api"
	"vcassist-backend/services/auth/db"
	"vcassist-backend/services/auth/verifier"

	"connectrpc.com/connect"
	"github.com/jordan-wright/email"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("services/auth")

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

func NewService(database *sql.DB, email EmailConfig) Service {
	return Service{
		db:       database,
		qry:      db.New(database),
		email:    email,
		verifier: verifier.NewVerifier(database),
	}
}

func generateVerificationCode() (string, error) {
	nonce := make([]byte, 4)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(fmt.Sprintf(
		"%s-%s",
		hex.EncodeToString(nonce[0:2]),
		hex.EncodeToString(nonce[2:]),
	)), nil
}

func (s Service) createVerificationCode(ctx context.Context, txqry *db.Queries, email string) (code string, err error) {
	ctx, span := tracer.Start(ctx, "auth:createVerificationCode")
	defer span.End()

	code, err = generateVerificationCode()
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
	ctx, span := tracer.Start(ctx, "auth:sendVerificationCode")
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

func (s Service) StartLogin(ctx context.Context, req *connect.Request[api.StartLoginRequest]) (*connect.Response[api.StartLoginResponse], error) {
	ctx, span := tracer.Start(ctx, "auth:StartLogin")
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to begin transaction")
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	email := req.Msg.GetEmail()

	err = txqry.EnsureUserExists(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "got unexpected error while ensuring user exists")
		return nil, err
	}
	code, err := s.createVerificationCode(ctx, txqry, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create verification code")
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return nil, err
	}

	err = s.sendVerificationCode(ctx, email, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send verification code")
		return nil, err
	}

	return &connect.Response[api.StartLoginResponse]{Msg: &api.StartLoginResponse{}}, nil
}

func (s Service) verifyAndDeleteCode(ctx context.Context, txqry *db.Queries, email, code string) error {
	ctx, span := tracer.Start(ctx, "auth:verifyAndDeleteCode")
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
	ctx, span := tracer.Start(ctx, "auth:createToken")
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

func (s Service) ConsumeVerificationCode(ctx context.Context, req *connect.Request[api.ConsumeVerificationCodeRequest]) (*connect.Response[api.ConsumeVerificationCodeResponse], error) {
	ctx, span := tracer.Start(ctx, "auth:ConsumeVerificationCode")
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not start transaction")
		return nil, err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	email := req.Msg.GetEmail()
	providedCode := req.Msg.GetProvidedCode()

	err = s.verifyAndDeleteCode(ctx, txqry, email, providedCode)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to verify or delete verification code")
		return nil, err
	}
	token, err := s.createToken(ctx, txqry, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create token")
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return nil, err
	}

	return &connect.Response[api.ConsumeVerificationCodeResponse]{
		Msg: &api.ConsumeVerificationCodeResponse{
			Token: token,
		},
	}, nil
}

func (s Service) VerifyToken(ctx context.Context, req *connect.Request[api.VerifyTokenRequest]) (*connect.Response[api.VerifyTokenResponse], error) {
	ctx, span := tracer.Start(ctx, "auth:VerifyToken")
	defer span.End()

	user, err := s.verifier.VerifyToken(ctx, req.Msg.GetToken())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.VerifyTokenResponse]{
		Msg: &api.VerifyTokenResponse{
			Email: user.Email,
		},
	}, nil
}
