package keychain

/* This file is the middleware that adds the sessionToken from the frontend to the current go context to be used in vcsis (returns powerschool)
Authored by Justin Shi and Shengzhi Hu CO 2025 */

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const moodleCtxKey = "vcassist:moodle"           //usernamepwd id
const powerschoolCtxKey = "vcassist:powerschool" //oauth id  or token

type AuthInterceptor struct {
	db  *sql.DB
	qry *db.Queries
}

func NewAuthInterceptor(sqldb *sql.DB) AuthInterceptor {
	return AuthInterceptor{
		db:  sqldb,
		qry: db.New(sqldb),
	}
}

type UserPass struct {
	Username string
	Password string
}

func (i AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	}
}

func (i AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		return next(ctx, s)
	}
}

// this function takes in the request from the frontend and modifies it to have the right session token context
func (i AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		authHeader := req.Header().Get("Authorization")
		split := strings.Split(authHeader, " ")
		if len(split) < 2 {
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("Unauthorized"))
		}

		token := split[1]
		user, err := i.qry.FindSessionToken(ctx, token)
		if err != nil {
			slog.Debug("session token finding went wrong")
			return nil, err
		}
		if user.Usernamepasswordid.Valid {
			user, err := i.qry.GetUsernamePassword(
				ctx,
				db.GetUsernamePasswordParams{
					Namespace: "moodle",
					ID:        user.Usernamepasswordid.Int64,
				},
			)
			if err != nil {
				slog.Debug("Usernamepassword not found")
				return nil, err
			}
			ctx = context.WithValue(ctx, "vcassist:moodle", user)
			return next(ctx, req)
		}
		if user.Oauthid.Valid {
			email, err := i.qry.FindEmailFromOauthId(ctx, user.Oauthid.Int64)
			if err != nil {
				slog.Debug("email from oauthId query went wrong")
			}
			ctx = context.WithValue(ctx, "vcassist:powerschool", email)
			return next(ctx, req)

		}
		//case where both are null
		slog.Error("this case should be inconcievable, but if it is ever reaced somehow the backend tried to login when the user didnt enter anything")
		return nil, fmt.Errorf("Somehow moodle and powerschool are both nil, try again please")
	}
}

// return the token for scraping
func PowerschoolFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	email, ok := ctx.Value(powerschoolCtxKey).(string)
	if !ok {
		slog.Debug("token ctx is not set")
	}
	if email == "" {
		slog.Debug("empty token ")
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "vcassist:powerschool",
		Value: attribute.StringValue(email),
	})
	return email
}

// returns moodle username and password -> scrape
func UsernamePasswordFromContext(ctx context.Context) UserPass {
	span := trace.SpanFromContext(ctx)
	userPass, ok := ctx.Value(moodleCtxKey).(UserPass) //grab the moodle id
	if !ok {
		slog.Debug("the id ctx is is nonexistent")
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "vcassist:moodle",
		Value: attribute.StringValue(userPass.Username),
	})

	return userPass
}
