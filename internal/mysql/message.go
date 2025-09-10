package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vk-rv/warnly/internal/warnly"
)

// MessageStore implements warnly.MessageStore for MySQL.
type MessageStore struct {
	db ExtendedDB
}

// NewMessageStore is a constructor of MessageStore repository.
func NewMessageStore(db ExtendedDB) *MessageStore {
	return &MessageStore{db: db}
}

// CreateMessage creates a new message when user comments issue.
func (s *MessageStore) CreateMessage(ctx context.Context, m *warnly.Message) error {
	const query = `INSERT INTO message (issue_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`

	res, err := s.db.ExecContext(
		ctx,
		query,
		m.IssueID,
		m.UserID,
		m.Content,
		m.CreatedAt)
	if err != nil {
		return fmt.Errorf("mysql message store: create message: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("mysql message store: create message: no rows affected")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql message store: create message last insert id: %w", err)
	}
	m.ID = int(id)

	return nil
}

// CountMessages counts all messages in the issue discussion.
func (s *MessageStore) CountMessages(ctx context.Context, issueID int64) (int, error) {
	const query = `SELECT COUNT(*) FROM message WHERE issue_id = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, issueID).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("mysql message store: count messages: %w", err)
	}
	return count, nil
}

// ListIssueMessages is a method that lists all messages (comments) in the issue discussion.
func (s *MessageStore) ListIssueMessages(ctx context.Context, issueID int64) ([]warnly.IssueMessage, error) {
	const query = `SELECT m.id, u.name, m.user_id, m.content, m.created_at
		FROM message AS m
		JOIN user AS u ON m.user_id = u.id
		WHERE m.issue_id = ? ORDER BY m.created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, issueID)
	if err != nil {
		return nil, fmt.Errorf("mysql message store: list issue messages: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("mysql message store: list issue messages: %w", cerr)
		}
	}()

	return scan(rows, scanIssueMessage)
}

// scanIssueMessage scans a single issue message from the given sql.Rows.
func scanIssueMessage(rows *sql.Rows) (warnly.IssueMessage, error) {
	var m warnly.IssueMessage
	if err := rows.Scan(&m.ID, &m.Username, &m.UserID, &m.Content, &m.CreatedAt); err != nil {
		return m, fmt.Errorf("mysql message store: scan issue message: %w", err)
	}
	return m, nil
}

// MentionStore implements warnly.MentionStore for MySQL.
type MentionStore struct {
	db ExtendedDB
}

// NewMentionStore is a constructor of MentionStore.
func NewMentionStore(db ExtendedDB) *MentionStore {
	return &MentionStore{db: db}
}

// CreateMentions creates new mentions in issue discussion (when user was tagged using "@").
func (s *MentionStore) CreateMentions(ctx context.Context, mentions []warnly.Mention) error {
	query := `INSERT INTO mention (message_id, mentioned_user_id, created_at) VALUES `
	values := make([]any, 0, len(mentions)*3)

	for i, mention := range mentions {
		if i > 0 {
			query += ", "
		}
		query += "(?, ?, ?)"
		values = append(values, mention.MessageID, mention.MentionedUserID, mention.CreatedAt)
	}

	_, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("mysql mention store: create mentions: %w", err)
	}

	return nil
}

// DeleteMentions deletes all mentions from issue comment.
func (s *MentionStore) DeleteMentions(ctx context.Context, messageID int) error {
	const query = `DELETE FROM mention WHERE message_id = ?`
	_, err := s.db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("mysql mention store: delete mentions: %w", err)
	}
	return nil
}

// DeleteMessage deletes a message in the issue discussion.
func (s *MessageStore) DeleteMessage(ctx context.Context, messageID, userID int) error {
	const query = `DELETE FROM message WHERE id = ? AND user_id = ?`
	_, err := s.db.ExecContext(ctx, query, messageID, userID)
	if err != nil {
		return fmt.Errorf("mysql message store: delete message: %w", err)
	}
	return nil
}

// CountMessagesByIDs counts all messages in the issue discussion by IDs.
func (s *MessageStore) CountMessagesByIDs(ctx context.Context, issueIDs []int64) ([]warnly.MessageCount, error) {
	if len(issueIDs) == 0 {
		return nil, nil
	}

	const baseQuery = `SELECT i.id AS issue_id, COUNT(m.id) AS message_count
					   FROM issue i
					   LEFT JOIN message m ON i.id = m.issue_id`
	query := baseQuery + " WHERE i.id IN ("
	args := make([]any, len(issueIDs))
	for i, id := range issueIDs {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args[i] = id
	}
	query += ") GROUP BY i.id"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql message store: count messages by ids: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("mysql message store: count messages by ids: %w", cerr)
		}
	}()

	var counts []warnly.MessageCount
	for rows.Next() {
		var count warnly.MessageCount
		if err := rows.Scan(&count.IssueID, &count.MessageCount); err != nil {
			return nil, fmt.Errorf("mysql message store: count messages by ids scan: %w", err)
		}
		counts = append(counts, count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql message store: count messages by ids rows: %w", err)
	}

	return counts, nil
}
