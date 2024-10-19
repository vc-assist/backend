package linkerv1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/linker/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	LinkerServiceTracer TracerLike = otel.Tracer("vcassist.services.linker.v1.LinkerService")
)

type InstrumentedLinkerServiceClient struct {
	inner           LinkerServiceClient
	WithInputOutput bool
}

func NewInstrumentedLinkerServiceClient(inner LinkerServiceClient) InstrumentedLinkerServiceClient {
	return InstrumentedLinkerServiceClient{inner: inner}
}

func (c InstrumentedLinkerServiceClient) GetExplicitLinks(ctx context.Context, req *connect.Request[v1.GetExplicitLinksRequest]) (*connect.Response[v1.GetExplicitLinksResponse], error) {

	res, err := c.inner.GetExplicitLinks(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) AddExplicitLink(ctx context.Context, req *connect.Request[v1.AddExplicitLinkRequest]) (*connect.Response[v1.AddExplicitLinkResponse], error) {

	res, err := c.inner.AddExplicitLink(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) DeleteExplicitLink(ctx context.Context, req *connect.Request[v1.DeleteExplicitLinkRequest]) (*connect.Response[v1.DeleteExplicitLinkResponse], error) {

	res, err := c.inner.DeleteExplicitLink(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) GetKnownSets(ctx context.Context, req *connect.Request[v1.GetKnownSetsRequest]) (*connect.Response[v1.GetKnownSetsResponse], error) {

	res, err := c.inner.GetKnownSets(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) GetKnownKeys(ctx context.Context, req *connect.Request[v1.GetKnownKeysRequest]) (*connect.Response[v1.GetKnownKeysResponse], error) {

	res, err := c.inner.GetKnownKeys(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) DeleteKnownSets(ctx context.Context, req *connect.Request[v1.DeleteKnownSetsRequest]) (*connect.Response[v1.DeleteKnownSetsResponse], error) {

	res, err := c.inner.DeleteKnownSets(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) DeleteKnownKeys(ctx context.Context, req *connect.Request[v1.DeleteKnownKeysRequest]) (*connect.Response[v1.DeleteKnownKeysResponse], error) {

	res, err := c.inner.DeleteKnownKeys(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) Link(ctx context.Context, req *connect.Request[v1.LinkRequest]) (*connect.Response[v1.LinkResponse], error) {

	res, err := c.inner.Link(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedLinkerServiceClient) SuggestLinks(ctx context.Context, req *connect.Request[v1.SuggestLinksRequest]) (*connect.Response[v1.SuggestLinksResponse], error) {

	res, err := c.inner.SuggestLinks(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}
