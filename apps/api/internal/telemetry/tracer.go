package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracer configures the OpenTelemetry tracer provider and returns a shutdown
// function that flushes and closes the exporter. Callers must invoke shutdown on exit.
//
// Exporter selection:
//   - exporterEndpoint non-empty → OTLP HTTP (production: Jaeger / OTel Collector)
//   - exporterEndpoint empty     → stdout (development / fallback)
func InitTracer(serviceName, exporterEndpoint string) (shutdown func(context.Context) error, err error) {
	var exporter sdktrace.SpanExporter

	if exporterEndpoint != "" {
		exporter, err = otlptracehttp.New(context.Background(),
			otlptracehttp.WithEndpoint(exporterEndpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
		}
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("create stdout trace exporter: %w", err)
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
