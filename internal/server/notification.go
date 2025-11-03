package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/warnly"
)

// notificationHandler handles HTTP requests related to notifications.
type notificationHandler struct {
	*BaseHandler

	notificationService warnly.NotificationService
	logger              *slog.Logger
}

// newNotificationHandler creates a new notificationHandler instance.
func newNotificationHandler(
	notificationService warnly.NotificationService,
	logger *slog.Logger,
) *notificationHandler {
	return &notificationHandler{
		BaseHandler:         NewBaseHandler(logger),
		notificationService: notificationService,
		logger:              logger,
	}
}

// SaveWebhook handles POST /settings/webhook.
func (h *notificationHandler) SaveWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	req, err := newSaveWebhookConfigRequest(r, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "save webhook config", err)
		return
	}

	if err := h.notificationService.SaveWebhookConfig(ctx, req); err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "save webhook config", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func newSaveWebhookConfigRequest(r *http.Request, user *warnly.User) (*warnly.SaveWebhookConfigRequest, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	teamID, err := strconv.Atoi(r.FormValue("team_id"))
	if err != nil {
		return nil, fmt.Errorf("parse team ID: %w", err)
	}
	if teamID == 0 {
		return nil, errors.New("team_id is 0")
	}
	url := r.FormValue("url")
	secret := r.FormValue("secret")

	return &warnly.SaveWebhookConfigRequest{
		User:   user,
		TeamID: teamID,
		URL:    url,
		Secret: secret,
	}, nil
}
