package vcsmoodlev1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
)

var (
	moodleServiceTracer = otel.Tracer("vcassist.services.vcsmoodle.v1.MoodleService")
)

type InstrumentedMoodleServiceClient struct {
	inner MoodleServiceClient
}

func NewInstrumentedMoodleServiceClient(inner MoodleServiceClient) InstrumentedMoodleServiceClient {
	return InstrumentedMoodleServiceClient{inner: inner}
}

func (c InstrumentedMoodleServiceClient) GetAuthStatus(ctx context.Context, req *connect.Request[v1.GetAuthStatusRequest]) (*connect.Response[v1.GetAuthStatusResponse], error) {
	ctx, span := moodleServiceTracer.Start(ctx, "GetAuthStatus")
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

	res, err := c.inner.GetAuthStatus(ctx, req)
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

func (c InstrumentedMoodleServiceClient) ProvideUsernamePassword(ctx context.Context, req *connect.Request[v1.ProvideUsernamePasswordRequest]) (*connect.Response[v1.ProvideUsernamePasswordResponse], error) {
	ctx, span := moodleServiceTracer.Start(ctx, "ProvideUsernamePassword")
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

	res, err := c.inner.ProvideUsernamePassword(ctx, req)
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

func (c InstrumentedMoodleServiceClient) GetStudentData(ctx context.Context, req *connect.Request[v1.GetStudentDataRequest]) (*connect.Response[v1.GetStudentDataResponse], error) {
	ctx, span := moodleServiceTracer.Start(ctx, "GetStudentData")
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

	res, err := c.inner.GetStudentData(ctx, req)
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

