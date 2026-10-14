package server

import (
	"context"
	"encoding/json/v2"
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
	h.writeJSON(w, code, errorResponse{Error: http.StatusText(code)})
}

// writeJSON buffers encoding before committing headers, so encoding failures return 500.
func (h *BaseHandler) writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		h.logger.Error("encode JSON response", slog.Any("error", err))
		status = http.StatusInternalServerError
		data = []byte(`{"error":"Internal Server Error"}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}
