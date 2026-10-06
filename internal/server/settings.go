package server

import (
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/warnly"
)

// settingsHandler handles HTTP requests related to Warnly settings.
type settingsHandler struct {
	*BaseHandler

	notificationService warnly.NotificationService
	logger              *slog.Logger
}

// newSettingsHandler creates a new settingsHandler instance.
func newSettingsHandler(notificationService warnly.NotificationService, logger *slog.Logger) *settingsHandler {
	return &settingsHandler{
		BaseHandler:         NewBaseHandler(logger),
		notificationService: notificationService,
		logger:              logger,
	}
}

// listSettings handles the HTTP request to render the settings page.
func (h *settingsHandler) listSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	webhook, err := h.notificationService.GetWebhookConfigWithSecretByTeamID(ctx, 1)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "get webhook config", err)
		return
	}

	data := map[string]any{
		"User":    &user,
		"Webhook": webhook,
	}

	writeJSON(w, http.StatusOK, data)
}
