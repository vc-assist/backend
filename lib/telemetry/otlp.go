package telemetry

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type otlpConnConfig struct {
	GrpcEndpoint string            `json:"grpc_endpoint"`
	HttpEndpoint string            `json:"http_endpoint"`
	Headers      map[string]string `json:"headers"`
}

type otlpConfig struct {
	Traces  otlpConnConfig `json:"traces"`
	Metrics otlpConnConfig `json:"metrics"`
}

type config struct {
	Otlp otlpConfig `json:"otlp"`
}

func newResource(serviceName string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
}

func newTraceProvider(ctx context.Context, r *resource.Resource, config config) (*trace.TracerProvider, error) {
	exporter, err := otlpTracerExportFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(r),
	)
	return traceProvider, nil
}

func otlpTracerExportFromConfig(ctx context.Context, c config) (trace.SpanExporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if c.Otlp.Traces.GrpcEndpoint != "" {
		slog.Info(
			"tracer export initialized",
			"type", "grpc",
			"endpoint", c.Otlp.Traces.GrpcEndpoint,
			"headers", len(c.Otlp.Traces.Headers) > 0,
		)
		return otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpointURL(c.Otlp.Traces.GrpcEndpoint),
			otlptracegrpc.WithHeaders(c.Otlp.Traces.Headers),
		)
	}

	slog.Info(
		"tracer export initialized",
		"type", "http",
		"endpoint", c.Otlp.Traces.HttpEndpoint,
		"headers", len(c.Otlp.Traces.Headers) > 0,
	)
	return otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(c.Otlp.Traces.HttpEndpoint),
		otlptracehttp.WithHeaders(c.Otlp.Traces.Headers),
	)
}

func newMetricProvider(ctx context.Context, r *resource.Resource, config config) (*metric.MeterProvider, error) {
	exporter, err := otlpMetricExportFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(time.Second*5))),
		metric.WithResource(r),
	)
	return provider, nil
}

func otlpMetricExportFromConfig(ctx context.Context, c config) (metric.Exporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if c.Otlp.Metrics.GrpcEndpoint != "" {
		slog.Info(
			"metric exporter initialized",
			"type", "grpc",
			"endpoint", c.Otlp.Metrics.GrpcEndpoint,
			"headers", len(c.Otlp.Metrics.Headers) > 0,
		)
		return otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpointURL(c.Otlp.Metrics.GrpcEndpoint),
			otlpmetricgrpc.WithHeaders(c.Otlp.Metrics.Headers),
		)
	}
	slog.Info(
		"metric exporter initialized",
		"type", "http",
		"endpoint", c.Otlp.Metrics.HttpEndpoint,
		"headers", len(c.Otlp.Metrics.Headers) > 0,
	)
	return otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpointURL(c.Otlp.Metrics.HttpEndpoint),
		otlpmetrichttp.WithHeaders(c.Otlp.Metrics.Headers),
	)
}
