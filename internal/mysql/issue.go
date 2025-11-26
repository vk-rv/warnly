package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/vk-rv/warnly/internal/warnly"
)

// IssueStore encapsulates issue-related database operations.
type IssueStore struct {
	db ExtendedDB
}

// NewIssueStore is a constructor of issue database repository.
func NewIssueStore(db ExtendedDB) *IssueStore {
	return &IssueStore{db: db}
}

// GetIssue returns an issue by project identifier and hash obtained from event stacktrace or message.
func (s *IssueStore) GetIssue(ctx context.Context, criteria warnly.GetIssueCriteria) (*warnly.Issue, error) {
	const query = `SELECT id, uuid, first_seen, last_seen, hash, message, view, 
				   num_comments, project_id, priority FROM issue WHERE project_id = ? AND hash = ?`

	i := warnly.Issue{}
	err := s.
		db.
		QueryRowContext(ctx, query, criteria.ProjectID, criteria.Hash).
		Scan(&i.ID,
			&i.UUID,
			&i.FirstSeen,
			&i.LastSeen,
			&i.Hash,
			&i.Message,
			&i.View,
			&i.NumComments,
			&i.ProjectID,
			&i.Priority)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, warnly.ErrNotFound
		}
		return nil, fmt.Errorf("mysql issue store: get issue: %w", err)
	}

	return &i, nil
}

// GetIssueByID returns an issue by its unique database identifier.
func (s *IssueStore) GetIssueByID(ctx context.Context, issueID int64) (*warnly.Issue, error) {
	const query = `SELECT id, uuid, first_seen, last_seen, hash, message, view, 
				   num_comments, project_id, priority, error_type 
				   FROM issue WHERE id = ?`

	i := &warnly.Issue{}
	err := s.
		db.
		QueryRowContext(ctx, query, issueID).
		Scan(&i.ID,
			&i.UUID,
			&i.FirstSeen,
			&i.LastSeen,
			&i.Hash,
			&i.Message,
			&i.View,
			&i.NumComments,
			&i.ProjectID,
			&i.Priority,
			&i.ErrorType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, warnly.ErrNotFound
		}
		return nil, fmt.Errorf("mysql issue store: get issue: %w", err)
	}

	return i, nil
}

// ListIssues returns a list of issues for given project IDs and time range.
func (s *IssueStore) ListIssues(ctx context.Context, criteria *warnly.ListIssuesCriteria) ([]warnly.Issue, error) {
	query := `SELECT id, uuid, first_seen, last_seen, hash, message, view, num_comments,
project_id, priority, error_type
FROM issue WHERE project_id IN (?` + strings.Repeat(",?", len(criteria.ProjectIDs)-1) + `)
AND ((last_seen BETWEEN ? AND ?) OR (first_seen BETWEEN ? AND ?))`

	if len(criteria.GroupIDs) > 0 {
		query += ` AND id IN (?` + strings.Repeat(",?", len(criteria.GroupIDs)-1) + `)`
	}

	args := make([]any, len(criteria.ProjectIDs)+4)
	for i, id := range criteria.ProjectIDs {
		args[i] = id
	}
	args[len(criteria.ProjectIDs)] = criteria.From
	args[len(criteria.ProjectIDs)+1] = criteria.To
	args[len(criteria.ProjectIDs)+2] = criteria.From
	args[len(criteria.ProjectIDs)+3] = criteria.To

	if len(criteria.GroupIDs) > 0 {
		for _, gid := range criteria.GroupIDs {
			//nolint:makezero // false positive
			args = append(args, gid)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql issue store: list issues: %w", err)
	}

	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var issues []warnly.Issue
	for rows.Next() {
		i := warnly.Issue{}
		err = rows.Scan(
			&i.ID,
			&i.UUID,
			&i.FirstSeen,
			&i.LastSeen,
			&i.Hash,
			&i.Message,
			&i.View,
			&i.NumComments,
			&i.ProjectID,
			&i.Priority,
			&i.ErrorType)
		if err != nil {
			return nil, fmt.Errorf("mysql issue store: list issues: %w", err)
		}
		issues = append(issues, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql issue store: list issues: %w", err)
	}

	return issues, nil
}

// StoreIssue stores a new issue in the database.
func (s *IssueStore) StoreIssue(ctx context.Context, i *warnly.Issue) error {
	const query = `INSERT INTO issue (uuid, first_seen, last_seen, hash, message, view, 
					num_comments, project_id, priority, error_type) 
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := s.db.ExecContext(
		ctx,
		query,
		i.UUID,
		i.FirstSeen,
		i.LastSeen,
		i.Hash,
		i.Message,
		i.View,
		i.NumComments,
		i.ProjectID,
		i.Priority,
		i.ErrorType)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateKey {
			return warnly.ErrDuplicate
		}
		return fmt.Errorf("mysql issue store: store issue: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql issue store: rows affected: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("mysql issue store: bad rows affected : %d", affected)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql issue store: last insert id %w", err)
	}

	i.ID = id

	return nil
}

// UpdateLastSeen updates the last seen time of an issue.
func (s *IssueStore) UpdateLastSeen(ctx context.Context, upd *warnly.UpdateLastSeen) error {
	const query = `UPDATE issue SET last_seen = ?, message = ?, error_type = ?, view = ? WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, upd.LastSeen, upd.Message, upd.ErrorType, upd.View, upd.IssueID)
	if err != nil {
		return fmt.Errorf("mysql issue store: update last seen: %w", err)
	}

	return nil
}
