// Package chprometheus provides a Prometheus collector for ClickHouse connection pool statistics.
package chprometheus

import (
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"
)

// StatGetter provides a method to get pool statistics.
type StatGetter interface {
	Stats() driver.Stats
}

// Collector collects statistics from a redis client.
// It implements the prometheus.Collector interface.
type Collector struct {
	getter      StatGetter
	idleDesc    *prometheus.Desc
	openDesc    *prometheus.Desc
	maxIdleDesc *prometheus.Desc
	maxOpenDesc *prometheus.Desc
}

var _ prometheus.Collector = (*Collector)(nil)

// NewClickhouseCollector returns a new Collector based on the provided StatGetter.
// The given namespace and subsystem are used to build the fully qualified metric name,
// i.e. "{namespace}_{subsystem}_{metric}".
func NewClickhouseCollector(getter StatGetter, dbName string) *Collector {
	fqName := func(name string) string {
		return "analytics_" + name
	}
	return &Collector{
		getter: getter,
		idleDesc: prometheus.NewDesc(
			fqName("conn_idle_current"),
			"Current number of idle connections in the pool",
			nil, prometheus.Labels{"db_name": dbName},
		),
		openDesc: prometheus.NewDesc(
			fqName("conn_open_current"),
			"Current number of open connections in the pool",
			nil, prometheus.Labels{"db_name": dbName},
		),
		maxIdleDesc: prometheus.NewDesc(
			fqName("conn_max_idle_current"),
			"Max number of idle connections in the pool",
			nil, prometheus.Labels{"db_name": dbName},
		),
		maxOpenDesc: prometheus.NewDesc(
			fqName("conn_max_open_current"),
			"Max number of open connections in the pool",
			nil, prometheus.Labels{"db_name": dbName},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (s *Collector) Describe(descs chan<- *prometheus.Desc) {
	descs <- s.idleDesc
	descs <- s.openDesc
	descs <- s.maxIdleDesc
	descs <- s.maxOpenDesc
}

// Collect implements the prometheus.Collector interface.
func (s *Collector) Collect(metrics chan<- prometheus.Metric) {
	stats := s.getter.Stats()
	metrics <- prometheus.MustNewConstMetric(
		s.idleDesc,
		prometheus.GaugeValue,
		float64(stats.Idle),
	)
	metrics <- prometheus.MustNewConstMetric(
		s.openDesc,
		prometheus.GaugeValue,
		float64(stats.Open),
	)
	metrics <- prometheus.MustNewConstMetric(
		s.maxIdleDesc,
		prometheus.GaugeValue,
		float64(stats.MaxIdleConns),
	)
	metrics <- prometheus.MustNewConstMetric(
		s.maxOpenDesc,
		prometheus.GaugeValue,
		float64(stats.MaxOpenConns),
	)
}
