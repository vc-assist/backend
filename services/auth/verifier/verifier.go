package verifier

import (
	"context"
	"database/sql"
	"fmt"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/timezone"
	"vcassist-backend/services/auth/db"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = telemetry.Tracer("vcassist.services.auth.verifier")
var meter = otel.Meter("vcassist.services.auth.verifier")

var uniqueLoginCounter, _ = meter.Int64Counter("auth_service.unique_login_counter")
var loggedInToday = map[string]struct{}{}
var todaysDate = timezone.Now().Day()

func countLogin(ctx context.Context, email string) {
	if timezone.Now().Day() != todaysDate {
		loggedInToday = map[string]struct{}{}
	}
	_, alreadyLoggedIn := loggedInToday[email]
	if alreadyLoggedIn {
		return
	}
	uniqueLoginCounter.Add(ctx, 1)
	loggedInToday[email] = struct{}{}
}

type Verifier struct {
	qry *db.Queries
}

func NewVerifier(database *sql.DB) Verifier {
	return Verifier{qry: db.New(database)}
}

var InvalidToken = fmt.Errorf("invalid token")

func (v Verifier) VerifyToken(ctx context.Context, token string) (db.User, error) {
	ctx, span := tracer.Start(ctx, "VerifyToken")
	defer span.End()

	email, err := v.qry.GetUserFromToken(ctx, token)
	if sql.ErrNoRows == err {
		span.SetStatus(codes.Error, "invalid token")
		return db.User{}, InvalidToken
	} else if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "got unexpected error while reading token")
		return db.User{}, err
	}

	countLogin(ctx, email)

	return db.User{Email: email}, nil
}
