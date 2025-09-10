package warnly

import (
	"context"
	"time"
)

// Message represents a text message in issue discussion (user comment).
type Message struct {
	CreatedAt time.Time `json:"created_at"`
	Content   string    `json:"content"`
	ID        int       `json:"id"`
	IssueID   int64     `json:"issue_id"`
	UserID    int       `json:"user_id"`
}

// IssueMessage represents a message in issue discussion with additional user information.
type IssueMessage struct {
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
}

// MessageStore encapsulates the methods to interact with database for message entity (issue comment).
type MessageStore interface {
	// CreateMessage creates a new message in the issue discussion.
	// Set identifier to pointer to the message created.
	CreateMessage(ctx context.Context, message *Message) error
	// ListIssueMessages lists all messages in the issue discussion.
	ListIssueMessages(ctx context.Context, issueID int64) ([]IssueMessage, error)
	// CountMessages counts all messages in the issue discussion.
	CountMessages(ctx context.Context, issueID int64) (int, error)
	// DeleteMessage deletes a message in the issue discussion.
	DeleteMessage(ctx context.Context, messageID, userID int) error
	// CountMessagesByIDs counts all messages in the issue discussion by IDs.
	CountMessagesByIDs(ctx context.Context, issueIDs []int64) ([]MessageCount, error)
}

// MessageCount represents a count of messages in the issue discussion.
type MessageCount struct {
	IssueID      int64 `json:"issue_id"`
	MessageCount int   `json:"message_count"`
}

// Mention represents a mention of a user in a message in issue discussion.
type Mention struct {
	CreatedAt       time.Time `json:"created_at"`
	ID              int       `json:"id"`
	MessageID       int       `json:"message_id"`
	MentionedUserID int       `json:"mentioned_user_id"`
}

// MentionStore encapsulates the methods to interact with database for Mention entity.
type MentionStore interface {
	// CreateMentions creates new mentions in issue discussion (when user was tagged with "@").
	CreateMentions(ctx context.Context, mentions []Mention) error
	// DeleteMentions deletes mentions in issue discussion.
	DeleteMentions(ctx context.Context, messageID int) error
}

// MessageView represents a view of a message by a user.
// It is used to track which users have viewed a message in issue discussion.
type MessageView struct {
	ViewedAt  time.Time `json:"viewed_at"`
	ID        int       `json:"id"`
	MessageID int       `json:"message_id"`
	UserID    int       `json:"user_id"`
}
