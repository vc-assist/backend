package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var globalTracerProvider *trace.TracerProvider

func Shutdown(ctx context.Context) error {
	return globalTracerProvider.Shutdown(ctx)
}

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span)
}

// A wrapper around `trace.Tracer` from `go.opentelemetry.io/otel/trace`
// that formats methods like `service.<span>`
type wrappedTracer struct {
	libraryName string
	tracer      oteltrace.Tracer
}

func (w *wrappedTracer) getTracer() oteltrace.Tracer {
	if w.tracer != nil {
		return w.tracer
	}
	w.tracer = globalTracerProvider.Tracer(w.libraryName)
	return w.tracer
}

func (w *wrappedTracer) Start(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return w.Start(ctx, fmt.Sprintf("%s.%s", w.libraryName, spanName), opts...)
}

func Tracer(libraryName string) TracerLike {
	return &wrappedTracer{libraryName: libraryName}
}

func Setup(ctx context.Context, serviceName string, config config) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	r, err := newResource(serviceName)
	if err != nil {
		return err
	}

	tracerProvider, err := newTraceProvider(ctx, r, config)
	if err != nil {
		return err
	}
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMetricProvider(ctx, r, config)
	if err != nil {
		return err
	}
	otel.SetMeterProvider(meterProvider)

	globalTracerProvider = tracerProvider

	return nil
}
