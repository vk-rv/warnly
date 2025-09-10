package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/vk-rv/warnly/internal/web"
)

// recoverMw is a middleware for recovering from golang panics in HTTP handlers.
type recoverMw struct {
	metrics *recoverMetrics
	logger  *slog.Logger
}

// recoverMetrics holds metrics for the recover middleware.
type recoverMetrics struct {
	panicRecoversTotal *prometheus.CounterVec
}

// newRecoverMw is a constructor of recoverMw.
func newRecoverMw(r prometheus.Registerer, logger *slog.Logger) *recoverMw {
	return &recoverMw{
		logger: logger,
		metrics: &recoverMetrics{
			panicRecoversTotal: promauto.With(r).NewCounterVec(prometheus.CounterOpts{
				Name: "http_panic_recovers_total",
				Help: "Total number of HTTP panics recovered.",
			}, []string{"path", "method"}),
		},
	}
}

// recover is a middleware that recovers from golang panics in HTTP handlers.
func (mw *recoverMw) recover(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				mw.metrics.panicRecoversTotal.WithLabelValues(r.Pattern, r.Method).Inc()
				if vrvr, ok := rvr.(error); ok && errors.Is(vrvr, http.ErrAbortHandler) {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}

				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				err, ok := rvr.(error)
				if !ok {
					err = fmt.Errorf("%v", rvr)
				}
				mw.logger.Error(err.Error(), "event", "panic", "stack", "...\n"+string(buf))

				if r.Header.Get("Connection") != "Upgrade" {
					if r.Header.Get(htmxHeader) != "" {
						w.Header().Add("Hx-Redirect", "/error")
					} else {
						if err = web.ServerError().Render(r.Context(), w); err != nil {
							mw.logger.Error("server error web render", slog.Any("error", err))
						}
					}
				}
				return
			}
		}()
		handler.ServeHTTP(w, r)
	}
}
