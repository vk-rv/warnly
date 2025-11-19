package project_test

import (
	"context"
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
