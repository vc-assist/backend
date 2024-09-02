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
	inner LinkerServiceClient
	WithInputOutput bool
}

func NewInstrumentedLinkerServiceClient(inner LinkerServiceClient) InstrumentedLinkerServiceClient {
	return InstrumentedLinkerServiceClient{inner: inner}
}

func (c InstrumentedLinkerServiceClient) GetExplicitLinks(ctx context.Context, req *connect.Request[v1.GetExplicitLinksRequest]) (*connect.Response[v1.GetExplicitLinksResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "GetExplicitLinks")
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

	res, err := c.inner.GetExplicitLinks(ctx, req)
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

func (c InstrumentedLinkerServiceClient) AddExplicitLink(ctx context.Context, req *connect.Request[v1.AddExplicitLinkRequest]) (*connect.Response[v1.AddExplicitLinkResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "AddExplicitLink")
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

	res, err := c.inner.AddExplicitLink(ctx, req)
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

func (c InstrumentedLinkerServiceClient) DeleteExplicitLink(ctx context.Context, req *connect.Request[v1.DeleteExplicitLinkRequest]) (*connect.Response[v1.DeleteExplicitLinkResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "DeleteExplicitLink")
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

	res, err := c.inner.DeleteExplicitLink(ctx, req)
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

func (c InstrumentedLinkerServiceClient) GetKnownSets(ctx context.Context, req *connect.Request[v1.GetKnownSetsRequest]) (*connect.Response[v1.GetKnownSetsResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "GetKnownSets")
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

	res, err := c.inner.GetKnownSets(ctx, req)
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

func (c InstrumentedLinkerServiceClient) GetKnownKeys(ctx context.Context, req *connect.Request[v1.GetKnownKeysRequest]) (*connect.Response[v1.GetKnownKeysResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "GetKnownKeys")
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

	res, err := c.inner.GetKnownKeys(ctx, req)
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

func (c InstrumentedLinkerServiceClient) DeleteKnownSets(ctx context.Context, req *connect.Request[v1.DeleteKnownSetsRequest]) (*connect.Response[v1.DeleteKnownSetsResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "DeleteKnownSets")
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

	res, err := c.inner.DeleteKnownSets(ctx, req)
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

func (c InstrumentedLinkerServiceClient) DeleteKnownKeys(ctx context.Context, req *connect.Request[v1.DeleteKnownKeysRequest]) (*connect.Response[v1.DeleteKnownKeysResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "DeleteKnownKeys")
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

	res, err := c.inner.DeleteKnownKeys(ctx, req)
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

func (c InstrumentedLinkerServiceClient) Link(ctx context.Context, req *connect.Request[v1.LinkRequest]) (*connect.Response[v1.LinkResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "Link")
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

	res, err := c.inner.Link(ctx, req)
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

func (c InstrumentedLinkerServiceClient) SuggestLinks(ctx context.Context, req *connect.Request[v1.SuggestLinksRequest]) (*connect.Response[v1.SuggestLinksResponse], error) {
	ctx, span := LinkerServiceTracer.Start(ctx, "SuggestLinks")
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

	res, err := c.inner.SuggestLinks(ctx, req)
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

