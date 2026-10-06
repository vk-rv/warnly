package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type BaseHandler struct {
	logger *slog.Logger
}

func NewBaseHandler(logger *slog.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

func (h *BaseHandler) writeError(_ context.Context, w http.ResponseWriter, code int, msg string, err error) {
	h.logger.Error(msg, slog.Any("error", err))
	writeJSON(w, code, map[string]string{"error": http.StatusText(code)})
}

// writeJSON buffers encoding before committing headers, so encoding failures return 500.
func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}
