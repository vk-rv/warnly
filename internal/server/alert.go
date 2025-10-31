package server

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

// AlertsHandler is a handler for alerts.
type AlertsHandler struct {
	*BaseHandler

	alertService warnly.AlertService
	logger       *slog.Logger
}

// NewAlertsHandler is a constructor of AlertsHandler.
func NewAlertsHandler(alertService warnly.AlertService, logger *slog.Logger) *AlertsHandler {
	return &AlertsHandler{
		BaseHandler:  NewBaseHandler(logger),
		alertService: alertService,
		logger:       logger,
	}
}

// ListAlerts is a handler for GET /alerts.
func (h *AlertsHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	teamName := r.URL.Query().Get("team_name")
	projectName := r.URL.Query().Get("project_name")
	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err != nil {
			h.writeError(ctx, w, http.StatusBadRequest, "list alerts: can't parse offset", err)
			return
		}
		offset = o
	}
	partial := r.URL.Query().Get("partial")

	res, err := h.alertService.ListAlerts(ctx, &warnly.ListAlertsRequest{
		User:        &user,
		TeamName:    teamName,
		ProjectName: projectName,
		Offset:      offset,
		Limit:       25,
	})
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list alerts: can't list alerts", err)
		return
	}

	h.writeAlerts(ctx, w, r, res, &user, partial)
}

// CreateAlert is a handler for GET /alerts/new.
func (h *AlertsHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	if r.Header.Get(htmxHeader) == "true" {
		if err := web.AddNewAlertHtmx(&user).Render(ctx, w); err != nil {
			h.logger.Error("add new alert htmx web render", slog.Any("error", err))
		}
	} else {
		if err := web.AddNewAlert(&user).Render(ctx, w); err != nil {
			h.logger.Error("add new alert web render", slog.Any("error", err))
		}
	}
}

func (h *AlertsHandler) writeAlerts(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	res *warnly.ListAlertsResult,
	user *warnly.User,
	partial string,
) {
	target := r.Header.Get("Hx-Target")

	switch {
	case partial == body:
		if err := web.AlertsBody(res).Render(ctx, w); err != nil {
			h.logger.Error("alerts body web render", slog.Any("error", err))
		}
	case target == "content":
		if err := web.AlertsHtmx(res).Render(ctx, w); err != nil {
			h.logger.Error("alerts htmx web render", slog.Any("error", err))
		}
	default:
		if err := web.Layout(web.AlertsTitle, web.AlertsHtmx(res), "/alerts", user).Render(ctx, w); err != nil {
			h.logger.Error("alerts web render", slog.Any("error", err))
		}
	}
}
