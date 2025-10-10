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

	logger *slog.Logger
}

// newSettingsHandler creates a new settingsHandler instance.
func newSettingsHandler(logger *slog.Logger) *settingsHandler {
	return &settingsHandler{
		BaseHandler: NewBaseHandler(logger),
		logger:      logger,
	}
}

// listSettings handles the HTTP request to render the settings page.
func (h *settingsHandler) listSettings(w http.ResponseWriter, r *http.Request) {
	user := getUser(r.Context())
	h.writeSettings(w, r, &user)
}

// writeSettings writes the settings page to the response writer.
func (h *settingsHandler) writeSettings(w http.ResponseWriter, r *http.Request, user *warnly.User) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.SettingsHtmx().Render(r.Context(), w); err != nil {
			h.logger.Error("print settings htmx response", slog.Any("error", err))
		}
	} else {
		if err := web.Settings(user).Render(r.Context(), w); err != nil {
			h.logger.Error("print settings response", slog.Any("error", err))
		}
	}
}
