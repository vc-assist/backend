package telemetry

import (
	"fmt"
	"io"
	"net/http"
	"strings"

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

func instrumentRequestHeaders(out *[]attribute.KeyValue, headers http.Header) {
	for header, values := range headers {
		if len(values) == 1 {
			*out = append(*out, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("request/header: %s", header)),
				Value: attribute.StringValue(values[0]),
			})
			continue
		}
		for i, v := range values {
			*out = append(*out, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("request/header: %s (%d)", header, i)),
				Value: attribute.StringValue(v),
			})
		}
	}
}

func instrumentResponseHeaders(out *[]attribute.KeyValue, headers http.Header) {
	for header, values := range headers {
		if len(values) == 1 {
			*out = append(*out, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("response/header: %s", header)),
				Value: attribute.StringValue(values[0]),
			})
			continue
		}
		for i, v := range values {
			*out = append(*out, attribute.KeyValue{
				Key:   attribute.Key(fmt.Sprintf("response/header: %s (%d)", header, i)),
				Value: attribute.StringValue(v),
			})
		}
	}
}

func instrumentRequestBody(span trace.Span, req *http.Request) {
	reqbodyReader, err := req.GetBody()
	if err != nil {
		span.SetAttributes(attribute.KeyValue{
			Key: "request/body",
			Value: attribute.StringValue(fmt.Sprintf(
				"failed to get request body: %s",
				err.Error(),
			)),
		})
	} else if reqbodyReader != nil {
		reqbody, err := io.ReadAll(reqbodyReader)
		if err != nil {
			span.SetAttributes(attribute.KeyValue{
				Key: "request/body",
				Value: attribute.StringValue(fmt.Sprintf(
					"failed to read request body: %s",
					err.Error(),
				)),
			})
		} else {
			span.SetAttributes(attribute.KeyValue{
				Key:   "request/body",
				Value: attribute.StringValue(string(reqbody)),
			})
		}
	}
}

func onAfterResponse(_ *resty.Client, res *resty.Response) error {
	span := trace.SpanFromContext(res.Request.Context())
	defer span.End()

	span.SetAttributes(httpconv.ClientResponse(res.RawResponse)...)

	// setting request attributes here since res.Request.RawRequest is nil in onBeforeRequest
	span.SetName(fmt.Sprintf("http %s", res.Request.Method))
	span.SetAttributes(httpconv.ClientRequest(res.Request.RawRequest)...)

	var attrs []attribute.KeyValue
	instrumentRequestHeaders(&attrs, res.Request.Header)
	instrumentResponseHeaders(&attrs, res.Header())
	span.SetAttributes(attrs...)

	instrumentRequestBody(span, res.Request.RawRequest)
	span.SetAttributes(attribute.KeyValue{
		Key:   "response/body",
		Value: attribute.StringValue(res.String()),
	})

	return nil
}

func onError(req *resty.Request, err error) {
	span := trace.SpanFromContext(req.Context())
	defer span.End()

	if strings.Contains(err.Error(), "login successful") {
		defer span.SetStatus(codes.Ok, "error bypassed: moodle login successful")
	} else {
		defer span.SetStatus(codes.Error, err.Error())
		defer span.RecordError(err)
	}

	span.SetName(fmt.Sprintf("http %s", req.Method))
	var attrs []attribute.KeyValue
	instrumentRequestHeaders(&attrs, req.Header)
	span.SetAttributes(attrs...)

	if req.RawRequest == nil {
		return
	}
	span.SetAttributes(httpconv.ClientRequest(req.RawRequest)...)
	instrumentRequestBody(span, req.RawRequest)
}
