package server

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// prometheusMW is a middleware for Prometheus metrics.
type prometheusMW struct {
	metrics *mwMetrics
	now     func() time.Time
}

// mwMetrics holds Prometheus metrics for the middleware.
type mwMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// newPrometheusMW is a constructor of prometheusMW.
func newPrometheusMW(r prometheus.Registerer, now func() time.Time) *prometheusMW {
	return &prometheusMW{
		metrics: &mwMetrics{
			requestsTotal: promauto.With(r).NewCounterVec(prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed.",
			}, []string{"path", "method", "code"}),
			requestDuration: promauto.With(r).NewHistogramVec(prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds.",
				Buckets: prometheus.DefBuckets,
			}, []string{"path", "method", "code"}),
		},
		now: now,
	}
}

// recordLatency is a middleware that records the latency of HTTP requests.
func (mw *prometheusMW) recordLatency(handler http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		start := mw.now()
		rec := &statusRecorder{ResponseWriter: writer, statusCode: http.StatusOK}

		handler.ServeHTTP(rec, request)

		duration := time.Since(start).Seconds()

		mw.metrics.requestsTotal.WithLabelValues(request.Pattern, request.Method, strconv.Itoa(rec.statusCode)).Inc()
		mw.metrics.requestDuration.WithLabelValues(request.Pattern, request.Method, strconv.Itoa(rec.statusCode)).Observe(duration)
	}
}

// statusRecorder is a wrapper for http.ResponseWriter that records the status code.
type statusRecorder struct {
	http.ResponseWriter

	statusCode int
}

// WriteHeader records the status code and calls the underlying WriteHeader method.
func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Flush implements the http.Flusher interface.
func (rec *statusRecorder) Flush() {
	fl, ok := rec.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	fl.Flush()
}

// Hijack implements the http.Hijacker interface.
func (rec *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rec.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hj.Hijack()
}
