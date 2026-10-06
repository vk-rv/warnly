// Package server provides HTTP handlers and middlewares for application.
package server

import (
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
)

// systemHandler reports resource usage.
type systemHandler struct {
	*BaseHandler

	svc         warnly.SystemService
	cookieStore *session.CookieStore
	logger      *slog.Logger
}

// newSystemtHandler is a constructor of a system handler.
func newSystemHandler(
	svc warnly.SystemService,
	cookieStore *session.CookieStore,
	logger *slog.Logger,
) *systemHandler {
	return &systemHandler{BaseHandler: NewBaseHandler(logger), svc: svc, cookieStore: cookieStore, logger: logger}
}

// listSlowQueries lists slow queries from olap.
func (h *systemHandler) listSlowQueries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	result, err := h.svc.ListSlowQueries(ctx)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list slow queries", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// listSchemas lists olap database schemas from largest to smallest.
func (h *systemHandler) listSchemas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	result, err := h.svc.ListSchemas(ctx)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list schemas", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// listErrors lists recent errors from olap system for the last 24 hours.
func (h *systemHandler) listErrors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	result, err := h.svc.ListErrors(ctx)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list errors", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
