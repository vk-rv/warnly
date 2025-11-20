package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRecoverMiddleware_NormalHandler(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		assert.NoError(t, err)
	}

	wrapped := mw.recover(handler)

	req := httptest.NewRequest(http.MethodGet, testPattern, http.NoBody)
	req.Pattern = testPattern
	w := httptest.NewRecorder()

	wrapped(w, req)

	assert.True(t, handlerCalled, "handler should be called")
	assert.Equal(t, http.StatusOK, w.Code)

	metricValue := testutil.ToFloat64(mw.metrics.panicRecoversTotal.WithLabelValues(testPattern, http.MethodGet))
	assert.Zero(t, metricValue, "no panics should be recovered")
}

func TestRecoverMiddleware_PanicRecovery_StringPanic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic string")
	}

	wrapped := mw.recover(handler)

	req := httptest.NewRequest(http.MethodGet, testPattern, http.NoBody)
	req.Pattern = testPattern
	w := httptest.NewRecorder()

	wrapped(w, req)

	metricValue := testutil.ToFloat64(mw.metrics.panicRecoversTotal.WithLabelValues(testPattern, http.MethodGet))
	assert.InEpsilon(t, 1.0, 0.1, metricValue, "panic should be recovered and counted")
}

func TestRecoverMiddleware_PanicRecovery_ErrorPanic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	customErr := io.EOF
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic(customErr)
	}

	wrapped := mw.recover(handler)

	req := httptest.NewRequest(http.MethodGet, testPattern, http.NoBody)
	req.Pattern = testPattern
	w := httptest.NewRecorder()

	wrapped(w, req)

	metricValue := testutil.ToFloat64(mw.metrics.panicRecoversTotal.WithLabelValues(testPattern, http.MethodGet))
	assert.InEpsilon(t, 1.0, 0.1, metricValue, "panic should be recovered and counted")
}

func TestRecoverMiddleware_ErrAbortHandler_IsPanic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	handler := func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}

	wrapped := mw.recover(handler)

	req := httptest.NewRequest(http.MethodGet, testPattern, http.NoBody)
	req.Pattern = testPattern
	w := httptest.NewRecorder()

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		wrapped(w, req)
	})
}

func TestRecoverMiddleware_WithHTMXRequest(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	wrapped := mw.recover(handler)

	req := httptest.NewRequest(http.MethodGet, testPattern, http.NoBody)
	req.Pattern = testPattern
	req.Header.Set(htmxHeader, "true")
	w := httptest.NewRecorder()

	wrapped(w, req)

	assert.Equal(t, "/error", w.Header().Get("Hx-Redirect"), "should redirect HTMX requests to /error")

	metricValue := testutil.ToFloat64(mw.metrics.panicRecoversTotal.WithLabelValues(testPattern, http.MethodGet))
	assert.InEpsilon(t, 1.0, 0.1, metricValue, "panic should be recovered and counted")
}

func TestRecoverMiddleware_WithUpgradeConnection(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	wrapped := mw.recover(handler)

	req := httptest.NewRequest(http.MethodGet, testPattern, http.NoBody)
	req.Pattern = testPattern
	req.Header.Set("Connection", "Upgrade")
	w := httptest.NewRecorder()

	wrapped(w, req)

	assert.Empty(t, w.Header().Get("Hx-Redirect"), "should not add Hx-Redirect for Upgrade connections")

	assert.Empty(t, w.Body.String(), "should not write response for Upgrade connections")

	metricValue := testutil.ToFloat64(mw.metrics.panicRecoversTotal.WithLabelValues(testPattern, http.MethodGet))
	assert.InEpsilon(t, 1.0, 0.1, metricValue, "panic should be recovered and counted")
}

func TestRecoverMiddleware_MultipleMetrics(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	registry := prometheus.NewRegistry()
	mw := newRecoverMw(registry, logger)

	panicHandler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	tests := []struct {
		name    string
		method  string
		pattern string
	}{
		{"GET /api/users", http.MethodGet, "/api/users"},
		{"POST /api/users", http.MethodPost, "/api/users"},
		{"PUT /api/users/1", http.MethodPut, "/api/users/1"},
	}

	wrapped := mw.recover(panicHandler)

	for _, tt := range tests {
		t.Parallel()

		req := httptest.NewRequest(tt.method, tt.pattern, http.NoBody)
		req.Pattern = tt.pattern
		w := httptest.NewRecorder()

		wrapped(w, req)

		metricValue := testutil.ToFloat64(mw.metrics.panicRecoversTotal.WithLabelValues(tt.pattern, tt.method))
		assert.InEpsilon(t, 1.0, 0.1, metricValue, "panic should be counted for %s %s", tt.method, tt.pattern)
	}
}
