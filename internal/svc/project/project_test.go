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
