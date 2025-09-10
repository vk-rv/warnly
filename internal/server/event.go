package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/vk-rv/warnly/internal/warnly"
)

// eventHandler ingests events via API.
type eventHandler struct {
	svc    warnly.EventService
	logger *slog.Logger
}

// newEventAPIHandler is a constructor of eventHandler.
func newEventAPIHandler(svc warnly.EventService, logger *slog.Logger) *eventHandler {
	return &eventHandler{svc: svc, logger: logger}
}

// ingestEvent ingests new event.
func (h *eventHandler) ingestEvent(w http.ResponseWriter, r *http.Request) {
	if err := h.handleIngestEvent(r); err != nil {
		h.logger.Error("ingest new event", slog.Any("error", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// handleIngestEvent handles the actual logic of ingesting an event.
func (h *eventHandler) handleIngestEvent(r *http.Request) error {
	ctx := r.Context()

	projectID, err := strconv.Atoi(r.PathValue("project_id"))
	if err != nil {
		return fmt.Errorf("parse project id: %w", err)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	lines := strings.Split(string(b), "\n")
	if len(lines) < 3 {
		return fmt.Errorf("split body: %w", err)
	}

	content := lines[2]

	event := warnly.EventBody{}
	if err := json.Unmarshal([]byte(content), &event); err != nil {
		return fmt.Errorf("unmarshal body: %w", err)
	}

	req := &warnly.IngestRequest{
		Event:     &event,
		ProjectID: projectID,
		IP:        r.RemoteAddr,
	}

	if err := h.svc.IngestEvent(ctx, req); err != nil {
		return fmt.Errorf("ingest event: %w", err)
	}

	return nil
}
