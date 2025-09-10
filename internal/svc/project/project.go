// Package project provides the implementation of the ProjectService interface,
// which includes methods for managing projects, issues, and teams.
package project

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/vk-rv/warnly/internal/uow"
	"github.com/vk-rv/warnly/internal/warnly"
)

const (
	defaultLimit = 50
)

// ProjectService implements warnly.ProjectService interface.
type ProjectService struct {
	now             func() time.Time
	projectStore    warnly.ProjectStore
	assingmentStore warnly.AssingmentStore
	teamStore       warnly.TeamStore
	issueStore      warnly.IssueStore
	analyticsStore  warnly.AnalyticsStore
	messageStore    warnly.MessageStore
	mentionStore    warnly.MentionStore
	uow             uow.StartUnitOfWork
	sanitizerPolicy *bluemonday.Policy
	logger          *slog.Logger
	baseURL         string
	scheme          string
}

// NewProjectService is a constructor of project service.
func NewProjectService(
	projectStore warnly.ProjectStore,
	assingmentStore warnly.AssingmentStore,
	teamStore warnly.TeamStore,
	issueStore warnly.IssueStore,
	messageStore warnly.MessageStore,
	mentionStore warnly.MentionStore,
	analyticsStore warnly.AnalyticsStore,
	uw uow.StartUnitOfWork,
	policy *bluemonday.Policy,
	baseURL string,
	scheme string,
	now func() time.Time,
	logger *slog.Logger,
) *ProjectService {
	return &ProjectService{
		assingmentStore: assingmentStore,
		projectStore:    projectStore,
		teamStore:       teamStore,
		issueStore:      issueStore,
		messageStore:    messageStore,
		mentionStore:    mentionStore,
		analyticsStore:  analyticsStore,
		baseURL:         baseURL,
		scheme:          scheme,
		uow:             uw,
		sanitizerPolicy: policy,
		logger:          logger,
		now:             now,
	}
}

// CreateProject creates a new project.
func (s *ProjectService) CreateProject(
	ctx context.Context,
	req *warnly.CreateProjectRequest,
	user *warnly.User,
) (*warnly.ProjectInfo, error) {
	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return nil, err
	}

	inTeam := false
	for _, team := range teams {
		if team.ID == req.TeamID {
			inTeam = true
			break
		}
	}
	if !inTeam {
		return nil, fmt.Errorf("team not found: %d", req.TeamID)
	}

	key, err := warnly.NewNanoID()
	if err != nil {
		return nil, err
	}

	project := &warnly.Project{
		CreatedAt: s.now().UTC(),
		Name:      req.ProjectName,
		UserID:    int(user.ID),
		TeamID:    req.TeamID,
		Platform:  warnly.PlatformByName(req.Platform),
		Key:       key,
	}

	if err := s.projectStore.CreateProject(ctx, project); err != nil {
		return nil, err
	}

	return &warnly.ProjectInfo{
		ID:   project.ID,
		Name: project.Name,
		DSN:  projectDSN(project.ID, project.Key, s.baseURL, s.scheme),
	}, nil
}

// DeleteProject deletes a project by unique identifier.
func (s *ProjectService) DeleteProject(ctx context.Context, projectID int, user *warnly.User) error {
	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return err
	}

	project, err := s.projectStore.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	for _, team := range teams {
		if team.ID == project.TeamID {
			return s.projectStore.DeleteProject(ctx, projectID)
		}
	}

	return warnly.ErrProjectNotFound
}

// GetProject returns a project by unique identifier.
func (s *ProjectService) GetProject(ctx context.Context, projectID int, user *warnly.User) (*warnly.Project, error) {
	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return nil, err
	}

	project, err := s.projectStore.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	for _, team := range teams {
		if team.ID == project.TeamID {
			return project, nil
		}
	}

	return nil, warnly.ErrProjectNotFound
}

// ListProjects returns a list of projects along with high-level event analytics.
func (s *ProjectService) ListProjects(
	ctx context.Context,
	criteria *warnly.ListProjectsCriteria,
	user *warnly.User,
) (*warnly.ListProjectsResult, error) {
	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return nil, err
	}

	if len(teams) == 0 {
		return &warnly.ListProjectsResult{}, nil
	}

	teamIDs := filterTeamIDs(teams, criteria.TeamID)

	projects, err := s.projectStore.ListProjects(ctx, teamIDs, criteria.Name)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return &warnly.ListProjectsResult{Teams: teams, Criteria: criteria}, nil
	}

	projectIDS := extractProjectIDs(projects, "")

	currentTime := s.now()
	events, err := s.analyticsStore.CalculateEvents(ctx, &warnly.ListIssueMetricsCriteria{
		ProjectIDs: projectIDS,
		From:       currentTime.Add(-24 * time.Hour),
		To:         currentTime,
	})
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return &warnly.ListProjectsResult{Teams: teams, Projects: projects, Criteria: criteria}, nil
	}

	eventsMap := mapEventsByProjectID(events)

	assignEventsToProjects(projects, eventsMap)

	return &warnly.ListProjectsResult{Teams: teams, Projects: projects, Criteria: criteria}, nil
}

// filterTeamIDs filters team IDs based on the provided team identifier.
func filterTeamIDs(teams []warnly.Team, teamID int) []int {
	teamIDs := make([]int, 0, len(teams))
	for _, team := range teams {
		if teamID != 0 && team.ID != teamID {
			continue
		}
		teamIDs = append(teamIDs, team.ID)
	}
	return teamIDs
}

// ListTeams returns a list of teams associated with user.
func (s *ProjectService) ListTeams(ctx context.Context, user *warnly.User) ([]warnly.Team, error) {
	return s.teamStore.ListTeams(ctx, int(user.ID))
}

// GetProjectDetails returns the project details.
func (s *ProjectService) GetProjectDetails(
	ctx context.Context,
	req *warnly.ProjectDetailsRequest,
	user *warnly.User,
) (*warnly.ProjectDetails, error) {
	if !warnly.IsAllowedIssueType(req.Issues) {
		req.Issues = warnly.IssuesTypeAll
	}

	project, err := s.GetProject(ctx, req.ProjectID, user)
	if err != nil {
		return nil, err
	}

	from, to, err := s.getTimeRange(req)
	if err != nil {
		return nil, err
	}

	issues, err := s.issueStore.ListIssues(ctx, warnly.ListIssuesCriteria{
		ProjectIDs: []int{project.ID},
		From:       from,
		To:         to,
	})
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return &warnly.ProjectDetails{Project: project}, nil
	}

	events, err := s.analyticsStore.CalculateEvents(ctx, &warnly.ListIssueMetricsCriteria{
		ProjectIDs: []int{project.ID},
		From:       from,
		To:         to,
	})
	if err != nil {
		return nil, err
	}

	issueList, err := s.listIssueEntries(ctx, project.ID, issues, from, to)
	if err != nil {
		return nil, err
	}

	issueList, err = s.populateMessagesCount(ctx, issueList)
	if err != nil {
		return nil, err
	}

	project.Events = events
	project.AllLength = len(issueList)
	project.NewIssueList = filterRecentIssues(issueList, s.now())
	project.NewLength = len(project.NewIssueList)
	project.IssueList = paginate(issueList, req.Page, warnly.PageSize)
	project.NewIssueList = paginate(project.NewIssueList, req.Page, warnly.PageSize)

	switch req.Issues {
	case warnly.IssuesTypeAll:
		project.ResultIssueList = project.IssueList
	case warnly.IssuesTypeNew:
		project.ResultIssueList = project.NewIssueList
	}

	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User:      user,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, err
	}

	assignments, err := s.buildTeammateAssigns(ctx, teammates, issues)
	if err != nil {
		return nil, err
	}

	return &warnly.ProjectDetails{
		Project:     project,
		Teammates:   teammates,
		Assignments: assignments,
	}, nil
}

// GetDiscussion returns issue discussions.
func (s *ProjectService) GetDiscussion(ctx context.Context, req *warnly.GetDiscussionsRequest) (*warnly.Discussion, error) {
	project, err := s.GetProject(ctx, req.ProjectID, req.User)
	if err != nil {
		return nil, err
	}

	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User:      req.User,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, err
	}

	issue, err := s.issueStore.GetIssueByID(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	messages, err := s.messageStore.ListIssueMessages(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	return &warnly.Discussion{
		Teammates: teammates,
		Messages:  messages,
		Info: warnly.DiscussionInfo{
			ProjectID:      project.ID,
			IssueID:        req.IssueID,
			IssueFirstSeen: issue.FirstSeen,
		},
	}, nil
}

// ListFields returns a list of fields related to an issue.
// e.g. how many times a field like "browser" or "os" was seen in events.
func (s *ProjectService) ListFields(ctx context.Context, req *warnly.ListFieldsRequest) (*warnly.ListFieldsResult, error) {
	project, err := s.GetProject(ctx, req.ProjectID, req.User)
	if err != nil {
		return nil, err
	}

	issue, err := s.issueStore.GetIssueByID(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()

	fieldCount, err := s.analyticsStore.CalculateFields(ctx, warnly.FieldsCriteria{
		IssueID:   req.IssueID,
		ProjectID: project.ID,
		From:      issue.FirstSeen,
		To:        now,
	})
	if err != nil {
		return nil, err
	}

	fieldMap := make(map[string]uint64, len(fieldCount))
	for i := range fieldCount {
		fieldMap[fieldCount[i].Tag] = fieldCount[i].Count
	}

	fieldValue, err := s.analyticsStore.CountFields(ctx, warnly.EventDefCriteria{
		GroupID:   req.IssueID,
		ProjectID: project.ID,
		From:      issue.FirstSeen,
		To:        now,
	})
	if err != nil {
		return nil, err
	}

	for i := range fieldValue {
		total, ok := fieldMap[fieldValue[i].Tag]
		if !ok {
			continue
		}
		fieldValue[i].PercentsOfTotal = (float64(fieldValue[i].Count) / float64(total)) * 100
	}

	return &warnly.ListFieldsResult{
		TagCount:      fieldCount,
		FieldValueNum: fieldValue,
	}, nil
}

// parseSearchQuery parses the search query into raw text and structured key-value pairs.
func parseSearchQuery(query string) (raw string, structured map[string]warnly.QueryValue, err error) {
	raw = query
	structured = make(map[string]warnly.QueryValue)

	if query == "" {
		return raw, structured, nil
	}

	parts := strings.Split(query, " ")
	for _, part := range parts {
		if strings.Contains(part, ":") {
			kv := strings.Split(part, ":")
			if len(kv) != 2 {
				return "", nil, fmt.Errorf("invalid query: %s", query)
			}
			key := kv[0]
			value := kv[1]
			queryValue := warnly.QueryValue{Value: value}
			if strings.HasPrefix(key, "!") {
				queryValue.IsNot = true
				key = strings.TrimPrefix(key, "!")
			}
			structured[key] = queryValue
		} else {
			raw = part
		}
	}

	return raw, structured, nil
}

// ListEvents handles "All Errors" page showing all error events per issue.
func (s *ProjectService) ListEvents(ctx context.Context, req *warnly.ListEventsRequest) (*warnly.ListEventsResult, error) {
	raw, structured, err := parseSearchQuery(req.Query)
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, req.ProjectID, req.User)
	if err != nil {
		return nil, err
	}

	issue, err := s.issueStore.GetIssueByID(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()

	criteria := &warnly.EventCriteria{
		ProjectID: project.ID,
		GroupID:   req.IssueID,
		From:      issue.FirstSeen,
		To:        now,
		Message:   raw,
		Tags:      structured,
		Limit:     defaultLimit,
		Offset:    req.Offset,
	}

	totalEvents, err := s.analyticsStore.CountEvents(ctx, criteria)
	if err != nil {
		return nil, err
	}

	events, err := s.analyticsStore.ListEvents(ctx, criteria)
	if err != nil {
		return nil, err
	}

	return &warnly.ListEventsResult{
		TotalEvents: totalEvents,
		Events:      events,
		ProjectID:   project.ID,
		IssueID:     req.IssueID,
		Offset:      req.Offset,
	}, nil
}

// parseDuration parses the period string into a time.Duration.
func parseDuration(period string, defaultDur time.Duration) (time.Duration, error) {
	dur, err := warnly.ParseDuration(period)
	if err != nil {
		return 0, err
	}
	if dur == 0 {
		dur = defaultDur
	}
	return dur, nil
}

// ListIssues lists issues for the projects the user has access to.
func (s *ProjectService) ListIssues(ctx context.Context, req *warnly.ListIssuesRequest) (*warnly.ListIssuesResult, error) {
	dur, err := parseDuration(req.Period, 14*24*time.Hour)
	if err != nil {
		return nil, err
	}

	to := s.now().UTC()
	from := to.Add(-dur)

	teams, err := s.teamStore.ListTeams(ctx, int(req.User.ID))
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, warnly.ErrNotFound
	}

	teamIDS := extractTeamIDs(teams)

	projects, err := s.projectStore.ListProjects(ctx, teamIDS, "")
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return &warnly.ListIssuesResult{
			Issues:      nil,
			LastProject: nil,
		}, nil
	}

	lastProject := projects[len(projects)-1]

	slices.SortFunc(projects, func(a, b warnly.Project) int {
		return cmp.Compare(a.ID, b.ID)
	})

	projectIDS := extractProjectIDs(projects, req.ProjectName)

	issues, err := s.issueStore.ListIssues(ctx, warnly.ListIssuesCriteria{
		ProjectIDs: projectIDS,
		From:       from,
		To:         to,
	})
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return &warnly.ListIssuesResult{
			Issues:           nil,
			LastProject:      &lastProject,
			Projects:         projects,
			RequestedProject: req.ProjectName,
		}, nil
	}

	issueList, err := s.buildIssueList(ctx, projectIDS, issues, from, to)
	if err != nil {
		return nil, err
	}

	issueList, err = s.populateMessagesCount(ctx, issueList)
	if err != nil {
		return nil, err
	}

	return &warnly.ListIssuesResult{
		Issues:           issueList,
		LastProject:      &lastProject,
		Projects:         projects,
		RequestedProject: req.ProjectName,
	}, nil
}

// GetIssue returns detailed information about a specific issue.
func (s *ProjectService) GetIssue(ctx context.Context, req *warnly.GetIssueRequest) (*warnly.IssueDetails, error) {
	project, err := s.GetProject(ctx, req.ProjectID, req.User)
	if err != nil {
		return nil, err
	}

	issue, err := s.issueStore.GetIssueByID(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	dur, err := warnly.ParseDuration(req.Period)
	if err != nil {
		return nil, err
	}

	to := s.now().UTC()
	from := to.Add(-dur)
	_ = from

	metrics, err := s.analyticsStore.ListIssueMetrics(
		ctx,
		&warnly.ListIssueMetricsCriteria{
			ProjectIDs: []int{project.ID},
			GroupIDs:   []int64{issue.ID},
			From:       issue.FirstSeen.Add(time.Minute * -10),
			To:         to,
		})
	if err != nil {
		return nil, err
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("issue metrics not found: %d", issue.ID)
	}
	metric := metrics[0]

	events, err := s.analyticsStore.CalculateEventsPerDay(
		ctx,
		warnly.EventDefCriteria{
			GroupID:   req.IssueID,
			ProjectID: issue.ProjectID,
			From:      to.Add(-30 * 24 * time.Hour),
			To:        to,
		})
	if err != nil {
		return nil, err
	}

	total30Days, total24Hours := calculateTotalEvents(events)

	isNew := issue.FirstSeen.After(to.Add(-7 * 24 * time.Hour))

	fieldCount, fieldValue, err := s.calculateFieldMetrics(ctx, req.IssueID, project.ID, issue.FirstSeen, to)
	if err != nil {
		return nil, err
	}

	lastEvent, err := s.analyticsStore.GetIssueEvent(
		ctx,
		warnly.EventDefCriteria{
			GroupID:   req.IssueID,
			ProjectID: project.ID,
			From:      metric.LastSeen.Add(-1 * time.Minute),
			To:        metric.LastSeen.Add(1 * time.Minute),
		})
	if err != nil {
		return nil, err
	}

	messagesCount, err := s.messageStore.CountMessages(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User:      req.User,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, err
	}

	assignments, err := s.buildTeammateAssigns(ctx, teammates, []warnly.Issue{*issue})
	if err != nil {
		return nil, err
	}

	return &warnly.IssueDetails{
		IssueID:       issue.ID,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		ErrorType:     issue.ErrorType,
		View:          issue.View,
		ErrorValue:    issue.Message,
		Message:       lastEvent.Message,
		Priority:      issue.Priority,
		IsNew:         isNew,
		FirstSeen:     metric.FirstSeen,
		LastSeen:      metric.LastSeen,
		TimesSeen:     metric.TimesSeen,
		UserCount:     metric.UserCount,
		Total30Days:   total30Days,
		Total24Hours:  total24Hours,
		TagCount:      fieldCount,
		TagValueNum:   fieldValue,
		LastEvent:     lastEvent,
		StackDetails:  warnly.GetStackDetails(lastEvent),
		Platform:      project.Platform,
		MessagesCount: messagesCount,
		Assignments:   assignments,
		Teammates:     teammates,
	}, nil
}

// ListTeammates returns a list of teammates associated with the user.
func (s *ProjectService) ListTeammates(
	ctx context.Context,
	req *warnly.ListTeammatesRequest,
) ([]warnly.Teammate, error) {
	teams, err := s.teamStore.ListTeams(ctx, int(req.User.ID))
	if err != nil {
		return nil, err
	}

	if len(teams) == 0 {
		return nil, warnly.ErrNotFound
	}

	teamIDS := extractTeamIDs(teams)

	teammates, err := s.teamStore.ListTeammates(ctx, teamIDS)
	if err != nil {
		return nil, err
	}

	return teammates, nil
}

// DeleteMessage deletes a user message from the issue discussion.
func (s *ProjectService) DeleteMessage(
	ctx context.Context,
	req *warnly.DeleteMessageRequest,
) (*warnly.Discussion, error) {
	project, err := s.GetProject(ctx, req.ProjectID, req.User)
	if err != nil {
		return nil, err
	}

	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User:      req.User,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, err
	}

	err = s.uow(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		if err := uw.Messages().DeleteMessage(ctx, req.MessageID, int(req.User.ID)); err != nil {
			return err
		}
		return uw.Mentions().DeleteMentions(ctx, req.MessageID)
	}, s.messageStore, s.mentionStore)
	if err != nil {
		return nil, err
	}

	messages, err := s.messageStore.ListIssueMessages(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	return &warnly.Discussion{
		Teammates: teammates,
		Messages:  messages,
		Info: warnly.DiscussionInfo{
			ProjectID: project.ID,
			IssueID:   req.IssueID,
		},
	}, nil
}

// CreateMessage creates a new message in the issue discussion.
func (s *ProjectService) CreateMessage(
	ctx context.Context,
	req *warnly.CreateMessageRequest,
) (*warnly.Discussion, error) {
	project, err := s.GetProject(ctx, req.ProjectID, req.User)
	if err != nil {
		return nil, err
	}

	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User:      req.User,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(req.MentionedUsers)
	mentioned := slices.Compact(req.MentionedUsers)

	content := s.sanitizerPolicy.Sanitize(req.Content)
	if content == "" {
		s.logger.Error("message empty or sanitizer didn't allow input",
			slog.String("content", req.Content),
			slog.Int64("user_id", req.User.ID),
			slog.String("user", req.User.Name))
		messages, err := s.messageStore.ListIssueMessages(ctx, int64(req.IssueID))
		if err != nil {
			return nil, err
		}
		return &warnly.Discussion{
			Teammates: teammates,
			Messages:  messages,
			Info: warnly.DiscussionInfo{
				ProjectID: project.ID,
				IssueID:   req.IssueID,
			},
		}, nil
	}
	req.Content = content

	err = s.uow(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		now := s.now().UTC()
		message := &warnly.Message{
			IssueID:   int64(req.IssueID),
			UserID:    int(req.User.ID),
			Content:   req.Content,
			CreatedAt: now,
		}
		if err := uw.Messages().CreateMessage(ctx, message); err != nil {
			return err
		}

		if len(mentioned) == 0 {
			return nil
		}

		mentions := make([]warnly.Mention, 0, len(mentioned))
		for i := range mentioned {
			mentions = append(mentions, warnly.Mention{
				MessageID:       message.ID,
				MentionedUserID: mentioned[i],
				CreatedAt:       now,
			})
		}

		return s.mentionStore.CreateMentions(ctx, mentions)
	}, s.messageStore, s.mentionStore)
	if err != nil {
		return nil, err
	}

	messages, err := s.messageStore.ListIssueMessages(ctx, int64(req.IssueID))
	if err != nil {
		return nil, err
	}

	return &warnly.Discussion{
		Teammates: teammates,
		Messages:  messages,
		Info: warnly.DiscussionInfo{
			ProjectID: project.ID,
			IssueID:   req.IssueID,
		},
	}, nil
}

// DeleteAssignment unassigns an issue from a user.
func (s *ProjectService) DeleteAssignment(ctx context.Context, req *warnly.UnassignIssueRequest) error {
	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User: req.User,
	})
	if err != nil {
		return err
	}
	if err := s.validateTeammate(teammates, int(req.User.ID)); err != nil {
		return err
	}

	return s.assingmentStore.DeleteAssignment(ctx, int64(req.IssueID))
}

// AssignIssue assigns an issue to a user.
func (s *ProjectService) AssignIssue(ctx context.Context, req *warnly.AssignIssueRequest) error {
	teammates, err := s.ListTeammates(ctx, &warnly.ListTeammatesRequest{
		User: req.User,
	})
	if err != nil {
		return err
	}
	if err := s.validateTeammate(teammates, req.UserID); err != nil {
		return err
	}

	now := s.now().UTC()
	assign := &warnly.Assignment{
		AssignedAt:       now,
		IssueID:          int64(req.IssueID),
		AssignedToUserID: int64(req.UserID),
		AssignedByUserID: req.User.ID,
	}

	return s.assingmentStore.CreateAssingment(ctx, assign)
}

// SearchProject searches for a project by name within the user's teams.
func (s *ProjectService) SearchProject(ctx context.Context, name string, user *warnly.User) (*warnly.Project, error) {
	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return nil, err
	}

	if len(teams) == 0 {
		return nil, warnly.ErrNotFound
	}

	teamIDs := make([]int, 0, len(teams))
	for i := range teams {
		teamIDs = append(teamIDs, teams[i].ID)
	}

	projects, err := s.projectStore.ListProjects(ctx, teamIDs, name)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, warnly.ErrProjectNotFound
	}

	for i := range projects {
		if projects[i].Name == name {
			return &projects[i], nil
		}
	}

	return nil, warnly.ErrProjectNotFound
}

// validateTeammate checks if a user is part of the teammates list.
func (s *ProjectService) validateTeammate(teammates []warnly.Teammate, userID int) error {
	for i := range teammates {
		if teammates[i].ID == int64(userID) {
			return nil
		}
	}
	return fmt.Errorf("user %d is not a teammate", userID)
}

// calculateFieldMetrics calculates field metrics for a given issue and project within a time range.
func (s *ProjectService) calculateFieldMetrics(
	ctx context.Context,
	issueID,
	projectID int,
	from,
	to time.Time,
) ([]warnly.TagCount, []warnly.FieldValueNum, error) {
	fieldCount, err := s.analyticsStore.CalculateFields(ctx, warnly.FieldsCriteria{
		IssueID:   issueID,
		ProjectID: projectID,
		From:      from,
		To:        to,
	})
	if err != nil {
		return nil, nil, err
	}

	fieldMap := make(map[string]uint64, len(fieldCount))
	for _, field := range fieldCount {
		fieldMap[field.Tag] = field.Count
	}

	fieldValue, err := s.analyticsStore.CountFields(ctx, warnly.EventDefCriteria{
		GroupID:   issueID,
		ProjectID: projectID,
		From:      from,
		To:        to,
	})
	if err != nil {
		return nil, nil, err
	}

	for i := range fieldValue {
		total, ok := fieldMap[fieldValue[i].Tag]
		if ok {
			fieldValue[i].PercentsOfTotal = (float64(fieldValue[i].Count) / float64(total)) * 100
		}
	}

	return fieldCount, fieldValue, nil
}

// calculateTotalEvents calculates the total number of events over the last 30 days and the last 24 hours.
func calculateTotalEvents(events []warnly.EventPerDay) (uint64, uint64) {
	var total30Days, total24Hours uint64
	if len(events) > 0 {
		for _, event := range events {
			total30Days += event.Count
		}
		total24Hours = events[0].Count
	}
	return total30Days, total24Hours
}

// extractTeamIDs extracts team identifiers from a list of teams.
func extractTeamIDs(teams []warnly.Team) []int {
	teamIDS := make([]int, 0, len(teams))
	for i := range teams {
		teamIDS = append(teamIDS, teams[i].ID)
	}
	return teamIDS
}

// extractIssueIDs extracts issue IDs from a list of issues.
func extractIssueIDs(issues []warnly.Issue) []int64 {
	ids := make([]int64, 0, len(issues))
	for i := range issues {
		ids = append(ids, issues[i].ID)
	}
	return ids
}

// buildIssueList builds a list of issue entries with their associated metrics and sorts them.
func (s *ProjectService) buildIssueList(
	ctx context.Context,
	projectIDS []int,
	issues []warnly.Issue,
	from,
	to time.Time,
) ([]warnly.IssueEntry, error) {
	ids := extractIssueIDs(issues)

	issueMetrics, err := s.analyticsStore.ListIssueMetrics(
		ctx,
		&warnly.ListIssueMetricsCriteria{
			ProjectIDs: projectIDS,
			GroupIDs:   ids,
			From:       from,
			To:         to,
		},
	)
	if err != nil {
		return nil, err
	}

	issueList := make([]warnly.IssueEntry, 0, len(issues))
	for i := range issues {
		metric, ok := warnly.GetMetrics(issueMetrics, issues[i].ID)
		if !ok {
			continue
		}
		iss := warnly.IssueEntry{
			ID:        issues[i].ID,
			Type:      issues[i].ErrorType,
			View:      issues[i].View,
			Message:   issues[i].Message,
			ProjectID: issues[i].ProjectID,
			LastSeen:  metric.LastSeen,
			FirstSeen: metric.FirstSeen,
			TimesSeen: metric.TimesSeen,
			UserCount: metric.UserCount,
		}
		issueList = append(issueList, iss)
	}

	compareFn := func(a, b warnly.IssueEntry) int {
		if a.TimesSeen == b.TimesSeen {
			return cmp.Compare(b.LastSeen.Unix(), a.LastSeen.Unix())
		}
		return cmp.Compare(b.TimesSeen, a.TimesSeen)
	}
	slices.SortFunc(issueList, compareFn)

	return issueList, nil
}

// listIssueEntries lists issue entries for a specific project along with their metrics.
func (s *ProjectService) listIssueEntries(
	ctx context.Context,
	projectID int,
	issues []warnly.Issue,
	from, to time.Time,
) ([]warnly.IssueEntry, error) {
	ids := make([]int64, len(issues))
	for i := range issues {
		ids[i] = issues[i].ID
	}

	issueMetrics, err := s.analyticsStore.ListIssueMetrics(
		ctx,
		&warnly.ListIssueMetricsCriteria{
			ProjectIDs: []int{projectID},
			GroupIDs:   ids,
			From:       from,
			To:         to,
		},
	)
	if err != nil {
		return nil, err
	}

	issueList := make([]warnly.IssueEntry, 0, len(issues))
	for i := range issues {
		metric, ok := warnly.GetMetrics(issueMetrics, issues[i].ID)
		if !ok {
			continue
		}
		issueList = append(issueList, warnly.IssueEntry{
			ID:        issues[i].ID,
			Type:      issues[i].ErrorType,
			View:      issues[i].View,
			Message:   issues[i].Message,
			LastSeen:  metric.LastSeen,
			FirstSeen: metric.FirstSeen,
			TimesSeen: metric.TimesSeen,
			UserCount: metric.UserCount,
		})
	}

	slices.SortFunc(issueList, func(a, b warnly.IssueEntry) int {
		if a.TimesSeen == b.TimesSeen {
			return cmp.Compare(b.LastSeen.Unix(), a.LastSeen.Unix())
		}
		return cmp.Compare(b.TimesSeen, a.TimesSeen)
	})

	return issueList, nil
}

// projectDSN constructs a DSN string for a project.
func projectDSN(projectID int, key, baseURL, scheme string) string {
	return fmt.Sprintf("%s://%s@%s/%d", scheme, key, baseURL+"/ingest", projectID)
}

// extractProjectIDs extracts project IDs from a list of projects, optionally filtering by project name.
func extractProjectIDs(projects []warnly.Project, projectName string) []int {
	projectIDS := make([]int, 0, len(projects))
	for i := range projects {
		if projectName != "" && projects[i].Name == projectName {
			return []int{projects[i].ID}
		}
		projectIDS = append(projectIDS, projects[i].ID)
	}
	return projectIDS
}

// mapEventsByProjectID maps events to their corresponding project IDs.
func mapEventsByProjectID(events []warnly.EventsPerHour) map[int][]warnly.EventsPerHour {
	eventsMap := make(map[int][]warnly.EventsPerHour, len(events))
	for _, event := range events {
		eventsMap[event.ProjectID] = append(eventsMap[event.ProjectID], event)
	}
	return eventsMap
}

// assignEventsToProjects assigns events to their corresponding projects.
func assignEventsToProjects(projects []warnly.Project, eventsMap map[int][]warnly.EventsPerHour) {
	for i := range projects {
		if events, ok := eventsMap[projects[i].ID]; ok {
			projects[i].Events = events
		}
	}
}

// populateMessagesCount populates the messages count for each issue in the list.
func (s *ProjectService) populateMessagesCount(
	ctx context.Context,
	issueList []warnly.IssueEntry,
) ([]warnly.IssueEntry, error) {
	ids := make([]int64, len(issueList))
	for i := range issueList {
		ids[i] = issueList[i].ID
	}
	messages, err := s.messageStore.CountMessagesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		for j := range issueList {
			if messages[i].IssueID == issueList[j].ID {
				issueList[j].MessagesCount = messages[i].MessageCount
				break
			}
		}
	}
	return issueList, nil
}

// buildTeammateAssigns builds a mapping of issues to their assigned teammates.
func (s *ProjectService) buildTeammateAssigns(
	ctx context.Context,
	teammates []warnly.Teammate,
	issues []warnly.Issue,
) (*warnly.Assignments, error) {
	assignments := &warnly.Assignments{
		IssueToAssigned: make(map[int64]*warnly.Teammate),
	}

	if len(issues) == 0 || len(teammates) == 0 {
		return assignments, nil
	}

	assignedUsers, err := s.assingmentStore.ListAssingments(ctx, extractIssueIDs(issues))
	if err != nil {
		return nil, err
	}

	// Build a map of teammate ID to *Teammate for quick lookup
	teammateMap := make(map[int64]*warnly.Teammate, len(teammates))
	for i := range teammates {
		teammate := teammates[i]
		teammateMap[teammate.ID] = &teammates[i]
	}

	for _, assigned := range assignedUsers {
		if assigned.AssignedToUserID.Valid {
			teammate, ok := teammateMap[assigned.AssignedToUserID.Int64]
			if ok {
				assignments.IssueToAssigned[assigned.IssueID] = teammate
			}
		}
	}

	return assignments, nil
}

// getTimeRange is a helper that returns the time range based on the request.
func (s *ProjectService) getTimeRange(req *warnly.ProjectDetailsRequest) (from, to time.Time, err error) {
	if req.Period != "" {
		dur, err := warnly.ParseDuration(req.Period)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		now := s.now().UTC()
		return now.Add(-dur), now, nil
	}

	return warnly.ParseTimeRange(req.Start, req.End)
}

// paginate is a helper that paginates the issue list based on the page and page size.
func paginate(issueList []warnly.IssueEntry, page, pageSize int) []warnly.IssueEntry {
	if page <= 0 || pageSize <= 0 {
		return issueList
	}
	start := (page - 1) * pageSize
	end := min(start+pageSize, len(issueList))
	if start >= len(issueList) {
		return nil
	}
	return issueList[start:end]
}

// filterRecentIssues filters and returns issues that have been first seen within the last 7 days from 'now'.
func filterRecentIssues(issueList []warnly.IssueEntry, now time.Time) []warnly.IssueEntry {
	res := make([]warnly.IssueEntry, 0, len(issueList))
	for i := range issueList {
		if issueList[i].FirstSeen.UTC().After(now.UTC().Add(-7 * 24 * time.Hour)) {
			res = append(res, issueList[i])
		}
	}
	return res
}
