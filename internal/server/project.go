package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/warnly"
)

// ProjectHandler handles operations on project resource.
type ProjectHandler struct {
	*BaseHandler

	svc    warnly.ProjectService
	logger *slog.Logger
}

// NewProjectHandler is a constructor of projectHandler.
func NewProjectHandler(
	svc warnly.ProjectService,
	logger *slog.Logger,
) *ProjectHandler {
	return &ProjectHandler{BaseHandler: NewBaseHandler(logger), svc: svc, logger: logger}
}

// DeleteAssignment deletes a user assignment for an issue.
func (h *ProjectHandler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "delete assignment: get project and issue", err)
		return
	}

	req := &warnly.UnassignIssueRequest{
		IssueID:   issueID,
		ProjectID: projectID,
		User:      &user,
	}

	err = h.svc.DeleteAssignment(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "delete assignment: unassign issue", err)
		return
	}

	issues := h.getFormOrQuery(r, "issues")
	period := h.getFormOrQuery(r, "period")
	if issues == "" && period == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	details, err := h.svc.GetProjectDetails(ctx, &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesType(issues),
		Period:    period,
		Start:     h.getFormOrQuery(r, "start"),
		End:       h.getFormOrQuery(r, "end"),
		Page:      h.getPage(h.getFormOrQuery(r, "page")),
	}, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "delete assignment: get project details", err)
		return
	}

	writeJSON(w, http.StatusOK, details)
}

// AssignIssue assigns an issue to a user.
func (h *ProjectHandler) AssignIssue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "assign issue: get project and issue", err)
		return
	}

	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "assign issue: parse user ID", err)
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
		h.writeError(ctx, w, http.StatusInternalServerError, "assign issue: assign issue", err)
		return
	}

	issues := h.getFormOrQuery(r, "issues")
	period := h.getFormOrQuery(r, "period")
	if issues == "" && period == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	details, err := h.svc.GetProjectDetails(ctx, &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesType(issues),
		Period:    period,
		Start:     h.getFormOrQuery(r, "start"),
		End:       h.getFormOrQuery(r, "end"),
		Page:      h.getPage(h.getFormOrQuery(r, "page")),
	}, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "assign issue: get project details", err)
		return
	}

	writeJSON(w, http.StatusOK, details)
}

// ListEvents lists all events per specified issue.
// it handles "All Errors" page in issue details.
func (h *ProjectHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "list events: get project and issue", err)
		return
	}

	offset, err := parseOffset(r.URL.Query().Get("offset"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "list events: parse offset", err)
		return
	}

	req := &warnly.ListEventsRequest{
		Query:     r.URL.Query().Get("query"),
		Period:    r.URL.Query().Get("period"),
		Start:     r.URL.Query().Get("start"),
		End:       r.URL.Query().Get("end"),
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
		Offset:    offset,
	}

	if req.Period == "" && req.Start == "" && req.End == "" {
		req.Period = "90d"
	}

	res, err := h.svc.ListEvents(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list events: get events", err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// ListFields returns list of fields related to an issue with some statistics,
// e.g. how many times a field like browser or os was seen in events.
func (h *ProjectHandler) ListFields(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "list fields: get project and issue", err)
		return
	}

	req := &warnly.ListFieldsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
	}

	fields, err := h.svc.ListFields(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list fields: get fields", err)
		return
	}

	writeJSON(w, http.StatusOK, fields)
}

// DeleteMessage deletes a message (user comment for issue) by identifier in issue discussion.
func (h *ProjectHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "post discussion: get project and issue", err)
		return
	}

	messageID, err := strconv.Atoi(r.PathValue("message_id"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "delete message: parse message ID", err)
		return
	}

	discussion, err := h.svc.DeleteMessage(ctx, &warnly.DeleteMessageRequest{
		MessageID: messageID,
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
	})
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "delete message: delete message", err)
		return
	}

	writeJSON(w, http.StatusOK, discussion)
}

// PostMessage creates a new message (user comment) in issue discussion.
func (h *ProjectHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "post discussion: get project and issue", err)
		return
	}

	req, err := newMessageRequest(r, projectID, issueID, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "post discussion: new message request", err)
		return
	}

	discussion, err := h.svc.CreateMessage(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "post discussion: create discussion", err)
		return
	}

	writeJSON(w, http.StatusOK, discussion)
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

// GetDiscussions retrieves discussions for an issue.
func (h *ProjectHandler) GetDiscussions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "get discussions: get project and issue", err)
		return
	}

	req := &warnly.GetDiscussionsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      &user,
	}

	discussion, err := h.svc.GetDiscussion(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "get discussions: get discussions", err)
		return
	}

	writeJSON(w, http.StatusOK, discussion)
}

// GetIssue returns issue page.
func (h *ProjectHandler) GetIssue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, issueID, err := getProjectIssue(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "project get issue: get project and issue", err)
		return
	}

	req := &warnly.GetIssueRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		Period:    r.URL.Query().Get("period"),
		EventID:   r.URL.Query().Get("event_id"),
		User:      &user,
		Source:    warnly.GetIssueRequestSource(r.URL.Query().Get("source")),
	}

	issue, err := h.svc.GetIssue(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "project get issue: get issue", err)
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

// SearchProjectByName is a method that searches projects by name.
func (h *ProjectHandler) SearchProjectByName(w http.ResponseWriter, r *http.Request) {
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
		h.writeError(ctx, w, http.StatusInternalServerError, "search project by name", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ProjectDetails returns project details page.
func (h *ProjectHandler) ProjectDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	projectID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "project details: parse project ID", err)
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
		h.writeError(ctx, w, http.StatusInternalServerError, "project details: get project details", err)
		return
	}

	writeJSON(w, http.StatusOK, details)
}

// ListProjects returns a list of projects among with errors for last 24 hours.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	name := r.URL.Query().Get("name")
	teamID := 0
	var err error
	if team := r.URL.Query().Get("team"); team != "" {
		teamID, err = strconv.Atoi(team)
		if err != nil {
			h.writeError(ctx, w, http.StatusBadRequest, "list projects: parse team ID", err)
			return
		}
	}

	criteria := warnly.ListProjectsCriteria{
		TeamID: teamID,
		Name:   name,
	}

	res, err := h.svc.ListProjects(ctx, &criteria, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list projects: get projects", err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// DeleteProject is a method that deletes a project.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	id := r.PathValue("id")
	projectID, err := strconv.Atoi(id)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "delete project: parse project ID", err)
		return
	}

	err = h.svc.DeleteProject(ctx, projectID, &user)
	if err != nil {
		if errors.Is(err, warnly.ErrProjectNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.writeError(ctx, w, http.StatusInternalServerError, "delete project: delete project", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ProjectSettings is a method that returns project settings page.
func (h *ProjectHandler) ProjectSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	id := r.PathValue("id")
	projectID, err := strconv.Atoi(id)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "project settings: parse project ID", err)
		return
	}

	project, err := h.svc.GetProject(ctx, projectID, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "project settings: get project", err)
		return
	}

	writeJSON(w, http.StatusOK, project)
}

// CreateProject creates a new project.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	req, err := decodeProject(r)
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "create new project: decode project info", err)
		return
	}

	res, err := h.svc.CreateProject(ctx, req, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "create new project: create project", err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// GetPlatforms returns possible project platforms.
func (h *ProjectHandler) GetPlatforms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	teams, err := h.svc.ListTeams(ctx, &user)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "get platforms: get teams", err)
		return
	}

	writeJSON(w, http.StatusOK, teams)
}

// GettingStarted returns getting started page.
func (h *ProjectHandler) GettingStarted(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("id", r.PathValue("projectID"))
	h.ProjectSettings(w, r)
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
func (h *ProjectHandler) getPage(page string) int {
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

// getFormOrQuery returns the value from form data first, falling back to query parameters.
func (h *ProjectHandler) getFormOrQuery(r *http.Request, key string) string {
	if val := r.FormValue(key); val != "" {
		return val
	}
	return r.URL.Query().Get(key)
}

// getPage parses the page number from string to int.
func parseOffset(offsetParam string) (int, error) {
	if offsetParam == "" {
		return 0, nil
	}
	return strconv.Atoi(offsetParam)
}
