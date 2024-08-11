package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"
	"vcassist-backend/lib/configuration"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
	return errors.Join(errlist...)
}

type OtlpConnConfig struct {
	GrpcEndpoint string            `json:"grpc_endpoint"`
	HttpEndpoint string            `json:"http_endpoint"`
	Headers      map[string]string `json:"headers"`
}

type OtlpConfig struct {
	Traces  OtlpConnConfig `json:"traces"`
	Metrics OtlpConnConfig `json:"metrics"`
}

type Config struct {
	Otlp OtlpConfig `json:"otlp"`
}

var setupTestEnvironments = map[string]bool{}

// sets up telemetry in a testing environment, ensuring that it isn't
// set up more than once
func SetupForTesting(t testing.TB, serviceName string) func() {
	_, setupAlready := setupTestEnvironments[serviceName]
	if setupAlready {
		return func() {}
	}
	ctx := context.WithValue(context.Background(), "telemetry_test_env", struct{}{})
	tel, err := SetupFromEnv(ctx, serviceName)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		err := tel.Shutdown(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// searches up the filesystem from the cwd to find a file
// called telemetry.json5, once found it will then use it
// as a config to setup telemetry
func SetupFromEnv(ctx context.Context, serviceName string) (Telemetry, error) {
	config, err := configuration.ReadRecursively[Config]("telemetry.json5")
	if err != nil {
		return Telemetry{}, err
	}
	return Setup(ctx, serviceName, config)
}

func Setup(ctx context.Context, serviceName string, config Config) (Telemetry, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	r, err := newResource(serviceName)
	if err != nil {
		return Telemetry{}, err
	}

	tracerProvider, err := newTraceProvider(ctx, r, config)
	if err != nil {
		return Telemetry{}, err
	}
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMetricProvider(ctx, r, config)
	if err != nil {
		return Telemetry{}, err
	}
	otel.SetMeterProvider(meterProvider)

	return Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	}, nil
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

func newTraceProvider(ctx context.Context, r *resource.Resource, config Config) (*trace.TracerProvider, error) {
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

func otlpTracerExportFromConfig(ctx context.Context, c Config) (trace.SpanExporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if c.Otlp.Traces.GrpcEndpoint != "" {
		return otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpointURL(c.Otlp.Traces.GrpcEndpoint),
			otlptracegrpc.WithHeaders(c.Otlp.Traces.Headers),
		)
	}
	return otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(c.Otlp.Traces.HttpEndpoint),
		otlptracehttp.WithHeaders(c.Otlp.Traces.Headers),
	)
}

func newMetricProvider(ctx context.Context, r *resource.Resource, config Config) (*metric.MeterProvider, error) {
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

func otlpMetricExportFromConfig(ctx context.Context, c Config) (metric.Exporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if c.Otlp.Metrics.GrpcEndpoint != "" {
		return otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpointURL(c.Otlp.Metrics.GrpcEndpoint),
			otlpmetricgrpc.WithHeaders(c.Otlp.Metrics.Headers),
		)
	}
	return otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpointURL(c.Otlp.Metrics.HttpEndpoint),
		otlpmetrichttp.WithHeaders(c.Otlp.Metrics.Headers),
	)
}
