package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vk-rv/warnly/internal/warnly"
)

func (s *Store) ListSchemas(ctx context.Context) ([]warnly.Schema, error) {
	return collect(ctx, s.db, `SELECT TABLE_NAME, COALESCE(DATA_LENGTH, 0) + COALESCE(INDEX_LENGTH, 0),
		COALESCE(TABLE_ROWS, 0), ENGINE FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() ORDER BY COALESCE(DATA_LENGTH, 0) + COALESCE(INDEX_LENGTH, 0) DESC`, nil,
		func(r *sql.Rows, m *warnly.Schema) error {
			if err := r.Scan(&m.Name, &m.TotalBytes, &m.TotalRows, &m.Engine); err != nil {
				return err
			}
			m.ReadableBytes = fmt.Sprintf("%d B", m.TotalBytes)
			return nil
		})
}

func (s *Store) ListErrors(ctx context.Context, c warnly.ListErrorsCriteria) ([]warnly.AnalyticsStoreErr, error) {
	return collect(ctx, s.db, `SELECT ERROR_NAME, SUM_ERROR_RAISED, LAST_SEEN
		FROM performance_schema.events_errors_summary_global_by_error
		WHERE LAST_SEEN > ? AND ERROR_NAME IS NOT NULL ORDER BY SUM_ERROR_RAISED DESC`, []any{c.LastErrorTime.UTC()},
		func(r *sql.Rows, m *warnly.AnalyticsStoreErr) error {
			return r.Scan(&m.Name, &m.Count, &m.MaxLastErrorTime)
		})
}

// ListSlowQueries reads MySQL digest statistics since startup/reset. They expose runtime
// and rows, but not per-query read bytes; those unavailable fields remain zero.
func (s *Store) ListSlowQueries(ctx context.Context) ([]warnly.SQLQuery, error) {
	return collect(ctx, s.db, `SELECT DIGEST_TEXT, AVG_TIMER_WAIT / 1000000000,
		SUM_ROWS_SENT / GREATEST(COUNT_STAR, 1),
		COUNT_STAR / GREATEST(TIMESTAMPDIFF(MINUTE, FIRST_SEEN, LAST_SEEN), 1), COUNT_STAR,
		COALESCE(100 * SUM_TIMER_WAIT / NULLIF(SUM(SUM_TIMER_WAIT) OVER (), 0), 0), DIGEST
		FROM performance_schema.events_statements_summary_by_digest
		WHERE SCHEMA_NAME = DATABASE() AND DIGEST IS NOT NULL
		ORDER BY SUM_TIMER_WAIT DESC LIMIT 10`, nil,
		func(r *sql.Rows, m *warnly.SQLQuery) error {
			m.TotalReadBytes = "N/A"
			return r.Scan(&m.NormalizedQuery, &m.AvgDuration, &m.AvgResultRows, &m.CallsPerMinute,
				&m.TotalCalls, &m.PercentageRuntime, &m.NormalizedQueryHash)
		})
}
