package globals

import (
	"context"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
)

const key = "linker-cli.ctx"

type Value struct {
	Client linkerv1connect.LinkerServiceClient
}

func Set(ctx context.Context, value *Value) context.Context {
	return context.WithValue(ctx, key, value)
}

func Get(ctx context.Context) *Value {
	return ctx.Value(key).(*Value)
}
