package project_test

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/mock"
	"github.com/vk-rv/warnly/internal/svc/project"
	"github.com/vk-rv/warnly/internal/warnly"
)

// TestNewProjectServiceReturnsValidService tests that NewProjectService creates a valid ProjectService.
func TestNewProjectServiceReturnsValidService(t *testing.T) {
	t.Parallel()

	projectStore := &mock.ProjectStore{}
	assingmentStore := &mock.AssingmentStore{}
	teamStore := &mock.TeamStore{}
	issueStore := &mock.IssueStore{}
	messageStore := &mock.MessageStore{}
	mentionStore := &mock.MentionStore{}
	analyticsStore := &mock.AnalyticsStore{}

	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	customTimeFunc := func() time.Time {
		return customTime
	}

	svc := project.NewProjectService(
		projectStore,
		assingmentStore,
		teamStore,
		issueStore,
		messageStore,
		mentionStore,
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"https",
		"example.com",
		"https",
		customTimeFunc,
		slog.Default(),
	)

	assert.NotNil(t, svc)
}

func TestCreateProjectSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := &warnly.User{ID: 1}
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		CreateProjectFn: func(_ context.Context, proj *warnly.Project) error {
			proj.ID = 1
			return nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	req := &warnly.CreateProjectRequest{
		ProjectName: "Test Project",
		TeamID:      teamID,
		Platform:    "go",
	}

	result, err := svc.CreateProject(ctx, req, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Project", result.Name)
	assert.Equal(t, 1, result.ID)
	assert.NotEmpty(t, result.DSN)
}

func TestDeleteProjectSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
		DeleteProjectFn: func(_ context.Context, _ int) error {
			return nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	err := svc.DeleteProject(ctx, projectID, user)

	assert.NoError(t, err)
}

func TestGetProjectSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	result, err := svc.GetProject(ctx, projectID, user)

	require.NoError(t, err)
	assert.Equal(t, projectID, result.ID)
	assert.Equal(t, teamID, result.TeamID)
	assert.Equal(t, "Test Project", result.Name)
}

func TestListProjectsSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: 1, TeamID: teamID, Name: "Project 1"},
				{ID: 2, TeamID: teamID, Name: "Project 2"},
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CalculateEventsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.EventsPerHour, error) {
			return []warnly.EventsPerHour{
				{ProjectID: 1, Count: 10},
				{ProjectID: 2, Count: 20},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	criteria := &warnly.ListProjectsCriteria{}
	result, err := svc.ListProjects(ctx, criteria, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Projects, 2)
	assert.Len(t, result.Teams, 1)
	assert.Equal(t, "Project 1", result.Projects[0].Name)
	assert.Equal(t, "Project 2", result.Projects[1].Name)
}

func TestListProjectsNoProjects(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	criteria := &warnly.ListProjectsCriteria{}
	result, err := svc.ListProjects(ctx, criteria, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Projects)
	assert.Len(t, result.Teams, 1)
}

func TestListProjectsNoEvents(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: 1, TeamID: teamID, Name: "Project 1"},
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CalculateEventsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.EventsPerHour, error) {
			return []warnly.EventsPerHour{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	criteria := &warnly.ListProjectsCriteria{}
	result, err := svc.ListProjects(ctx, criteria, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Projects, 1)
	assert.Len(t, result.Teams, 1)
}

func TestListTeamsSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, userID int) ([]warnly.Team, error) {
			assert.Equal(t, 1, userID)
			return []warnly.Team{
				{ID: 10, Name: "Team A"},
				{ID: 20, Name: "Team B"},
				{ID: 30, Name: "Team C"},
			}, nil
		},
	}

	svc := project.NewProjectService(
		&mock.ProjectStore{},
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	result, err := svc.ListTeams(ctx, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)
	assert.Equal(t, "Team A", result[0].Name)
	assert.Equal(t, "Team B", result[1].Name)
	assert.Equal(t, "Team C", result[2].Name)
}

func TestListTeamsUserNoTeams(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{}, nil
		},
	}

	svc := project.NewProjectService(
		&mock.ProjectStore{},
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	result, err := svc.ListTeams(ctx, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetProjectDetailsSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	customTimeFunc := func() time.Time {
		return customTime
	}

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		ListIssuesFn: func(_ context.Context, _ *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
			return []warnly.Issue{
				{
					ID:        1,
					ProjectID: projectID,
					ErrorType: "TypeError",
					Message:   "Test error",
					FirstSeen: customTime.Add(-1 * time.Hour),
				},
			}, nil
		},
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        1,
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: customTime.Add(-1 * time.Hour),
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CalculateEventsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.EventsPerHour, error) {
			return []warnly.EventsPerHour{
				{ProjectID: projectID, Count: 10},
			}, nil
		},
		ListIssueMetricsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.IssueMetrics, error) {
			return []warnly.IssueMetrics{
				{
					GID:       1,
					TimesSeen: 10,
					UserCount: 5,
					FirstSeen: customTime.Add(-1 * time.Hour),
					LastSeen:  customTime,
				},
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		CountMessagesByIDsFn: func(_ context.Context, _ []int64) ([]warnly.MessageCount, error) {
			return []warnly.MessageCount{
				{IssueID: 1, MessageCount: 3},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		messageStore,
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		customTimeFunc,
		slog.Default(),
	)

	req := &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesTypeAll,
		Period:    "24h",
	}

	result, err := svc.GetProjectDetails(ctx, req, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Project)
	assert.Equal(t, projectID, result.Project.ID)
	assert.Equal(t, "Test Project", result.Project.Name)
	assert.NotEmpty(t, result.Project.Events)
	assert.Equal(t, 1, result.Project.AllLength)
}

func TestGetProjectDetailsNoIssues(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		ListIssuesFn: func(_ context.Context, _ *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
			return []warnly.Issue{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	req := &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesTypeAll,
		Period:    "24h",
	}

	result, err := svc.GetProjectDetails(ctx, req, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.Project.ID)
	assert.Equal(t, 0, result.Project.AllLength)
}

func TestGetProjectDetailsWithTeammates(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	customTimeFunc := func() time.Time {
		return customTime
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		ListIssuesFn: func(_ context.Context, _ *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
			return []warnly.Issue{
				{
					ID:        1,
					ProjectID: projectID,
					ErrorType: "TypeError",
					Message:   "Test error",
					FirstSeen: customTime.Add(-1 * time.Hour),
				},
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CalculateEventsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.EventsPerHour, error) {
			return []warnly.EventsPerHour{
				{ProjectID: projectID, Count: 10},
			}, nil
		},
		ListIssueMetricsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.IssueMetrics, error) {
			return []warnly.IssueMetrics{
				{
					GID:       1,
					TimesSeen: 10,
					UserCount: 5,
					FirstSeen: customTime.Add(-1 * time.Hour),
					LastSeen:  customTime,
				},
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		CountMessagesByIDsFn: func(_ context.Context, _ []int64) ([]warnly.MessageCount, error) {
			return []warnly.MessageCount{
				{IssueID: 1, MessageCount: 3},
			}, nil
		},
	}

	assingmentStore := &mock.AssingmentStore{
		ListAssingmentsFn: func(_ context.Context, _ []int64) ([]*warnly.AssignedUser, error) {
			return []*warnly.AssignedUser{
				{
					IssueID:          1,
					AssignedToUserID: sql.NullInt64{Int64: 2, Valid: true},
				},
			}, nil
		},
	}

	teamStore2 := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 2, Name: "John Doe", Email: "john@example.com"},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		assingmentStore,
		teamStore2,
		issueStore,
		messageStore,
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		customTimeFunc,
		slog.Default(),
	)

	req := &warnly.ProjectDetailsRequest{
		ProjectID: projectID,
		Issues:    warnly.IssuesTypeAll,
		Period:    "24h",
	}

	result, err := svc.GetProjectDetails(ctx, req, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Teammates)
}

func TestGetDiscussionSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 2, Name: "John Doe", Email: "john@example.com"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		ListIssueMessagesFn: func(_ context.Context, _ int64) ([]warnly.IssueMessage, error) {
			return []warnly.IssueMessage{
				{ID: 1, Content: "First message", Username: "user1"},
				{ID: 2, Content: "Second message", Username: "user2"},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		messageStore,
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	req := &warnly.GetDiscussionsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      user,
	}

	result, err := svc.GetDiscussion(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Teammates)
	assert.Len(t, result.Teammates, 1)
	assert.Len(t, result.Messages, 2)
	assert.Equal(t, projectID, result.Info.ProjectID)
	assert.Equal(t, issueID, result.Info.IssueID)
	assert.Equal(t, "John Doe", result.Teammates[0].Name)
	assert.Equal(t, "First message", result.Messages[0].Content)
	assert.Equal(t, "Second message", result.Messages[1].Content)
}

func TestGetDiscussionNoMessages(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 2, Name: "John Doe", Email: "john@example.com"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		ListIssueMessagesFn: func(_ context.Context, _ int64) ([]warnly.IssueMessage, error) {
			return []warnly.IssueMessage{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		messageStore,
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	req := &warnly.GetDiscussionsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      user,
	}

	result, err := svc.GetDiscussion(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Messages)
	assert.Len(t, result.Teammates, 1)
}

func TestListFieldsSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: customTime.Add(-24 * time.Hour),
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CalculateFieldsFn: func(_ context.Context, _ warnly.FieldsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{
				{Tag: "browser", Count: 100},
				{Tag: "os", Count: 50},
			}, nil
		},
		CountFieldsFn: func(_ context.Context, _ *warnly.EventDefCriteria) ([]warnly.FieldValueNum, error) {
			return []warnly.FieldValueNum{
				{
					Tag:             "browser",
					Value:           "Chrome",
					Count:           60,
					PercentsOfTotal: 0,
					FirstSeen:       customTime.Add(-24 * time.Hour),
					LastSeen:        customTime,
				},
				{
					Tag:             "browser",
					Value:           "Firefox",
					Count:           40,
					PercentsOfTotal: 0,
					FirstSeen:       customTime.Add(-24 * time.Hour),
					LastSeen:        customTime,
				},
				{
					Tag:             "os",
					Value:           "Linux",
					Count:           30,
					PercentsOfTotal: 0,
					FirstSeen:       customTime.Add(-24 * time.Hour),
					LastSeen:        customTime,
				},
				{
					Tag:             "os",
					Value:           "Windows",
					Count:           20,
					PercentsOfTotal: 0,
					FirstSeen:       customTime.Add(-24 * time.Hour),
					LastSeen:        customTime,
				},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListFieldsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      user,
	}

	result, err := svc.ListFields(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Project", result.ProjectName)
	assert.Len(t, result.TagCount, 2)
	assert.Equal(t, "browser", result.TagCount[0].Tag)
	assert.Equal(t, uint64(100), result.TagCount[0].Count)
	assert.Equal(t, "os", result.TagCount[1].Tag)
	assert.Equal(t, uint64(50), result.TagCount[1].Count)
	assert.Len(t, result.FieldValueNum, 4)
	assert.InEpsilon(t, 60.0, result.FieldValueNum[0].PercentsOfTotal, 0.01)
	assert.InEpsilon(t, 40.0, result.FieldValueNum[1].PercentsOfTotal, 0.01)
	assert.InEpsilon(t, 60.0, result.FieldValueNum[2].PercentsOfTotal, 0.01)
	assert.InEpsilon(t, 40.0, result.FieldValueNum[3].PercentsOfTotal, 0.01)
}

func TestListFieldsNoFields(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: customTime.Add(-24 * time.Hour),
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CalculateFieldsFn: func(_ context.Context, _ warnly.FieldsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{}, nil
		},
		CountFieldsFn: func(_ context.Context, _ *warnly.EventDefCriteria) ([]warnly.FieldValueNum, error) {
			return []warnly.FieldValueNum{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListFieldsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		User:      user,
	}

	result, err := svc.ListFields(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Project", result.ProjectName)
	assert.Empty(t, result.TagCount)
	assert.Empty(t, result.FieldValueNum)
}

func TestListEventsSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: customTime.Add(-24 * time.Hour),
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CountEventsFn: func(_ context.Context, _ *warnly.EventCriteria) (uint64, error) {
			return 42, nil
		},
		ListEventsFn: func(_ context.Context, _ *warnly.EventCriteria) ([]warnly.EventEntry, error) {
			return []warnly.EventEntry{
				{
					CreatedAt:    customTime.Add(-10 * time.Hour),
					EventID:      "event-1",
					Title:        "TypeError",
					Message:      "Cannot read property 'x' of undefined",
					Release:      "1.0.0",
					Env:          "production",
					UserEmail:    "user@example.com",
					UserUsername: "john_doe",
				},
				{
					CreatedAt:    customTime.Add(-5 * time.Hour),
					EventID:      "event-2",
					Title:        "TypeError",
					Message:      "Cannot read property 'x' of undefined",
					Release:      "1.0.0",
					Env:          "staging",
					UserEmail:    "admin@example.com",
					UserUsername: "admin",
				},
			}, nil
		},
		CalculateFieldsFn: func(_ context.Context, _ warnly.FieldsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListEventsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		Query:     "",
		Offset:    0,
		User:      user,
	}

	result, err := svc.ListEvents(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.ProjectID)
	assert.Equal(t, issueID, result.IssueID)
	assert.Equal(t, uint64(42), result.TotalEvents)
	assert.Equal(t, 0, result.Offset)
	assert.Len(t, result.Events, 2)
	assert.Equal(t, "event-1", result.Events[0].EventID)
	assert.Equal(t, "user@example.com", result.Events[0].UserEmail)
	assert.Equal(t, "event-2", result.Events[1].EventID)
}

func TestListEventsNoEvents(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "TypeError",
				Message:   "Test error",
				FirstSeen: customTime.Add(-24 * time.Hour),
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		CountEventsFn: func(_ context.Context, _ *warnly.EventCriteria) (uint64, error) {
			return 0, nil
		},
		ListEventsFn: func(_ context.Context, _ *warnly.EventCriteria) ([]warnly.EventEntry, error) {
			return []warnly.EventEntry{}, nil
		},
		CalculateFieldsFn: func(_ context.Context, _ warnly.FieldsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListEventsRequest{
		ProjectID: projectID,
		IssueID:   issueID,
		Query:     "",
		Offset:    0,
		User:      user,
	}

	result, err := svc.ListEvents(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(0), result.TotalEvents)
	assert.Empty(t, result.Events)
}

func TestListIssuesSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10
	projectID := 5
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: projectID, TeamID: teamID, Name: "Test Project"},
				{ID: 6, TeamID: teamID, Name: "Project B"},
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		ListIssuesFn: func(_ context.Context, _ *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
			return []warnly.Issue{
				{
					ID:        1,
					ProjectID: projectID,
					ErrorType: "TypeError",
					Message:   "Cannot read property 'x' of undefined",
					FirstSeen: customTime.Add(-24 * time.Hour),
					View:      "home",
				},
				{
					ID:        2,
					ProjectID: projectID,
					ErrorType: "ReferenceError",
					Message:   "y is not defined",
					FirstSeen: customTime.Add(-12 * time.Hour),
					View:      "dashboard",
				},
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		ListIssueMetricsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.IssueMetrics, error) {
			return []warnly.IssueMetrics{
				{GID: 1, TimesSeen: 50, UserCount: 10, FirstSeen: customTime.Add(-24 * time.Hour), LastSeen: customTime},
				{GID: 2, TimesSeen: 30, UserCount: 8, FirstSeen: customTime.Add(-12 * time.Hour), LastSeen: customTime},
			}, nil
		},
		ListPopularTagsFn: func(_ context.Context, _ *warnly.ListPopularTagsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{
				{Tag: "browser", Count: 100},
				{Tag: "os", Count: 50},
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		CountMessagesByIDsFn: func(_ context.Context, _ []int64) ([]warnly.MessageCount, error) {
			return []warnly.MessageCount{
				{IssueID: 1, MessageCount: 5},
				{IssueID: 2, MessageCount: 3},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		messageStore,
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListIssuesRequest{
		User:   user,
		Period: "24h",
	}

	result, err := svc.ListIssues(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Issues, 2)
	assert.Len(t, result.Projects, 2)
	assert.Len(t, result.PopularTags, 2)
	assert.Equal(t, 2, result.TotalIssues)
	assert.NotNil(t, result.LastProject)
	assert.Equal(t, "Cannot read property 'x' of undefined", result.Issues[0].Message)
	assert.Equal(t, uint64(50), result.Issues[0].TimesSeen)
	assert.Equal(t, 5, result.Issues[0].MessagesCount)
}

func TestListIssuesNoIssues(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10
	projectID := 5
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: projectID, TeamID: teamID, Name: "Test Project"},
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		ListIssuesFn: func(_ context.Context, _ *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
			return []warnly.Issue{}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		ListPopularTagsFn: func(_ context.Context, _ *warnly.ListPopularTagsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{
				{Tag: "browser", Count: 100},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		issueStore,
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListIssuesRequest{
		User:   user,
		Period: "24h",
	}

	result, err := svc.ListIssues(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.Issues)
	assert.Len(t, result.Projects, 1)
	assert.NotNil(t, result.LastProject)
	assert.Equal(t, 0, result.TotalIssues)
}

func TestDeleteMessageSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	teamID := 10
	issueID := 100
	messageID := 42
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		DeleteMessageFn: func(_ context.Context, _ int, _ int) error {
			return nil
		},
		ListIssueMessagesFn: func(_ context.Context, _ int64) ([]warnly.IssueMessage, error) {
			return []warnly.IssueMessage{
				{
					ID:        1,
					UserID:    2,
					Username:  "Jane Smith",
					Content:   "First message",
					CreatedAt: customTime.Add(-2 * time.Hour),
				},
				{
					ID:        2,
					UserID:    1,
					Username:  "John Doe",
					Content:   "Second message",
					CreatedAt: customTime.Add(-1 * time.Hour),
				},
			}, nil
		},
	}

	mentionStore := &mock.MentionStore{
		DeleteMentionsFn: func(_ context.Context, _ int) error {
			return nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		messageStore,
		mentionStore,
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.DeleteMessageRequest{
		User:      user,
		ProjectID: projectID,
		IssueID:   issueID,
		MessageID: messageID,
	}

	result, err := svc.DeleteMessage(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.Info.ProjectID)
	assert.Equal(t, issueID, result.Info.IssueID)
	assert.Len(t, result.Teammates, 2)
	assert.Len(t, result.Messages, 2)
}

func TestCreateMessageSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1, Name: "John Doe"}
	projectID := 5
	teamID := 10
	issueID := 100
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		ListIssueMessagesFn: func(_ context.Context, _ int64) ([]warnly.IssueMessage, error) {
			return []warnly.IssueMessage{
				{
					ID:        1,
					UserID:    1,
					Username:  "John Doe",
					Content:   "Test message",
					CreatedAt: customTime,
				},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		messageStore,
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.CreateMessageRequest{
		User:           user,
		Content:        "<p>This is a valid message</p>",
		MentionedUsers: []int{2},
		ProjectID:      projectID,
		IssueID:        issueID,
	}

	result, err := svc.CreateMessage(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.Info.ProjectID)
	assert.Equal(t, issueID, result.Info.IssueID)
	assert.Len(t, result.Teammates, 2)
	assert.Len(t, result.Messages, 1)
}

func TestCreateMessageWithMentions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1, Name: "John Doe"}
	projectID := 5
	teamID := 10
	issueID := 100
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
				{ID: 3, Name: "Bob Johnson", Email: "bob@example.com"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:     projectID,
				TeamID: teamID,
				Name:   "Test Project",
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		ListIssueMessagesFn: func(_ context.Context, _ int64) ([]warnly.IssueMessage, error) {
			return []warnly.IssueMessage{
				{
					ID:        1,
					UserID:    1,
					Username:  "John Doe",
					Content:   "Mentioning @Jane Smith and @Bob Johnson",
					CreatedAt: customTime,
				},
			}, nil
		},
	}

	mentionStore := &mock.MentionStore{
		CreateMentionsFn: func(_ context.Context, _ []warnly.Mention) error {
			return nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		messageStore,
		mentionStore,
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.CreateMessageRequest{
		User:           user,
		Content:        "<p>Mentioning @Jane Smith and @Bob Johnson</p>",
		MentionedUsers: []int{2, 3},
		ProjectID:      projectID,
		IssueID:        issueID,
	}

	result, err := svc.CreateMessage(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.Info.ProjectID)
	assert.Equal(t, issueID, result.Info.IssueID)
	assert.Len(t, result.Teammates, 3)
	assert.Len(t, result.Messages, 1)
}

func TestListTagValuesSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: 1, TeamID: teamID, Name: "Test Project"},
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		ListTagValuesFn: func(_ context.Context, _ *warnly.ListTagValuesCriteria) ([]warnly.TagValueCount, error) {
			return []warnly.TagValueCount{
				{Value: "Chrome", Count: 150},
				{Value: "Firefox", Count: 80},
				{Value: "Safari", Count: 45},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListTagValuesRequest{
		User:   user,
		Tag:    "browser",
		Period: "24h",
		Limit:  10,
	}

	result, err := svc.ListTagValues(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)
	assert.Equal(t, "Chrome", result[0].Value)
	assert.Equal(t, uint64(150), result[0].Count)
	assert.Equal(t, "Firefox", result[1].Value)
	assert.Equal(t, uint64(80), result[1].Count)
	assert.Equal(t, "Safari", result[2].Value)
	assert.Equal(t, uint64(45), result[2].Count)
}

func TestListTagValuesNoResults(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: 1, TeamID: teamID, Name: "Test Project"},
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		ListTagValuesFn: func(_ context.Context, _ *warnly.ListTagValuesCriteria) ([]warnly.TagValueCount, error) {
			return []warnly.TagValueCount{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.ListTagValuesRequest{
		User:   user,
		Tag:    "browser",
		Period: "24h",
		Limit:  10,
	}

	result, err := svc.ListTagValues(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestSearchProjectSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10
	projectID := 5
	projectName := "Test Project"

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, userID int) ([]warnly.Team, error) {
			assert.Equal(t, 1, userID)
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, teamIDs []int, name string) ([]warnly.Project, error) {
			assert.Len(t, teamIDs, 1)
			assert.Equal(t, teamID, teamIDs[0])
			assert.Equal(t, projectName, name)
			return []warnly.Project{
				{ID: projectID, TeamID: teamID, Name: projectName},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	result, err := svc.SearchProject(ctx, projectName, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.ID)
	assert.Equal(t, teamID, result.TeamID)
	assert.Equal(t, projectName, result.Name)
}

func TestSearchProjectMultipleProjects(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10
	projectID := 5
	projectName := "Backend API"

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{
				{ID: projectID, TeamID: teamID, Name: "Backend API"},
				{ID: 6, TeamID: teamID, Name: "Frontend Web"},
				{ID: 7, TeamID: teamID, Name: "Mobile App"},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	result, err := svc.SearchProject(ctx, projectName, user)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, projectID, result.ID)
	assert.Equal(t, projectName, result.Name)
}

func TestSearchProjectNotFound(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	teamID := 10

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		ListProjectsFn: func(_ context.Context, _ []int, _ string) ([]warnly.Project, error) {
			return []warnly.Project{}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		&mock.AssingmentStore{},
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	result, err := svc.SearchProject(ctx, "Nonexistent Project", user)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, warnly.ErrProjectNotFound, err)
}

func TestGetIssueSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	projectID := 5
	issueID := 100
	teamID := 10
	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	issueFirstSeen := customTime.Add(-30 * 24 * time.Hour)
	lastSeen := customTime.Add(-1 * time.Hour)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: teamID, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			}, nil
		},
	}

	projectStore := &mock.ProjectStore{
		GetProjectFn: func(_ context.Context, _ int) (*warnly.Project, error) {
			return &warnly.Project{
				ID:       projectID,
				TeamID:   teamID,
				Name:     "Test Project",
				Platform: warnly.PlatformGolang,
			}, nil
		},
	}

	issueStore := &mock.IssueStore{
		GetIssueByIDFn: func(_ context.Context, _ int64) (*warnly.Issue, error) {
			return &warnly.Issue{
				ID:        int64(issueID),
				ProjectID: projectID,
				ErrorType: "RuntimeError",
				Message:   "Division by zero",
				View:      "main.go:42",
				FirstSeen: issueFirstSeen,
				Priority:  warnly.PriorityHigh,
			}, nil
		},
	}

	analyticsStore := &mock.AnalyticsStore{
		ListIssueMetricsFn: func(_ context.Context, _ *warnly.ListIssueMetricsCriteria) ([]warnly.IssueMetrics, error) {
			return []warnly.IssueMetrics{
				{
					GID:       uint64(issueID),
					FirstSeen: issueFirstSeen,
					LastSeen:  lastSeen,
					TimesSeen: 150,
					UserCount: 25,
				},
			}, nil
		},
		CalculateEventsPerDayFn: func(_ context.Context, _ *warnly.EventDefCriteria) ([]warnly.EventPerDay, error) {
			return []warnly.EventPerDay{
				{Time: customTime, GID: uint64(issueID), Count: 25},
				{Time: customTime.Add(-1 * 24 * time.Hour), GID: uint64(issueID), Count: 50},
			}, nil
		},
		CalculateFieldsFn: func(_ context.Context, _ warnly.FieldsCriteria) ([]warnly.TagCount, error) {
			return []warnly.TagCount{
				{Tag: "browser", Count: 60},
				{Tag: "os", Count: 150},
			}, nil
		},
		CountFieldsFn: func(_ context.Context, _ *warnly.EventDefCriteria) ([]warnly.FieldValueNum, error) {
			return []warnly.FieldValueNum{
				{Tag: "browser", Value: "Chrome", Count: 40, PercentsOfTotal: 66.67},
				{Tag: "browser", Value: "Firefox", Count: 20, PercentsOfTotal: 33.33},
				{Tag: "os", Value: "Windows", Count: 90, PercentsOfTotal: 60},
				{Tag: "os", Value: "macOS", Count: 60, PercentsOfTotal: 40},
			}, nil
		},
		GetIssueEventFn: func(_ context.Context, _ *warnly.EventDefCriteria) (*warnly.IssueEvent, error) {
			return &warnly.IssueEvent{
				EventID:   "event-123",
				UserID:    "user-456",
				UserEmail: "test@example.com",
				UserName:  "Test User",
				Message:   "Division by zero at line 42",
				TagsKey:   []string{"browser", "os"},
				TagsValue: []string{"Chrome", "Windows"},
			}, nil
		},
	}

	messageStore := &mock.MessageStore{
		CountMessagesFn: func(_ context.Context, _ int64) (int, error) {
			return 3, nil
		},
	}

	assignmentStore := &mock.AssingmentStore{
		ListAssingmentsFn: func(_ context.Context, _ []int64) ([]*warnly.AssignedUser, error) {
			return []*warnly.AssignedUser{
				{
					IssueID:          int64(issueID),
					AssignedToUserID: sql.NullInt64{Int64: 1, Valid: true},
				},
			}, nil
		},
	}

	svc := project.NewProjectService(
		projectStore,
		assignmentStore,
		teamStore,
		issueStore,
		messageStore,
		&mock.MentionStore{},
		analyticsStore,
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.GetIssueRequest{
		User:      user,
		ProjectID: projectID,
		IssueID:   issueID,
		Period:    "24h",
		EventID:   "event-123",
	}

	result, err := svc.GetIssue(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(issueID), result.IssueID)
	assert.Equal(t, projectID, result.ProjectID)
	assert.Equal(t, "Test Project", result.ProjectName)
	assert.Equal(t, "RuntimeError", result.ErrorType)
	assert.Equal(t, "Division by zero", result.ErrorValue)
	assert.Equal(t, "main.go:42", result.View)
	assert.Equal(t, warnly.PriorityHigh, result.Priority)
	assert.Equal(t, issueFirstSeen, result.FirstSeen)
	assert.Equal(t, lastSeen, result.LastSeen)
	assert.Equal(t, uint64(150), result.TimesSeen)
	assert.Equal(t, uint64(25), result.UserCount)
	assert.Equal(t, uint64(25), result.Total24Hours)
	assert.Equal(t, uint64(75), result.Total30Days)
	assert.Equal(t, 3, result.MessagesCount)
	assert.Equal(t, warnly.PlatformGolang, result.Platform)
	assert.Len(t, result.Teammates, 2)
	assert.NotNil(t, result.Assignments)
	assert.NotNil(t, result.LastEvent)
	assert.Equal(t, "event-123", result.LastEvent.EventID)
	assert.False(t, result.IsNew)
}

func TestAssignIssueSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	targetUserID := 2
	projectID := 5
	issueID := 100
	customTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: 10, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			}, nil
		},
	}

	assignmentStore := &mock.AssingmentStore{
		CreateAssingmentFn: func(_ context.Context, assignment *warnly.Assignment) error {
			assert.Equal(t, int64(issueID), assignment.IssueID)
			assert.Equal(t, int64(targetUserID), assignment.AssignedToUserID)
			assert.Equal(t, int64(1), assignment.AssignedByUserID)
			assert.Equal(t, customTime, assignment.AssignedAt)
			return nil
		},
	}

	svc := project.NewProjectService(
		&mock.ProjectStore{},
		assignmentStore,
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		func() time.Time { return customTime },
		slog.Default(),
	)

	req := &warnly.AssignIssueRequest{
		User:      user,
		ProjectID: projectID,
		IssueID:   issueID,
		UserID:    targetUserID,
	}

	err := svc.AssignIssue(ctx, req)

	assert.NoError(t, err)
}

func TestDeleteAssignmentSuccess(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	user := &warnly.User{ID: 1}
	issueID := 100

	teamStore := &mock.TeamStore{
		ListTeamsFn: func(_ context.Context, _ int) ([]warnly.Team, error) {
			return []warnly.Team{
				{ID: 10, Name: "Team A"},
			}, nil
		},
		ListTeammatesFn: func(_ context.Context, _ []int) ([]warnly.Teammate, error) {
			return []warnly.Teammate{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
			}, nil
		},
	}

	assignmentStore := &mock.AssingmentStore{
		DeleteAssignmentFn: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(issueID), id)
			return nil
		},
	}

	svc := project.NewProjectService(
		&mock.ProjectStore{},
		assignmentStore,
		teamStore,
		&mock.IssueStore{},
		&mock.MessageStore{},
		&mock.MentionStore{},
		&mock.AnalyticsStore{},
		mock.StartUnitOfWork,
		bluemonday.NewPolicy(),
		"localhost:8080",
		"http",
		"localhost:8080",
		"http",
		time.Now,
		slog.Default(),
	)

	req := &warnly.UnassignIssueRequest{
		User:      user,
		ProjectID: 5,
		IssueID:   issueID,
	}

	err := svc.DeleteAssignment(ctx, req)

	assert.NoError(t, err)
}
