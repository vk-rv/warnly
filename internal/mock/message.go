package mock

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// MessageStore is a mock implementation of warnly.MessageStore.
type MessageStore struct {
	CreateMessageFn      func(ctx context.Context, message *warnly.Message) error
	ListIssueMessagesFn  func(ctx context.Context, issueID int64) ([]warnly.IssueMessage, error)
	CountMessagesByIDsFn func(ctx context.Context, issueIDs []int64) ([]warnly.MessageCount, error)
	CountMessagesFn      func(ctx context.Context, issueID int64) (int, error)
	DeleteMessageFn      func(ctx context.Context, messageID, userID int) error
}

func (m *MessageStore) CreateMessage(ctx context.Context, message *warnly.Message) error {
	return m.CreateMessageFn(ctx, message)
}

func (m *MessageStore) ListIssueMessages(ctx context.Context, issueID int64) ([]warnly.IssueMessage, error) {
	return m.ListIssueMessagesFn(ctx, issueID)
}

func (m *MessageStore) CountMessagesByIDs(ctx context.Context, issueIDs []int64) ([]warnly.MessageCount, error) {
	return m.CountMessagesByIDsFn(ctx, issueIDs)
}

func (m *MessageStore) CountMessages(ctx context.Context, issueID int64) (int, error) {
	return m.CountMessagesFn(ctx, issueID)
}

func (m *MessageStore) DeleteMessage(ctx context.Context, messageID, userID int) error {
	return m.DeleteMessageFn(ctx, messageID, userID)
}
