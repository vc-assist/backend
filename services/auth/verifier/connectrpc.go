package verifier

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

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
			_, err := verifier.VerifyToken(ctx, token)
			if err != nil {
				return nil, err
			}

			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}
