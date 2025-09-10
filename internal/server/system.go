// Package server provides HTTP handlers and middlewares for application.
package server

import (
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

// systemHandler reports resource usage.
type systemHandler struct {
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
	return &systemHandler{svc: svc, cookieStore: cookieStore, logger: logger}
}

// listSlowQueries lists slow queries from olap.
func (h *systemHandler) listSlowQueries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	result, err := h.svc.ListSlowQueries(ctx)
	if err != nil {
		h.logger.Error("list slow queries", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("list slow queries server error web render", slog.Any("error", err))
		}
		return
	}

	partial := r.URL.Query().Get("partial")
	isPartial := partial == "1"

	h.writeQueriesResponse(w, r, result, isPartial, &user)
}

// listSchemas lists olap database schemas from largest to smallest.
func (h *systemHandler) listSchemas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	result, err := h.svc.ListSchemas(ctx)
	if err != nil {
		h.logger.Error("get schemas", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("get schemas server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeSchemas(w, r, result, &user)
}

// listErrors lists recent errors from olap system for the last 24 hours.
func (h *systemHandler) listErrors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	result, err := h.svc.ListErrors(ctx)
	if err != nil {
		h.logger.Error("list errors", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("list errors server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeErrors(w, r, result, &user)
}

// writeQueriesResponse writes slow queries response.
func (h *systemHandler) writeQueriesResponse(
	w http.ResponseWriter,
	r *http.Request,
	result []warnly.SQLQuery,
	isPartial bool,
	user *warnly.User,
) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.SystemHtmx(result, isPartial).Render(r.Context(), w); err != nil {
			h.logger.Error("print queries htmx response", slog.Any("error", err))
		}
	} else {
		if err := web.System(result, isPartial, user).Render(r.Context(), w); err != nil {
			h.logger.Error("print queries response", slog.Any("error", err))
		}
	}
}

// writeErrors writes errors response.
func (h *systemHandler) writeErrors(
	w http.ResponseWriter,
	r *http.Request,
	result []warnly.AnalyticsStoreErr,
	user *warnly.User,
) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.ErrorsHtmx(result).Render(r.Context(), w); err != nil {
			h.logger.Error("print errors htmx response", slog.Any("error", err))
		}
	} else {
		if err := web.AnalyticStoreErrors(result, user).Render(r.Context(), w); err != nil {
			h.logger.Error("print errors response", slog.Any("error", err))
		}
	}
}

// writeSchemas writes schemas response.
func (h *systemHandler) writeSchemas(w http.ResponseWriter, r *http.Request, result []warnly.Schema, user *warnly.User) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.SchemaHtmx(result).Render(r.Context(), w); err != nil {
			h.logger.Error("print schema htmx response", slog.Any("error", err))
		}
	} else {
		if err := web.Schema(result, user).Render(r.Context(), w); err != nil {
			h.logger.Error("print schema response", slog.Any("error", err))
		}
	}
}
