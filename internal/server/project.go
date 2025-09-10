package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

// projectHandler handles operations on project resource.
type projectHandler struct {
	svc         warnly.ProjectService
	cookieStore *session.CookieStore
	logger      *slog.Logger
}

// newProjectHandler is a constructor of projectHandler.
func newProjectHandler(
	svc warnly.ProjectService,
	cookieStore *session.CookieStore,
	logger *slog.Logger,
) *projectHandler {
	return &projectHandler{svc: svc, cookieStore: cookieStore, logger: logger}
}

// deleteAssignment deletes a user assignment for an issue.
func (h *projectHandler) deleteAssignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, "delete assignment: get project and issue", err)
		return
	}

	req := &warnly.UnassignIssueRequest{
		IssueID:   issueID,
		ProjectID: projectID,
		User:      &user,
	}

	err = h.svc.DeleteAssignment(ctx, req)
	if err != nil {
		h.writeError(ctx, w, "delete assignment: unassign issue", err)
		return
	}

	issues, period := r.URL.Query().Get("issues"), r.URL.Query().Get("period")
	if issues == "" && period == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	details, err := h.svc.GetProjectDetails(ctx, &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesType(issues),
		Period:    period,
		Start:     r.URL.Query().Get("start"),
		End:       r.URL.Query().Get("end"),
		Page:      h.getPage(r.URL.Query().Get("page")),
	}, &user)
	if err != nil {
		h.logger.Error("delete assignment: get project details", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project details server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeProjectDetails(ctx, w, r, details, &user)
}

// assignIssue assigns an issue to a user.
func (h *projectHandler) assignIssue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, "assign issue: get project and issue", err)
		return
	}

	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		h.writeError(ctx, w, "assign issue: parse user ID", err)
		return
	}

	req := &warnly.AssignIssueRequest{
		IssueID:   issueID,
		ProjectID: projectID,
		User:      &user,
		UserID:    userID,
	}

	err = h.svc.AssignIssue(ctx, req)
	if err != nil {
		h.writeError(ctx, w, "assign issue: assign issue", err)
		return
	}

	issues, period := r.URL.Query().Get("issues"), r.URL.Query().Get("period")
	if issues == "" && period == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	details, err := h.svc.GetProjectDetails(ctx, &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesType(issues),
		Period:    period,
		Start:     r.URL.Query().Get("start"),
		End:       r.URL.Query().Get("end"),
		Page:      h.getPage(r.URL.Query().Get("page")),
	}, &user)
	if err != nil {
		h.logger.Error("assign issue: get project details", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project details server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeProjectDetails(ctx, w, r, details, &user)
}

// listEvents lists all events per specified issue.
// it handles "All Errors" page in issue details.
func (h *projectHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, "list events: get project and issue", err)
		return
	}

	offset, err := parseOffset(r.URL.Query().Get("offset"))
	if err != nil {
		h.writeError(ctx, w, "list events: parse offset", err)
		return
	}

	req := &warnly.ListEventsRequest{
		Query:     r.URL.Query().Get("query"),
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
		Offset:    offset,
	}

	res, err := h.svc.ListEvents(ctx, req)
	if err != nil {
		h.writeError(ctx, w, "list events: get events", err)
		return
	}

	if err = web.Events(res).Render(ctx, w); err != nil {
		h.logger.Error("list events web render", slog.Any("error", err))
	}
}

// writeError logs the error and renders a server error page.
func (h *projectHandler) writeError(ctx context.Context, w http.ResponseWriter, msg string, err error) {
	h.logger.Error(msg, slog.Any("error", err))
	if err = web.ServerError().Render(ctx, w); err != nil {
		h.logger.Error(msg+" server error web render", slog.Any("error", err))
	}
}

// getPage parses the page number from string to int.
func parseOffset(offsetParam string) (int, error) {
	if offsetParam == "" {
		return 0, nil
	}
	return strconv.Atoi(offsetParam)
}

// listFields renders list of fields related to an issue with some statistics,
// e.g. how many times a field like browser or os was seen in events.
func (h *projectHandler) listFields(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.logger.Error("list fields: get project and issue", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("list fields server error web render", slog.Any("error", err))
		}
		return
	}

	req := &warnly.ListFieldsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
	}

	fields, err := h.svc.ListFields(ctx, req)
	if err != nil {
		h.logger.Error("list fields: get fields", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("list fields server error web render", slog.Any("error", err))
		}
		return
	}

	if err = web.Fields(fields).Render(ctx, w); err != nil {
		h.logger.Error("list fields web render", slog.Any("error", err))
	}
}

// deleteMessage deletes a message (user comment for issue) by identifier in issue discussion.
func (h *projectHandler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.logger.Error("post discussion: get project and issue", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("post discussion server error web render", slog.Any("error", err))
		}
		return
	}

	messageID, err := strconv.Atoi(r.PathValue("message_id"))
	if err != nil {
		h.logger.Error("delete message: parse message ID", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("delete message server error web render", slog.Any("error", err))
		}
		return
	}

	discussion, err := h.svc.DeleteMessage(ctx, &warnly.DeleteMessageRequest{
		MessageID: messageID,
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
	})
	if err != nil {
		h.logger.Error("delete message: delete message", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("delete message server error web render", slog.Any("error", err))
		}
		return
	}

	if err = web.DiscussionMessages(discussion).Render(ctx, w); err != nil {
		h.logger.Error("delete message web render", slog.Any("error", err))
	}
}

// postMessage creates a new message (user comment) in issue discussion.
func (h *projectHandler) postMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.logger.Error("post discussion: get project and issue", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("post discussion server error web render", slog.Any("error", err))
		}
		return
	}

	req, err := newMessageRequest(r, projectID, issueID, &user)
	if err != nil {
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("post discussion server error web render", slog.Any("error", err))
		}
		return
	}

	discussion, err := h.svc.CreateMessage(ctx, req)
	if err != nil {
		h.logger.Error("post discussion: create discussion", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("post discussion server error web render", slog.Any("error", err))
		}
		return
	}

	if err = web.DiscussionMessages(discussion).Render(ctx, w); err != nil {
		h.logger.Error("post discussion web render", slog.Any("error", err))
	}
}

// newMessageRequest creates a new message request from the HTTP request.
func newMessageRequest(r *http.Request, projectID, issueID int, user *warnly.User) (*warnly.CreateMessageRequest, error) {
	content := r.FormValue("content")
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}

	mentionedUsers := r.Form["mentioned_users"]
	mentionedUserIDs := make([]int, 0, len(mentionedUsers))
	for _, mentionedUser := range mentionedUsers {
		id, err := strconv.Atoi(mentionedUser)
		if err != nil {
			return nil, fmt.Errorf("invalid mentioned user ID: %s", mentionedUser)
		}
		mentionedUserIDs = append(mentionedUserIDs, id)
	}

	return &warnly.CreateMessageRequest{
		ProjectID:      projectID,
		IssueID:        issueID,
		User:           user,
		Content:        content,
		MentionedUsers: mentionedUserIDs,
	}, nil
}

// getDiscussions retrieves discussions for an issue.
func (h *projectHandler) getDiscussions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.logger.Error("get discussions: get project and issue", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("get discussions server error web render", slog.Any("error", err))
		}
		return
	}

	req := &warnly.GetDiscussionsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
	}

	discussion, err := h.svc.GetDiscussion(ctx, req)
	if err != nil {
		h.logger.Error("get discussions: get discussions", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("get discussions server error web render", slog.Any("error", err))
		}
		return
	}

	if err = web.Discussion(discussion).Render(ctx, w); err != nil {
		h.logger.Error("get discussions web render", slog.Any("error", err))
	}
}

// getIssue renders issue page.
func (h *projectHandler) getIssue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.logger.Error("project get issue: get project and issue", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project get issue server error web render", slog.Any("error", err))
		}
		return
	}

	req := &warnly.GetIssueRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		Period:    r.URL.Query().Get("period"),
		User:      &user,
	}

	issue, err := h.svc.GetIssue(ctx, req)
	if err != nil {
		h.logger.Error("project get issue: get issue", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project get issue server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeIssue(ctx, w, r, issue, &user)
}

// writeIssue writes issue details to the response writer.
func (h *projectHandler) writeIssue(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	issue *warnly.IssueDetails, user *warnly.User,
) {
	source := r.URL.Query().Get("source")
	if r.Header.Get(htmxHeader) != "" {
		if err := web.GetIssueHtmx(issue, user, source).Render(ctx, w); err != nil {
			h.logger.Error("project get issue htmx web render", slog.Any("error", err))
		}
	} else {
		if err := web.GetIssue(issue, user, source).Render(ctx, w); err != nil {
			h.logger.Error("project get issue web render", slog.Any("error", err))
		}
	}
}

// searchProjectByName is a method that searches projects by name.
func (h *projectHandler) searchProjectByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	name := r.URL.Query().Get("projectName")
	if name == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err := h.svc.SearchProject(ctx, name, &user)
	if err != nil {
		if errors.Is(err, warnly.ErrProjectNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.logger.Error("search project by name", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project details server error web render", slog.Any("error", err))
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// projectDetails renders project details page.
func (h *projectHandler) projectDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.logger.Error("project details: parse project ID", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project details server error web render", slog.Any("error", err))
		}
		return
	}

	req := &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesType(r.URL.Query().Get("issues")),
		Period:    r.URL.Query().Get("period"),
		Start:     r.URL.Query().Get("start"),
		End:       r.URL.Query().Get("end"),
		Page:      h.getPage(r.URL.Query().Get("page")),
	}

	details, err := h.svc.GetProjectDetails(ctx, req, &user)
	if err != nil {
		h.logger.Error("project details: get project details", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project details server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeProjectDetails(ctx, w, r, details, &user)
}

// writeProjectDetails writes project details to the response writer.
func (h *projectHandler) writeProjectDetails(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	details *warnly.ProjectDetails,
	user *warnly.User,
) {
	if r.URL.Query().Get("out") == "table" && r.Header.Get(htmxHeader) != "" {
		w.Header().Add("Hx-Push-Url", r.URL.Path+"?"+r.URL.RawQuery)
		if err := web.IssueListTable(details /* isHtmx call */, true).Render(ctx, w); err != nil {
			h.logger.Error("project details issue list table web render", slog.Any("error", err))
		}
		return
	}
	if r.Header.Get(htmxHeader) != "" {
		if err := web.ProjectDetailsHtmx(details, user /* isHtmx call */, true).Render(ctx, w); err != nil {
			h.logger.Error("project details htmx web render", slog.Any("error", err))
		}
	} else {
		if err := web.ProjectDetails(details, user).Render(ctx, w); err != nil {
			h.logger.Error("project details web render", slog.Any("error", err))
		}
	}
}

// listProjects returns a list of projects among with errors for last 24 hours.
func (h *projectHandler) listProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	name := r.URL.Query().Get("name")
	teamID := 0
	var err error
	if team := r.URL.Query().Get("team"); team != "" {
		teamID, err = strconv.Atoi(team)
		if err != nil {
			h.logger.Error("list projects: parse team ID", slog.Any("error", err))
			return
		}
	}

	criteria := warnly.ListProjectsCriteria{
		TeamID: teamID,
		Name:   name,
	}

	res, err := h.svc.ListProjects(ctx, &criteria, &user)
	if err != nil {
		h.logger.Error("list projects: get projects", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("list projects server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeProjectContents(ctx, r, w, &user, res)
}

// writeProjectContents is a method that renders project contents.
// It is used for both HTMX and non-HTMX requests.
func (h *projectHandler) writeProjectContents(
	ctx context.Context,
	r *http.Request,
	w http.ResponseWriter,
	user *warnly.User,
	res *warnly.ListProjectsResult,
) {
	// don't update the whole screen when searching project by name
	const projectGrid = "projectGrid"

	if r.Header.Get(htmxHeader) != "" {
		if r.Header.Get(htmxTarget) == projectGrid {
			if err := web.ProjectGrid(res).Render(ctx, w); err != nil {
				h.logger.Error("project contents htmx web render", slog.Any("error", err))
			}
			return
		}
		if err := web.ProjectContentHtmx(user, res).Render(ctx, w); err != nil {
			h.logger.Error("project contents htmx web render", slog.Any("error", err))
		}
	} else {
		if err := web.ProjectContents(user, res).Render(ctx, w); err != nil {
			h.logger.Error("project contents web render", slog.Any("error", err))
		}
	}
}

// deleteProject is a method that deletes a project.
func (h *projectHandler) deleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	id := r.PathValue("id")
	projectID, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Error("delete project: parse project ID", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("delete project server error web render", slog.Any("error", err))
		}
		return
	}

	err = h.svc.DeleteProject(ctx, projectID, &user)
	if err != nil {
		if errors.Is(err, warnly.ErrProjectNotFound) {
			w.Header().Add("Hx-Redirect", "/")
			return
		}
		h.logger.Error("delete project: delete project", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("delete project server error web render", slog.Any("error", err))
		}
		return
	}

	w.Header().Add("Hx-Redirect", "/projects")
}

// projectSettings is a method that renders project settings page.
func (h *projectHandler) projectSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	id := r.PathValue("id")
	projectID, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Error("project settings: parse project ID", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project settings server error web render", slog.Any("error", err))
		}
		return
	}

	project, err := h.svc.GetProject(ctx, projectID, &user)
	if err != nil {
		h.logger.Error("project settings: get project", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("project settings server error web render", slog.Any("error", err))
		}
		return
	}

	h.writeProjectSettings(ctx, w, r, project, &user)
}

// writeProjectSettings writes project settings to the response writer.
func (h *projectHandler) writeProjectSettings(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	project *warnly.Project,
	user *warnly.User,
) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.ProjectSettingsHtmx(project).Render(ctx, w); err != nil {
			h.logger.Error("project settings htmx web render", slog.Any("error", err))
		}
	} else {
		if err := web.ProjectSettings(project, user).Render(ctx, w); err != nil {
			h.logger.Error("project settings web render", slog.Any("error", err))
		}
	}
}

// createProject creates a new project.
func (h *projectHandler) createProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	req, problems, err := decodeValid[warnly.CreateProjectRequest](r)
	if err != nil {
		h.logger.Error("create new project: decodeValid project info", slog.Any("problems", problems), slog.Any("error", err))
		if err = web.Hello("").Render(ctx, w); err != nil {
			h.logger.Error("create new project: hello web render", slog.Any("error", err))
		}
		return
	}

	res, err := h.svc.CreateProject(ctx, &req, &user)
	if err != nil {
		h.logger.Error("create new project: create project", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("create new project server error web render", slog.Any("error", err))
		}
		return
	}

	if err = web.GettingStarted(res).Render(r.Context(), w); err != nil {
		h.logger.Error("create new project: getting started web render", slog.Any("error", err))
	}
}

// getPlatforms renders possible project platforms.
func (h *projectHandler) getPlatforms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	teams, err := h.svc.ListTeams(ctx, &user)
	if err != nil {
		h.logger.Error("get platforms: get teams", slog.Any("error", err))
		if err = web.ServerError().Render(ctx, w); err != nil {
			h.logger.Error("get platforms server error web render", slog.Any("error", err))
		}
		return
	}

	h.writePlatform(w, r, teams, &user)
}

// writePlatform writes platform information to the response writer.
func (h *projectHandler) writePlatform(w http.ResponseWriter, r *http.Request, teams []warnly.Team, user *warnly.User) {
	if r.Header.Get(htmxHeader) != "" {
		if err := web.PlatformHtmx(teams).Render(r.Context(), w); err != nil {
			h.logger.Error("get platforms htmx web render", slog.Any("error", err))
		}
	} else {
		if err := web.Platform(teams, user).Render(r.Context(), w); err != nil {
			h.logger.Error("get platforms web render", slog.Any("error", err))
		}
	}
}

// gettingStarted renders getting started page.
func (h *projectHandler) gettingStarted(w http.ResponseWriter, r *http.Request) {
	if err := web.GettingStarted(nil).Render(r.Context(), w); err != nil {
		h.logger.Error("getting started web render", slog.Any("error", err))
	}
}

// getProjectIssues is a helper method that retrieves project id and issue id from the request.
func getProjectIssue(r *http.Request) (projectID, issueID int, err error) {
	projectID, err = strconv.Atoi(r.PathValue("project_id"))
	if err != nil {
		return 0, 0, fmt.Errorf("parse project ID: %w", err)
	}
	issueID, err = strconv.Atoi(r.PathValue("issue_id"))
	if err != nil {
		return 0, 0, fmt.Errorf("parse issue ID: %w", err)
	}

	return projectID, issueID, nil
}

// getPage parses the page number from string to int.
func (h *projectHandler) getPage(page string) int {
	const defaultPage = 1
	if page == "" {
		return defaultPage
	}
	p, err := strconv.Atoi(page)
	if err != nil || p < 1 {
		h.logger.Error("get page: parse page number", slog.Any("error", err), slog.String("page", page))
		return defaultPage
	}
	return p
}
