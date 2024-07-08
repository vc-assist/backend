package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"
	"vcassist-backend/lib/configuration"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

type Telemetry struct {
	TracerProvider *trace.TracerProvider
}

func (t Telemetry) Shutdown(ctx context.Context) error {
	errlist := []error{}
	err := t.TracerProvider.Shutdown(ctx)
	if err != nil {
		errlist = append(errlist, err)
	}
	return errors.Join(errlist...)
}

type Config struct {
	TracesOtlpGrpcEndpoint string `json:"traces_otlp_grpc_endpoint"`
	TracesOtlpHttpEndpoint string `json:"traces_otlp_http_endpoint"`
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

func setupPrometheus() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", mux)
	if err != nil {
		slog.Error("failed to setup prometheus", "err", err.Error())
	}
}

func Setup(ctx context.Context, serviceName string, config Config) (Telemetry, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	tracerProvider, err := newTraceProvider(ctx, serviceName, config)
	if err != nil {
		return Telemetry{}, err
	}
	otel.SetTracerProvider(tracerProvider)
	_, ok := ctx.Value("telemetry_test_env").(struct{})
	if !ok {
		go setupPrometheus()
	}
	return Telemetry{TracerProvider: tracerProvider}, nil
}

func newTraceProvider(ctx context.Context, serviceName string, config Config) (*trace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

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
