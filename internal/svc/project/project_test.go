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
