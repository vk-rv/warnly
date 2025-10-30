package svcotel_test

import (
	"context"
	"testing"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	"github.com/vk-rv/warnly/internal/svcotel"
)

func TestNewNoopProvider(t *testing.T) {
	t.Parallel()

	provider := svcotel.NewNoopProvider()
	if provider == nil {
		t.Fatal("NewNoopProvider() returned nil")
	}
}

func TestNoopProvider_Shutdown(t *testing.T) {
	t.Parallel()

	provider := svcotel.NewNoopProvider()
	ctx := context.Background()

	err := provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() returned error: %v", err)
	}
}

func TestNoopProvider_RegisterSpanProcessor(t *testing.T) {
	t.Parallel()

	provider := svcotel.NewNoopProvider()

	sp := &mockSpanProcessor{}

	provider.RegisterSpanProcessor(sp)
}

type mockSpanProcessor struct{}

func (m *mockSpanProcessor) OnStart(ctx context.Context, span tracesdk.ReadWriteSpan) {}
func (m *mockSpanProcessor) OnEnd(span tracesdk.ReadOnlySpan)                         {}
func (m *mockSpanProcessor) Shutdown(ctx context.Context) error                       { return nil }
func (m *mockSpanProcessor) ForceFlush(ctx context.Context) error                     { return nil }
