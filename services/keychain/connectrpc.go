package keychain

/* This file is the middleware that adds the sessionToken from the frontend to the current go context to be used in vcsis (returns powerschool)
Authored by Justin Shi and Shengzhi Hu CO 2025 */

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const moodleCtxKey = "vcassist:moodle" //usernamepwd id 
const powerschoolCtxKey = "vcassist:powerschool" //oauth id  or token 

type AuthInterceptor struct {
	db     *sql.DB
	qry    *db.Queries
}

//this function takes in the request from the frontend and modifies it to have the right session token context
func (i AuthInterceptor ) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
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
			return nil, err
		}

		ctx = context.WithValue(ctx, sessionCtxKey, user)
		return next(ctx, req)
	}
}
 
func ProfileFromContext(ctx context.Context) db.User {
	span := trace.SpanFromContext(ctx)
	profile, ok := ctx.Value(profileCtxKey).(db.User)
	if !ok {
		panic("user ctx is not set")
	}
	if profile.Email == "" {
		panic("empty profile email")
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "profile:email",
		Value: attribute.StringValue(profile.Email),
	})
	return profile
}

//return the token for scraping 
func PowerschoolFromContext(ctx context.Context) (string, error) {
	span := trace.SpanFromContext(ctx)
	token, err := ctx.Value(powerschoolCtxKey).GetToken
	if err != nil {

	}
}
 //returns moodle username and password -> scrape 
func usernampasswordfromcontexr 