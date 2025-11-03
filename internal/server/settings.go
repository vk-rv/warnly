package server

import (
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
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

	data := &web.SettingsData{
		User:    &user,
		Webhook: webhook,
	}

	h.writeSettings(w, r, data)
}

// writeSettings writes the settings page to the response writer.
func (h *settingsHandler) writeSettings(w http.ResponseWriter, r *http.Request, data *web.SettingsData) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.SettingsHtmx(data).Render(r.Context(), w); err != nil {
			h.logger.Error("print settings htmx response", slog.Any("error", err))
		}
	} else {
		if err := web.Settings(data).Render(r.Context(), w); err != nil {
			h.logger.Error("print settings response", slog.Any("error", err))
		}
	}
}
