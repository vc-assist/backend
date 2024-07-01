package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"strings"
	"time"
	"vcassist-backend/cmd/authd/db"
	"vcassist-backend/cmd/authd/verifier"

	"github.com/jordan-wright/email"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("service")

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
		Expiresat: time.Now().Add(time.Hour).Unix(),
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

func (s Service) StartLogin(ctx context.Context, email string) error {
	ctx, span := tracer.Start(ctx, "auth:StartLogin")
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to begin transaction")
		return err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	err = txqry.EnsureUserExists(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "got unexpected error while ensuring user exists")
		return err
	}
	code, err := s.createVerificationCode(ctx, txqry, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create verification code")
		return err
	}

	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return err
	}

	err = s.sendVerificationCode(ctx, email, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send verification code")
		return err
	}

	return nil
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

func (s Service) ConsumeVerificationCode(ctx context.Context, email, providedCode string) (token string, err error) {
	ctx, span := tracer.Start(ctx, "auth:ConsumeVerificationCode")
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not start transaction")
		return "", err
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)

	err = s.verifyAndDeleteCode(ctx, txqry, email, providedCode)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to verify or delete verification code")
		return "", err
	}
	token, err = s.createToken(ctx, txqry, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create token")
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return "", err
	}

	return token, nil
}

func (s Service) VerifyToken(ctx context.Context, token string) (db.User, error) {
	return s.verifier.VerifyToken(ctx, token)
}
