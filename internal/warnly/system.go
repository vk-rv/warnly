package warnly

import (
	"context"
	"time"
)

// SystemService reports olap resource usage.
type SystemService interface {
	// ListSlowQueries lists olap slow queries from the system.
	ListSlowQueries(ctx context.Context) ([]SQLQuery, error)
	// ListSchemas lists olap database schemas from largest to smallest.
	ListSchemas(ctx context.Context) ([]Schema, error)
	// ListErrors lists recent errors from olap system for the last 24 hours.
	ListErrors(ctx context.Context) ([]AnalyticsStoreErr, error)
}

// AnalyticsStoreErr represents an error entry in the analytics store.
type AnalyticsStoreErr struct {
	MaxLastErrorTime time.Time
	Name             string
	Count            uint64
}

// Schema represents a database schema.
type Schema struct {
	Name          string `json:"name"`
	ReadableBytes string `json:"readable_bytes"`
	Engine        string `json:"engine"`
	PartitionKey  string `json:"partition_key"`
	TotalBytes    uint64 `json:"total_bytes"`
	TotalRows     uint64 `json:"total_rows"`
}

// SQLQuery represents a slow SQL query.
type SQLQuery struct {
	NormalizedQuery       string  `json:"normalized_query"`
	TotalReadBytes        string  `json:"total_read_bytes"`
	NormalizedQueryHash   string  `json:"normalized_query_hash"`
	TotalReadBytesNumeric float64 `json:"total_read_bytes_numeric"`
	AvgDuration           float64 `json:"avg_duration"`
	AvgResultRows         float64 `json:"avg_result_rows"`
	CallsPerMinute        float64 `json:"calls_per_minute"`
	TotalCalls            uint64  `json:"total_calls"`
	PercentageIOPS        float64 `json:"percentage_iops"`
	PercentageRuntime     float64 `json:"percentage_runtime"`
	Percent               float64 `json:"percent"`
	ReadBytes             uint64  `json:"read_bytes"`
}
