package mock

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// MentionStore is a mock implementation of warnly.MentionStore.
type MentionStore struct {
	CreateMentionsFn func(ctx context.Context, mentions []warnly.Mention) error
	DeleteMentionsFn func(ctx context.Context, messageID int) error
}

func (m *MentionStore) CreateMentions(ctx context.Context, mentions []warnly.Mention) error {
	return m.CreateMentionsFn(ctx, mentions)
}

func (m *MentionStore) DeleteMentions(ctx context.Context, messageID int) error {
	return m.DeleteMentionsFn(ctx, messageID)
}
