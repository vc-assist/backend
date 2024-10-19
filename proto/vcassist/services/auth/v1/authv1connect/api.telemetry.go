package authv1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/auth/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	AuthServiceTracer TracerLike = otel.Tracer("vcassist.services.auth.v1.AuthService")
)

type InstrumentedAuthServiceClient struct {
	inner           AuthServiceClient
	WithInputOutput bool
}

func NewInstrumentedAuthServiceClient(inner AuthServiceClient) InstrumentedAuthServiceClient {
	return InstrumentedAuthServiceClient{inner: inner}
}

func (c InstrumentedAuthServiceClient) StartLogin(ctx context.Context, req *connect.Request[v1.StartLoginRequest]) (*connect.Response[v1.StartLoginResponse], error) {

	res, err := c.inner.StartLogin(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedAuthServiceClient) ConsumeVerificationCode(ctx context.Context, req *connect.Request[v1.ConsumeVerificationCodeRequest]) (*connect.Response[v1.ConsumeVerificationCodeResponse], error) {

	res, err := c.inner.ConsumeVerificationCode(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedAuthServiceClient) VerifyToken(ctx context.Context, req *connect.Request[v1.VerifyTokenRequest]) (*connect.Response[v1.VerifyTokenResponse], error) {

	res, err := c.inner.VerifyToken(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}
