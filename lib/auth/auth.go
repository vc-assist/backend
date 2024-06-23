package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"strings"
	"time"
	"vcassist-backend/lib/auth/db"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/net/context"
)

var tracer = otel.Tracer("auth")

type AuthService struct {
	db    *sql.DB
	qry   *db.Queries
	email EmailConfig
}

func NewAuthService(config Config) (AuthService, error) {
	sqlite, err := config.Libsql.OpenDB()
	if err != nil {
		return AuthService{}, err
	}
	return AuthService{
		qry:   db.New(sqlite),
		email: config.Email,
	}, nil
}

var InvalidToken = fmt.Errorf("invalid token")

func (s AuthService) VerifyToken(ctx context.Context, token string) (db.User, error) {
	ctx, span := tracer.Start(ctx, "auth:VerifyToken")
	defer span.End()

	email, err := s.qry.GetUserFromToken(ctx, token)
	if sql.ErrNoRows == err {
		span.SetStatus(codes.Error, "invalid token")
		return db.User{}, InvalidToken
	} else if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "got unexpected error while reading token")
		return db.User{}, err
	}

	return db.User{Email: email}, nil
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

func (s AuthService) createVerificationCode(ctx context.Context, txqry *db.Queries, email string) (code string, err error) {
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

func (s AuthService) sendVerificationCode(ctx context.Context, email, code string) error {
	ctx, span := tracer.Start(ctx, "auth:sendVerificationCode")
	defer span.End()

	auth := smtp.PlainAuth("", s.email.Address, s.email.Password, s.email.Server)
	client, err := smtp.Dial(fmt.Sprintf("%s:%d", s.email.Server, s.email.Port))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to connect to SMTP server")
		return err
	}
	defer client.Close()

	err = client.Auth(auth)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to authenticate email client")
		return err
	}
	err = client.Rcpt(email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to set client recipient")
		return err
	}
	writer, err := client.Data()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to open body writer")
		return err
	}

	body := fmt.Sprintf(`Please enter the following verification code for you VC Assist account when prompted.

%s

If you don't recognize this account, please ignore this email.`, code)
	_, err = writer.Write([]byte(body))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to write body")
		return err
	}
	err = writer.Close()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to close body")
		return err
	}

	err = client.Quit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to close client")
		return err
	}

	return nil
}

func (s AuthService) StartLogin(ctx context.Context, email string) error {
	ctx, span := tracer.Start(ctx, "auth:CreateOrLogin")
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

func (s AuthService) verifyAndDeleteCode(ctx context.Context, txqry *db.Queries, email, code string) error {
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

func (s AuthService) createToken(ctx context.Context, txqry *db.Queries, email string) (string, error) {
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

func (s AuthService) ConsumeVerificationCode(ctx context.Context, email, providedCode string) (token string, err error) {
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
