package mock

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// IssueStore is a mock implementation of warnly.IssueStore.
type IssueStore struct {
	StoreIssueFn     func(ctx context.Context, issue *warnly.Issue) error
	GetIssueByIDFn   func(ctx context.Context, id int64) (*warnly.Issue, error)
	ListIssuesFn     func(ctx context.Context, criteria *warnly.ListIssuesCriteria) ([]warnly.Issue, error)
	UpdateLastSeenFn func(ctx context.Context, upd *warnly.UpdateLastSeen) error
	GetIssueFn       func(ctx context.Context, criteria warnly.GetIssueCriteria) (*warnly.Issue, error)
}

func (m *IssueStore) StoreIssue(ctx context.Context, issue *warnly.Issue) error {
	return m.StoreIssueFn(ctx, issue)
}

func (m *IssueStore) GetIssueByID(ctx context.Context, id int64) (*warnly.Issue, error) {
	return m.GetIssueByIDFn(ctx, id)
}

func (m *IssueStore) ListIssues(ctx context.Context, criteria *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
	return m.ListIssuesFn(ctx, criteria)
}

func (m *IssueStore) UpdateLastSeen(ctx context.Context, upd *warnly.UpdateLastSeen) error {
	return m.UpdateLastSeenFn(ctx, upd)
}

func (m *IssueStore) GetIssue(ctx context.Context, criteria warnly.GetIssueCriteria) (*warnly.Issue, error) {
	return m.GetIssueFn(ctx, criteria)
}
