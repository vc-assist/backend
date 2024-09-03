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

func NewAuthInterceptor(verifier Verifier) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			authHeader := req.Header().Get("Authorization")
			split := strings.Split(authHeader, " ")
			if len(split) < 2 {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("Unauthorized"))
			}

			token := split[1]
			user, err := verifier.VerifyToken(ctx, token)
			if err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, profileCtxKey, user)
			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}

func ProfileFromContext(ctx context.Context) db.User {
	span := trace.SpanFromContext(ctx)
	profile, ok := ctx.Value(profileCtxKey).(db.User)
	if !ok || profile.Email == "" {
		panic("failed to get user from context")
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "profile:email",
		Value: attribute.StringValue(profile.Email),
	})
	return profile
}
