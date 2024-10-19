package gradesnapshotsv1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	GradeSnapshotsServiceTracer TracerLike = otel.Tracer("vcassist.services.gradesnapshots.v1.GradeSnapshotsService")
)

type InstrumentedGradeSnapshotsServiceClient struct {
	inner           GradeSnapshotsServiceClient
	WithInputOutput bool
}

func NewInstrumentedGradeSnapshotsServiceClient(inner GradeSnapshotsServiceClient) InstrumentedGradeSnapshotsServiceClient {
	return InstrumentedGradeSnapshotsServiceClient{inner: inner}
}

func (c InstrumentedGradeSnapshotsServiceClient) Push(ctx context.Context, req *connect.Request[v1.PushRequest]) (*connect.Response[v1.PushResponse], error) {

	res, err := c.inner.Push(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedGradeSnapshotsServiceClient) Pull(ctx context.Context, req *connect.Request[v1.PullRequest]) (*connect.Response[v1.PullResponse], error) {

	res, err := c.inner.Pull(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}
