package server

import (
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

// issueHandler handles HTTP requests related to issues.
type issueHandler struct {
	svc         warnly.ProjectService
	cookieStore *session.CookieStore
	logger      *slog.Logger
}

// newIssueHandler creates a new issueHandler instance.
func newIssueHandler(
	svc warnly.ProjectService,
	cookieStore *session.CookieStore,
	logger *slog.Logger,
) *issueHandler {
	return &issueHandler{svc: svc, cookieStore: cookieStore, logger: logger}
}

// listIssues handles the HTTP request to list issues on main page.
func (h *issueHandler) listIssues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	req := &warnly.ListIssuesRequest{
		User:        &user,
		Period:      r.URL.Query().Get("period"),
		ProjectName: r.URL.Query().Get("project_name"),
	}

	res, err := h.svc.ListIssues(ctx, req)
	if err != nil {
		h.logger.Error("list issues: get project", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("list issues server error web render", slog.Any("error", err))
		}
		return
	}

	if err = web.Issues(res).Render(ctx, w); err != nil {
		h.logger.Error("list issues web render", slog.Any("error", err))
	}
}
