// Package svcotel provides OpenTelemetry tracer provider interface and a no-op implementation.
package svcotel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerProvider is an interface that wraps the trace.TracerProvider and adds the Shutdown method
// that is not part of the interface but is part of the implementation.
type TracerProvider interface {
	trace.TracerProvider
	Shutdown(ctx context.Context) error
	RegisterSpanProcessor(sp tracesdk.SpanProcessor)
}

// NoopProvider is a no-op tracer provider implementation.
type NoopProvider struct {
	trace.TracerProvider
}

// NewNoopProvider returns a no-op tracer provider.
func NewNoopProvider() *NoopProvider {
	return &NoopProvider{
		TracerProvider: noop.NewTracerProvider(),
	}
}

// Shutdown is a no-op implementation of the Shutdown method.
func (p NoopProvider) Shutdown(context.Context) error {
	return nil
}

// RegisterSpanProcessor is a no-op implementation of the RegisterSpanProcessor method.
func (p NoopProvider) RegisterSpanProcessor(sp tracesdk.SpanProcessor) {}

// StartTracing configure open telemetry to be used.
func StartTracing(ctx context.Context, serviceName, reporterURI string, probability float64) (*tracesdk.TracerProvider, error) {
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(reporterURI),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	traceProvider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(probability)),
		tracesdk.WithBatcher(exporter,
			tracesdk.WithMaxExportBatchSize(tracesdk.DefaultMaxExportBatchSize),
			tracesdk.WithBatchTimeout(tracesdk.DefaultScheduleDelay*time.Millisecond),
		),
		tracesdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			),
		),
	)

	// We must set this provider as the global provider for things to work,
	// but we pass this provider around the program where needed to collect
	// our traces.
	otel.SetTracerProvider(traceProvider)

	// Chooses the HTTP header formats we extract incoming trace contexts from,
	// and the headers we set in outgoing requests.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return traceProvider, nil
}
