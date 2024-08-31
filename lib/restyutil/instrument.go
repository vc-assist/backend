package restyutil

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv/v1.13.0/httpconv"
	"go.opentelemetry.io/otel/trace"
)

type InstrumentOutput interface {
	Write(id string, contents string)
}

type instrumentCtx struct {
	output    InstrumentOutput
	tracer    trace.Tracer
	idcounter *uint64
}

// `tracer` can be nil, it will default to a library name of "resty"
// `output` can also be nil, if it is, then the function is a no-op
func InstrumentClient(client *resty.Client, tracer trace.Tracer, output InstrumentOutput) {
	if output == nil {
		return
	}
	if tracer == nil {
		tracer = otel.Tracer("resty")
	}

	var idcounter uint64
	i := instrumentCtx{output: output, tracer: tracer, idcounter: &idcounter}
	client.OnBeforeRequest(i.onBeforeRequest(i.tracer))
	client.OnAfterResponse(i.onAfterResponse)
	client.OnError(i.onError)
}

const messageIdContextKey = "vcassist.restyutil.instrument.message_id"

func (i instrumentCtx) onBeforeRequest(tracer trace.Tracer) resty.RequestMiddleware {
	return func(cli *resty.Client, req *resty.Request) error {
		ctx, _ := tracer.Start(req.Context(), req.Method)

		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			messageId := strconv.FormatUint(atomic.AddUint64(i.idcounter, 1), 10)
			slog.DebugContext(
				ctx, "start request",
				"method", req.Method,
				"url", req.URL,
				"message_id", messageId,
			)
			ctx = context.WithValue(ctx, messageIdContextKey, messageId)
		}

		req.SetContext(ctx)
		return nil
	}
}

func (i instrumentCtx) onAfterResponse(_ *resty.Client, res *resty.Response) error {
	ctx := res.Request.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()

	span.SetAttributes(httpconv.ClientResponse(res.RawResponse)...)

	// setting request attributes here since res.Request.RawRequest is nil in onBeforeRequest
	span.SetName(fmt.Sprintf("http %s", res.Request.Method))
	span.SetAttributes(httpconv.ClientRequest(res.Request.RawRequest)...)

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		messageId, ok := ctx.Value(messageIdContextKey).(string)
		if !ok {
			panic("failed to retrieve message_id from context")
		}
		i.output.Write(messageId, formatHttpMessage(res))
		slog.DebugContext(
			ctx, "request succeeded",
			"method", res.Request.Method,
			"url", res.Request.URL,
			"message_id", messageId,
		)
	}

	return nil
}

func (i instrumentCtx) onError(req *resty.Request, err error) {
	ctx := req.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()

	defer span.RecordError(err)
	defer span.SetStatus(codes.Error, "request failed")

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		messageId, ok := ctx.Value(messageIdContextKey).(string)
		if ok {
			panic("failed to retrieve message_id from context")
		}
		slog.ErrorContext(
			req.Context(), "request failed",
			"method", req.Method,
			"url", req.URL,
			"err", err,
			"message_id", messageId,
		)
	} else {
		slog.ErrorContext(
			req.Context(), "request failed",
			"method", req.Method,
			"url", req.URL,
			"err", err,
		)
	}

	span.SetName(fmt.Sprintf("http %s", req.Method))
	if req.RawRequest == nil {
		return
	}
	span.SetAttributes(httpconv.ClientRequest(req.RawRequest)...)
}
