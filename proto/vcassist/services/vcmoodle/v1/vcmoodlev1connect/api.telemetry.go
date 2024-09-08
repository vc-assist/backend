package vcmoodlev1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/vcmoodle/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	MoodleServiceTracer TracerLike = otel.Tracer("vcassist.services.vcmoodle.v1.MoodleService")
)

type InstrumentedMoodleServiceClient struct {
	inner MoodleServiceClient
	WithInputOutput bool
}

func NewInstrumentedMoodleServiceClient(inner MoodleServiceClient) InstrumentedMoodleServiceClient {
	return InstrumentedMoodleServiceClient{inner: inner}
}

func (c InstrumentedMoodleServiceClient) GetAuthStatus(ctx context.Context, req *connect.Request[v1.GetAuthStatusRequest]) (*connect.Response[v1.GetAuthStatusResponse], error) {
	ctx, span := MoodleServiceTracer.Start(ctx, "GetAuthStatus")
	defer span.End()

	if span.IsRecording() && c.WithInputOutput {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.GetAuthStatus(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() && c.WithInputOutput {
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

func (c InstrumentedMoodleServiceClient) ProvideUsernamePassword(ctx context.Context, req *connect.Request[v1.ProvideUsernamePasswordRequest]) (*connect.Response[v1.ProvideUsernamePasswordResponse], error) {
	ctx, span := MoodleServiceTracer.Start(ctx, "ProvideUsernamePassword")
	defer span.End()

	if span.IsRecording() && c.WithInputOutput {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.ProvideUsernamePassword(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() && c.WithInputOutput {
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

func (c InstrumentedMoodleServiceClient) GetCourses(ctx context.Context, req *connect.Request[v1.GetCoursesRequest]) (*connect.Response[v1.GetCoursesResponse], error) {
	ctx, span := MoodleServiceTracer.Start(ctx, "GetCourses")
	defer span.End()

	if span.IsRecording() && c.WithInputOutput {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.GetCourses(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() && c.WithInputOutput {
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

func (c InstrumentedMoodleServiceClient) RefreshCourses(ctx context.Context, req *connect.Request[v1.RefreshCoursesRequest]) (*connect.Response[v1.RefreshCoursesResponse], error) {
	ctx, span := MoodleServiceTracer.Start(ctx, "RefreshCourses")
	defer span.End()

	if span.IsRecording() && c.WithInputOutput {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.RefreshCourses(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() && c.WithInputOutput {
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

func (c InstrumentedMoodleServiceClient) GetChapterContent(ctx context.Context, req *connect.Request[v1.GetChapterContentRequest]) (*connect.Response[v1.GetChapterContentResponse], error) {
	ctx, span := MoodleServiceTracer.Start(ctx, "GetChapterContent")
	defer span.End()

	if span.IsRecording() && c.WithInputOutput {
		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.GetChapterContent(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if span.IsRecording() && c.WithInputOutput {
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

