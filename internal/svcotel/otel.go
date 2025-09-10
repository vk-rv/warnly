// Package svcotel provides OpenTelemetry tracer provider interface and a no-op implementation.
package svcotel

import (
	"context"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
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
