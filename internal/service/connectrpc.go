package service

import (
	"context"

	"connectrpc.com/connect"
)

type authInterceptor = func(ctx context.Context, token string) (context.Context, error)

type genericAuthInterceptor struct {
	fn authInterceptor
}

func newGenericAuthInterceptor(fn authInterceptor) genericAuthInterceptor {
	return genericAuthInterceptor{fn: fn}
}

func (a genericAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var err error
		ctx, err = a.fn(ctx, req.Header().Get("Authorization"))
		if err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (a genericAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a genericAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, shc connect.StreamingHandlerConn) error {
		var err error
		ctx, err = a.fn(ctx, shc.RequestHeader().Get("Authorization"))
		if err != nil {
			return err
		}
		return next(ctx, shc)
	}
}
