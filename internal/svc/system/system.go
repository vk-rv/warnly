// Package system provides the implementation of the SystemService interface,
// which includes methods for listing slow queries, schemas, and errors.
package system

import (
	"context"
	"log/slog"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// SystemService reports olap resource usage.
type SystemService struct {
	store  warnly.AnalyticsStore
	now    func() time.Time
	logger *slog.Logger
}

// NewSystemService is a constructor of SystemService.
func NewSystemService(store warnly.AnalyticsStore, now func() time.Time, logger *slog.Logger) *SystemService {
	return &SystemService{store: store, now: now, logger: logger}
}

// ListSlowQueries lists slow queries from the system.
func (s *SystemService) ListSlowQueries(ctx context.Context) ([]warnly.SQLQuery, error) {
	return s.store.ListSlowQueries(ctx)
}

// ListSchemas lists database schemas from largest to smallest.
func (s *SystemService) ListSchemas(ctx context.Context) ([]warnly.Schema, error) {
	return s.store.ListSchemas(ctx)
}

// ListErrors lists recent errors from olap system for the last 24 hours.
func (s *SystemService) ListErrors(ctx context.Context) ([]warnly.AnalyticsStoreErr, error) {
	return s.store.ListErrors(ctx, warnly.ListErrorsCriteria{
		LastErrorTime: s.now().UTC().Add(-time.Hour * 24),
	})
}
