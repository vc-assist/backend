package telemetry

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv/v1.13.0/httpconv"
	"go.opentelemetry.io/otel/trace"
)

var propagator = otel.GetTextMapPropagator()

func InstrumentResty(client *resty.Client, tracerName string) {
	tracer := otel.Tracer(tracerName)

	client.OnBeforeRequest(onBeforeRequest(tracer))
	client.OnAfterResponse(onAfterResponse)
	client.OnError(onError)

}

func onBeforeRequest(tracer trace.Tracer) resty.RequestMiddleware {
	return func(cli *resty.Client, req *resty.Request) error {
		ctx, _ := tracer.Start(req.Context(), req.Method)
		req.SetContext(ctx)
		return nil
	}
}

func onAfterResponse(_ *resty.Client, res *resty.Response) error {
	span := trace.SpanFromContext(res.Request.Context())

	span.SetAttributes(httpconv.ClientResponse(res.RawResponse)...)

	// Setting request attributes here since res.Request.RawRequest is nil in onBeforeRequest.
	span.SetName(fmt.Sprintf("http %s", res.Request.Method))
	span.SetAttributes(httpconv.ClientRequest(res.Request.RawRequest)...)

	attrs := []attribute.KeyValue{}
	for header, values := range res.Request.Header {
		if len(values) == 1 {
			attrs = append(attrs, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("request/header: %s", header)),
				Value: attribute.StringValue(values[0]),
			})
			continue
		}
		for i, v := range values {
			attrs = append(attrs, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("request/header: %s (%d)", header, i)),
				Value: attribute.StringValue(v),
			})
		}
	}

	for header, values := range res.Header() {
		if len(values) == 1 {
			attrs = append(attrs, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("response/header: %s", header)),
				Value: attribute.StringValue(values[0]),
			})
			continue
		}
		for i, v := range values {
			attrs = append(attrs, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("response/header: %s (%d)", header, i)),
				Value: attribute.StringValue(v),
			})
		}
	}

	span.SetAttributes(attribute.KeyValue{
		Key:   "response/body",
		Value: attribute.StringValue(res.String()),
	})
	span.SetAttributes(attrs...)

	span.End()
	return nil
}

func onError(req *resty.Request, err error) {
	span := trace.SpanFromContext(req.Context())

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetName(fmt.Sprintf("http %s", req.Method))
	span.SetAttributes(httpconv.ClientRequest(req.RawRequest)...)

	span.End()
}
