package mock

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AssingmentStore is a mock implementation of warnly.AssingmentStore.
type AssingmentStore struct {
	ListAssingmentsFn     func(ctx context.Context, issueIDs []int64) ([]*warnly.AssignedUser, error)
	CreateAssingmentFn    func(ctx context.Context, assignment *warnly.Assignment) error
	DeleteAssignmentFn    func(ctx context.Context, issueID int64) error
	ListAssignedFiltersFn func(ctx context.Context, criteria *warnly.GetAssignedFiltersCriteria) ([]warnly.Filter, error)
}

func (m *AssingmentStore) ListAssingments(ctx context.Context, issueIDs []int64) ([]*warnly.AssignedUser, error) {
	return m.ListAssingmentsFn(ctx, issueIDs)
}

func (m *AssingmentStore) CreateAssingment(ctx context.Context, assignment *warnly.Assignment) error {
	return m.CreateAssingmentFn(ctx, assignment)
}

func (m *AssingmentStore) DeleteAssignment(ctx context.Context, issueID int64) error {
	return m.DeleteAssignmentFn(ctx, issueID)
}

func (m *AssingmentStore) ListAssignedFilters(
	ctx context.Context,
	criteria *warnly.GetAssignedFiltersCriteria,
) ([]warnly.Filter, error) {
	return m.ListAssignedFiltersFn(ctx, criteria)
}
