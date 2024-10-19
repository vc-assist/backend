package keychainv1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/keychain/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	KeychainServiceTracer TracerLike = otel.Tracer("vcassist.services.keychain.v1.KeychainService")
)

type InstrumentedKeychainServiceClient struct {
	inner           KeychainServiceClient
	WithInputOutput bool
}

func NewInstrumentedKeychainServiceClient(inner KeychainServiceClient) InstrumentedKeychainServiceClient {
	return InstrumentedKeychainServiceClient{inner: inner}
}

func (c InstrumentedKeychainServiceClient) SetOAuth(ctx context.Context, req *connect.Request[v1.SetOAuthRequest]) (*connect.Response[v1.SetOAuthResponse], error) {

	res, err := c.inner.SetOAuth(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedKeychainServiceClient) GetOAuth(ctx context.Context, req *connect.Request[v1.GetOAuthRequest]) (*connect.Response[v1.GetOAuthResponse], error) {

	res, err := c.inner.GetOAuth(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedKeychainServiceClient) SetUsernamePassword(ctx context.Context, req *connect.Request[v1.SetUsernamePasswordRequest]) (*connect.Response[v1.SetUsernamePasswordResponse], error) {

	res, err := c.inner.SetUsernamePassword(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedKeychainServiceClient) GetUsernamePassword(ctx context.Context, req *connect.Request[v1.GetUsernamePasswordRequest]) (*connect.Response[v1.GetUsernamePasswordResponse], error) {

	res, err := c.inner.GetUsernamePassword(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}
