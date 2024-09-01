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
	inner GradeSnapshotsServiceClient
}

func NewInstrumentedGradeSnapshotsServiceClient(inner GradeSnapshotsServiceClient) InstrumentedGradeSnapshotsServiceClient {
	return InstrumentedGradeSnapshotsServiceClient{inner: inner}
}

func (c InstrumentedGradeSnapshotsServiceClient) Push(ctx context.Context, req *connect.Request[v1.PushRequest]) (*connect.Response[v1.PushResponse], error) {
	ctx, span := GradeSnapshotsServiceTracer.Start(ctx, "Push")
	defer span.End()

	if span.IsRecording() {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.Push(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() {
		output, err := protojson.Marshal(res.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("output", string(output)))
		} else {
			span.SetAttributes(attribute.String("output", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	return res, nil
}

func (c InstrumentedGradeSnapshotsServiceClient) Pull(ctx context.Context, req *connect.Request[v1.PullRequest]) (*connect.Response[v1.PullResponse], error) {
	ctx, span := GradeSnapshotsServiceTracer.Start(ctx, "Pull")
	defer span.End()

	if span.IsRecording() {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.Pull(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() {
		output, err := protojson.Marshal(res.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("output", string(output)))
		} else {
			span.SetAttributes(attribute.String("output", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	return res, nil
}
