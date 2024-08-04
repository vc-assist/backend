package powerservicev1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
)

var (
	powerschoolServiceTracer = otel.Tracer("vcassist.services.powerservice.v1.PowerschoolService")
)

type InstrumentedPowerschoolServiceClient struct {
	inner PowerschoolServiceClient
}

func NewInstrumentedPowerschoolServiceClient(inner PowerschoolServiceClient) InstrumentedPowerschoolServiceClient {
	return InstrumentedPowerschoolServiceClient{inner: inner}
}

func (c InstrumentedPowerschoolServiceClient) GetAuthStatus(ctx context.Context, req *connect.Request[v1.GetAuthStatusRequest]) (*connect.Response[v1.GetAuthStatusResponse], error) {
	ctx, span := powerschoolServiceTracer.Start(ctx, "GetAuthStatus")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(attribute.String("procedure", req.Spec().Procedure))

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

func (c InstrumentedPowerschoolServiceClient) GetOAuthFlow(ctx context.Context, req *connect.Request[v1.GetOAuthFlowRequest]) (*connect.Response[v1.GetOAuthFlowResponse], error) {
	ctx, span := powerschoolServiceTracer.Start(ctx, "GetOAuthFlow")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(attribute.String("procedure", req.Spec().Procedure))

		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.GetOAuthFlow(ctx, req)
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

func (c InstrumentedPowerschoolServiceClient) ProvideOAuth(ctx context.Context, req *connect.Request[v1.ProvideOAuthRequest]) (*connect.Response[v1.ProvideOAuthResponse], error) {
	ctx, span := powerschoolServiceTracer.Start(ctx, "ProvideOAuth")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(attribute.String("procedure", req.Spec().Procedure))

		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.ProvideOAuth(ctx, req)
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

func (c InstrumentedPowerschoolServiceClient) GetStudentData(ctx context.Context, req *connect.Request[v1.GetStudentDataRequest]) (*connect.Response[v1.GetStudentDataResponse], error) {
	ctx, span := powerschoolServiceTracer.Start(ctx, "GetStudentData")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(attribute.String("procedure", req.Spec().Procedure))

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

func (c InstrumentedPowerschoolServiceClient) GetKnownCourses(ctx context.Context, req *connect.Request[v1.GetKnownCoursesRequest]) (*connect.Response[v1.GetKnownCoursesResponse], error) {
	ctx, span := powerschoolServiceTracer.Start(ctx, "GetKnownCourses")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(attribute.String("procedure", req.Spec().Procedure))

		input, err := protojson.Marshal(req.Msg)
		if err == nil {
			span.SetAttributes(attribute.String("input", string(input)))
		} else {
			span.SetAttributes(attribute.String("input", "ERROR: FAILED TO SERIALIZE"))
			span.RecordError(err)
		}
	}

	res, err := c.inner.GetKnownCourses(ctx, req)
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

