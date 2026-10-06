package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/warnly"
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

	res, err := h.alertService.ListAlerts(ctx, &warnly.ListAlertsRequest{
		User:        &user,
		TeamName:    teamName,
		ProjectName: projectName,
		Offset:      offset,
		Limit:       50,
	})
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list alerts: can't list alerts", err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// CreateAlertGet is a handler for GET /alerts/new.
func (h *AlertsHandler) CreateAlertGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	res, err := h.alertService.ListAlerts(ctx, &warnly.ListAlertsRequest{
		User:   &user,
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "create alert get: can't get projects", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"Projects": res.Projects})
}

// CreateAlert is a handler for POST /alerts.
func (h *AlertsHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	req, err := newAlertRequest(r, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "create alert: new alert request", err)
		return
	}

	_, err = h.alertService.CreateAlert(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "create alert: can't create alert", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EditAlertGet is a handler for GET /alerts/{id}/edit.
func (h *AlertsHandler) EditAlertGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	alertID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "edit alert get: parse alert ID", err)
		return
	}

	alert, err := h.alertService.GetAlert(ctx, alertID, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "edit alert get: can't get alert", err)
		return
	}

	res, err := h.alertService.ListAlerts(ctx, &warnly.ListAlertsRequest{
		User:   &user,
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "edit alert get: can't get projects", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"Alert": alert, "Projects": res.Projects})
}

// UpdateAlert is a handler for PUT /alerts/{id}.
func (h *AlertsHandler) UpdateAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	req, err := newUpdateAlertRequest(r, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "update alert: new update alert request", err)
		return
	}

	_, err = h.alertService.UpdateAlert(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "update alert: can't update alert", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteAlert is a handler for DELETE /alerts/{id}.
func (h *AlertsHandler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	alertID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "delete alert: parse alert ID", err)
		return
	}

	if err := h.alertService.DeleteAlert(ctx, alertID, &user); err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "delete alert: can't delete alert", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// newAlertRequest is a helper function that parses alert request from the HTTP request.
func newAlertRequest(r *http.Request, user *warnly.User) (*warnly.CreateAlertRequest, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("create alert: parse form: %w", err)
	}

	projectID, err := strconv.Atoi(r.FormValue("project_id"))
	if err != nil {
		return nil, fmt.Errorf("create alert: parse project ID: %w", err)
	}

	ruleName := r.FormValue("rule_name")
	if ruleName == "" {
		return nil, errors.New("create alert: rule name is required")
	}

	threshold, err := strconv.Atoi(r.FormValue("threshold"))
	if err != nil {
		return nil, fmt.Errorf("create alert: parse threshold: %w", err)
	}

	condition, err := strconv.Atoi(r.FormValue("condition"))
	if err != nil {
		return nil, fmt.Errorf("create alert: parse condition: %w", err)
	}

	timeframe, err := strconv.Atoi(r.FormValue("timeframe"))
	if err != nil {
		return nil, fmt.Errorf("create alert: parse timeframe: %w", err)
	}

	highPriority := r.FormValue("high_priority") == "true"

	return &warnly.CreateAlertRequest{
		User:         user,
		ProjectID:    projectID,
		RuleName:     ruleName,
		Threshold:    threshold,
		Condition:    warnly.AlertCondition(condition),
		Timeframe:    warnly.AlertTimeframe(timeframe),
		HighPriority: highPriority,
	}, nil
}

// newUpdateAlertRequest is a helper function that parses update alert request from the HTTP request.
func newUpdateAlertRequest(r *http.Request, user *warnly.User) (*warnly.UpdateAlertRequest, error) {
	alertID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return nil, fmt.Errorf("update alert: parse alert ID: %w", err)
	}

	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("update alert: parse form: %w", err)
	}

	ruleName := r.FormValue("rule_name")
	if ruleName == "" {
		return nil, errors.New("update alert: rule name is required")
	}

	threshold, err := strconv.Atoi(r.FormValue("threshold"))
	if err != nil {
		return nil, fmt.Errorf("update alert: parse threshold: %w", err)
	}

	condition, err := strconv.Atoi(r.FormValue("condition"))
	if err != nil {
		return nil, fmt.Errorf("update alert: parse condition: %w", err)
	}

	timeframe, err := strconv.Atoi(r.FormValue("timeframe"))
	if err != nil {
		return nil, fmt.Errorf("update alert: parse timeframe: %w", err)
	}

	highPriority := r.FormValue("high_priority") == "true"
	status := r.FormValue("status")

	return &warnly.UpdateAlertRequest{
		User:         user,
		AlertID:      alertID,
		RuleName:     ruleName,
		Threshold:    threshold,
		Condition:    warnly.AlertCondition(condition),
		Timeframe:    warnly.AlertTimeframe(timeframe),
		HighPriority: highPriority,
		Status:       warnly.AlertStatus(status),
	}, nil
}
