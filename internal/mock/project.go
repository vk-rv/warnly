package mock

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// ProjectStore is a mock implementation of warnly.ProjectStore.
type ProjectStore struct {
	CreateProjectFn func(ctx context.Context, project *warnly.Project) error
	GetProjectFn    func(ctx context.Context, projectID int) (*warnly.Project, error)
	DeleteProjectFn func(ctx context.Context, projectID int) error
	ListProjectsFn  func(ctx context.Context, teamIDs []int, name string) ([]warnly.Project, error)
	GetOptionsFn    func(ctx context.Context, projectID int, projectKey string) (*warnly.ProjectOptions, error)
}

func (m *ProjectStore) CreateProject(ctx context.Context, proj *warnly.Project) error {
	return m.CreateProjectFn(ctx, proj)
}

func (m *ProjectStore) GetProject(ctx context.Context, projectID int) (*warnly.Project, error) {
	return m.GetProjectFn(ctx, projectID)
}

func (m *ProjectStore) DeleteProject(ctx context.Context, projectID int) error {
	return m.DeleteProjectFn(ctx, projectID)
}

func (m *ProjectStore) ListProjects(ctx context.Context, teamIDs []int, name string) ([]warnly.Project, error) {
	return m.ListProjectsFn(ctx, teamIDs, name)
}

func (m *ProjectStore) GetOptions(ctx context.Context, projectID int, projectKey string) (*warnly.ProjectOptions, error) {
	return m.GetOptionsFn(ctx, projectID, projectKey)
}
