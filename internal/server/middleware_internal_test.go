package server

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestStatusRecorder_Flush(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w}
	rec.Flush()
}

func TestStatusRecorder_WriteHeader(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	rec.WriteHeader(http.StatusNotFound)
	if rec.statusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.statusCode)
	}
}

func TestStatusRecorder_Hijack(t *testing.T) {
	t.Parallel()

	t.Run("when ResponseWriter does not support Hijacker", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		rec := &statusRecorder{ResponseWriter: w}

		conn, rw, err := rec.Hijack()

		assert.Nil(t, conn)
		assert.Nil(t, rw)
		assert.Equal(t, http.ErrNotSupported, err)
	})

	t.Run("when ResponseWriter supports Hijacker", func(t *testing.T) {
		t.Parallel()
		mockConn := &mockConn{}
		mockHijacker := &mockHijacker{
			conn: mockConn,
			rw:   bufio.NewReadWriter(bufio.NewReader(mockConn), bufio.NewWriter(mockConn)),
		}

		rec := &statusRecorder{ResponseWriter: mockHijacker}

		conn, rw, err := rec.Hijack()

		assert.Equal(t, mockConn, conn)
		assert.NotNil(t, rw)
		assert.NoError(t, err)
	})
}

type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type mockHijacker struct {
	conn net.Conn
	rw   *bufio.ReadWriter
}

func (m *mockHijacker) Header() http.Header         { return http.Header{} }
func (m *mockHijacker) Write(b []byte) (int, error) { return len(b), nil }
func (m *mockHijacker) WriteHeader(statusCode int)  {}
func (m *mockHijacker) Flush()                      {}
func (m *mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return m.conn, m.rw, nil
}

func TestRecordLatencyWithDifferentMethods(t *testing.T) {
	t.Parallel()
	registry := prometheus.NewRegistry()
	mw := newPrometheusMW(registry, time.Now)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrapped := mw.recordLatency(handler)
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", http.NoBody)
		req.Pattern = "/test"
		w := httptest.NewRecorder()
		wrapped(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("method %s: expected status %d, got %d", method, http.StatusOK, w.Code)
		}
	}
}

func TestRecordLatency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		handler        http.HandlerFunc
		name           string
		expectedMetric string
		expectedCode   int
	}{
		{
			name: "successful request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("OK"))
				assert.NoError(t, err)
			},
			expectedCode:   http.StatusOK,
			expectedMetric: "http_requests_total",
		},
		{
			name: "request with custom status code",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedCode:   http.StatusNotFound,
			expectedMetric: "http_requests_total",
		},
		{
			name: "request without explicit WriteHeader",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, err := w.Write([]byte("default status"))
				assert.NoError(t, err)
			},
			expectedCode:   http.StatusOK,
			expectedMetric: "http_requests_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			registry := prometheus.NewRegistry()
			now := time.Now
			mw := newPrometheusMW(registry, now)

			wrapped := mw.recordLatency(tt.handler)

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.Pattern = "/test"
			w := httptest.NewRecorder()

			wrapped(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			families, err := testutil.GatherAndCount(registry)
			if err != nil {
				t.Fatalf("failed to gather metrics: %v", err)
			}
			if families == 0 {
				t.Error("expected metrics to be recorded")
			}

			collected, err := testutil.GatherAndCount(registry)
			require.NoError(t, err)
			assert.Positive(t, collected, "should have collected metrics")

			statusCodeStr := strconv.Itoa(tt.expectedCode)
			metricResult := testutil.ToFloat64(mw.metrics.requestsTotal.WithLabelValues("/test", http.MethodGet, statusCodeStr))
			assert.InEpsilon(t, 1.0, metricResult, 0.1, "expected %s to have value 1.0 for status %d", tt.expectedMetric, tt.expectedCode)
		})
	}
}

func TestEmailMatcherMiddleware_AllowedEmail(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	rgxEmail := regexp.MustCompile(`@example\.com$`)
	mw := newEmailMatcherMW([]*regexp.Regexp{rgxEmail}, logger)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	wrapped := mw.emailMatch(handler)

	user := warnly.User{Email: "test@example.com", ID: 1}
	ctx := NewContextWithUser(t.Context(), user)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped(w, req)

	assert.True(t, handlerCalled, "handler should be called for allowed email")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEmailMatcherMiddleware_DeniedEmail_WithoutHTMX(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	rgxEmail := regexp.MustCompile(`@example\.com$`)
	mw := newEmailMatcherMW([]*regexp.Regexp{rgxEmail}, logger)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	wrapped := mw.emailMatch(handler)

	user := warnly.User{Email: "test@invalid.com", ID: 1}
	ctx := NewContextWithUser(context.Background(), user)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped(w, req)

	assert.False(t, handlerCalled, "handler should not be called for denied email")
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestEmailMatcherMiddleware_DeniedEmail_WithHTMX(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	rgxEmail := regexp.MustCompile(`@example\.com$`)
	mw := newEmailMatcherMW([]*regexp.Regexp{rgxEmail}, logger)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	wrapped := mw.emailMatch(handler)

	user := warnly.User{Email: "test@invalid.com", ID: 1}
	ctx := NewContextWithUser(context.Background(), user)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req = req.WithContext(ctx)
	req.Header.Set(htmxHeader, "true")
	w := httptest.NewRecorder()

	wrapped(w, req)

	assert.False(t, handlerCalled, "handler should not be called for denied email")
	assert.Equal(t, "/login", w.Header().Get("Hx-Redirect"))
}

func TestEmailMatcherMiddleware_MultiplePatterns(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`@example\.com$`),
		regexp.MustCompile(`@company\.org$`),
		regexp.MustCompile(`admin@.*\.dev$`),
	}
	mw := newEmailMatcherMW(patterns, logger)

	tests := []struct {
		name    string
		email   string
		allowed bool
	}{
		{"allowed by first pattern", "user@example.com", true},
		{"allowed by second pattern", "user@company.org", true},
		{"allowed by third pattern", "admin@test.dev", true},
		{"denied", "user@other.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handlerCalled := false
			handler := func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			}

			wrapped := mw.emailMatch(handler)

			user := warnly.User{Email: tt.email, ID: 1}
			ctx := NewContextWithUser(context.Background(), user)

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			wrapped(w, req)

			assert.Equal(t, tt.allowed, handlerCalled, "expected handler called=%v for email %s", tt.allowed, tt.email)
		})
	}
}
