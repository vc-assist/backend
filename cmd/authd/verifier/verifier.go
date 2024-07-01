package verifier

import (
	"context"
	"database/sql"
	"fmt"
	"vcassist-backend/cmd/authd/db"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("verifier")

type Verifier struct {
	qry *db.Queries
}

func NewVerifier(database *sql.DB) Verifier {
	return Verifier{qry: db.New(database)}
}

var InvalidToken = fmt.Errorf("invalid token")

func (v Verifier) VerifyToken(ctx context.Context, token string) (db.User, error) {
	ctx, span := tracer.Start(ctx, "verifier:VerifyToken")
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

	return db.User{Email: email}, nil
}
