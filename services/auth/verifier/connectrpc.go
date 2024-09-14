package verifier

import (
	"context"
	"fmt"
	"strings"
	"vcassist-backend/services/auth/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const profileCtxKey = "vcassist:profile"

type AuthInterceptor struct {
	verifier Verifier
}

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
		user, err := i.verifier.VerifyToken(ctx, token)
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, profileCtxKey, user)
		return next(ctx, req)
	}
}

func (i AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		authHeader := conn.RequestHeader().Get("Authorization")
		split := strings.Split(authHeader, " ")
		if len(split) < 2 {
			return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("Unauthorized"))
		}
		token := split[1]
		user, err := i.verifier.VerifyToken(ctx, token)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, profileCtxKey, user)
		return next(ctx, conn)
	}
}

func (i AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func NewAuthInterceptor(verifier Verifier) AuthInterceptor {
	return AuthInterceptor{verifier: verifier}
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
