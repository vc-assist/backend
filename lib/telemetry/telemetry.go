package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

type Telemetry struct {
	TracerProvider *trace.TracerProvider
	MeterProvider  *metric.MeterProvider
}

func (t Telemetry) Shutdown(ctx context.Context) error {
	errlist := []error{}
	err := t.TracerProvider.Shutdown(ctx)
	if err != nil {
		errlist = append(errlist, err)
	}
	err = t.MeterProvider.Shutdown(ctx)
	if err != nil {
		errlist = append(errlist, err)
	}
	return errors.Join(errlist...)
}

type Config struct {
	ServiceName string `json:"service_name"`

	TracesOtlpGrpcEndpoint string `json:"traces_otlp_grpc_endpoint"`
	TracesOtlpHttpEndpoint string `json:"traces_otlp_http_endpoint"`

	MetricsOtlpGrpcEndpoint string `json:"metrics_otlp_grpc_endpoint"`
	MetricsOtlpHttpEndpoint string `json:"metrics_otlp_http_endpoint"`
}

func Setup(ctx context.Context, config Config) (Telemetry, error) {
	tracerProvider, err := newTraceProvider(ctx, config)
	if err != nil {
		return Telemetry{}, err
	}
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider(ctx, config)
	if err != nil {
		return Telemetry{}, err
	}
	otel.SetMeterProvider(meterProvider)

	return Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	}, nil
}

func newTraceProvider(ctx context.Context, config Config) (*trace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	slog.Info("setting up trace exporter...")
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

func newMeterProvider(ctx context.Context, config Config) (*metric.MeterProvider, error) {
	slog.Info("setting up meter exporter...")
	exporter, err := otlpMeterExportFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	)
	return meterProvider, nil
}

func otlpTracerExportFromConfig(ctx context.Context, c Config) (trace.SpanExporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if c.TracesOtlpGrpcEndpoint != "" {
		return otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpointURL(c.TracesOtlpGrpcEndpoint),
		)
	}
	return otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(c.TracesOtlpHttpEndpoint),
	)
}

func otlpMeterExportFromConfig(ctx context.Context, c Config) (metric.Exporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if c.MetricsOtlpGrpcEndpoint != "" {
		return otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpointURL(c.MetricsOtlpGrpcEndpoint),
		)
	}
	return otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpointURL(c.MetricsOtlpHttpEndpoint),
	)
}
