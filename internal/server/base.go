package server

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/web"
)

type BaseHandler struct {
	logger *slog.Logger
}

func NewBaseHandler(logger *slog.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

func (h *BaseHandler) writeError(ctx context.Context, w http.ResponseWriter, code int, msg string, err error) {
	h.logger.Error(msg, slog.Any("error", err))
	w.WriteHeader(code)
	if err = web.ServerError(strconv.Itoa(code), http.StatusText(code)).Render(ctx, w); err != nil {
		h.logger.Error(msg+" server error web render", slog.Any("error", err))
	}
}
