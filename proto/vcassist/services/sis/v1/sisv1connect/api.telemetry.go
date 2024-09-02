package sisv1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/sis/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	SIServiceTracer TracerLike = otel.Tracer("vcassist.services.sis.v1.SIService")
)

type InstrumentedSIServiceClient struct {
	inner SIServiceClient
	WithInputOutput bool
}

func NewInstrumentedSIServiceClient(inner SIServiceClient) InstrumentedSIServiceClient {
	return InstrumentedSIServiceClient{inner: inner}
}

func (c InstrumentedSIServiceClient) GetCredentialStatus(ctx context.Context, req *connect.Request[v1.GetCredentialStatusRequest]) (*connect.Response[v1.GetCredentialStatusResponse], error) {
	ctx, span := SIServiceTracer.Start(ctx, "GetCredentialStatus")
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

	res, err := c.inner.GetCredentialStatus(ctx, req)
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

func (c InstrumentedSIServiceClient) ProvideCredential(ctx context.Context, req *connect.Request[v1.ProvideCredentialRequest]) (*connect.Response[v1.ProvideCredentialResponse], error) {
	ctx, span := SIServiceTracer.Start(ctx, "ProvideCredential")
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

	res, err := c.inner.ProvideCredential(ctx, req)
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

func (c InstrumentedSIServiceClient) GetData(ctx context.Context, req *connect.Request[v1.GetDataRequest]) (*connect.Response[v1.GetDataResponse], error) {
	ctx, span := SIServiceTracer.Start(ctx, "GetData")
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

	res, err := c.inner.GetData(ctx, req)
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

func (c InstrumentedSIServiceClient) RefreshData(ctx context.Context, req *connect.Request[v1.RefreshDataRequest]) (*connect.Response[v1.RefreshDataResponse], error) {
	ctx, span := SIServiceTracer.Start(ctx, "RefreshData")
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

	res, err := c.inner.RefreshData(ctx, req)
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

