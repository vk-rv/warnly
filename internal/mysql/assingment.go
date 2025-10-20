package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/vk-rv/warnly/internal/warnly"
	"golang.org/x/exp/constraints"
)

// AssingmentStore provides issue assignment operations
// (when an issue is assigned to a user).
type AssingmentStore struct {
	db ExtendedDB
}

// NewAssingmentStore is a constructor of issue assignment database repository.
func NewAssingmentStore(db ExtendedDB) *AssingmentStore {
	return &AssingmentStore{db: db}
}

// CreateAssingment creates a new issue assignment in the database (assigns an issue to a user).
func (s *AssingmentStore) CreateAssingment(ctx context.Context, a *warnly.Assignment) error {
	const query = `INSERT INTO issue_assignment (issue_id, assigned_to_user_id, assigned_by_user_id, assigned_at)
				   VALUES (?, ?, ?, ?)
				   ON DUPLICATE KEY UPDATE
				   assigned_to_user_id = VALUES(assigned_to_user_id),
				   assigned_to_team_id = NULL,
				   assigned_by_user_id = VALUES(assigned_by_user_id),
				   assigned_at = VALUES(assigned_at);`

	_, err := s.db.ExecContext(
		ctx,
		query,
		a.IssueID,
		a.AssignedToUserID,
		a.AssignedByUserID,
		a.AssignedAt,
	)
	if err != nil {
		return fmt.Errorf("mysql issue assignment store: create assignment: %w", err)
	}

	return nil
}

// DeleteAssignment unassigns an issue from a user.
func (s *AssingmentStore) DeleteAssignment(ctx context.Context, issueID int64) error {
	const query = `DELETE FROM issue_assignment WHERE issue_id = ?`
	_, err := s.db.ExecContext(ctx, query, issueID)
	if err != nil {
		return fmt.Errorf("mysql issue assignment store: unassign issue: %w", err)
	}
	return nil
}

// ListAssingments lists all assignments for a given issue.
func (s *AssingmentStore) ListAssingments(ctx context.Context, issueIDs []int64) ([]*warnly.AssignedUser, error) {
	placeholders, args := makePlaceholders(issueIDs)

	query := fmt.Sprintf(`
		SELECT issue_id, assigned_to_user_id
		FROM issue_assignment
		WHERE issue_id IN (%s)
	`, placeholders)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql issue assignment store: get assigned users: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("mysql issue assignment store: get assigned users, close rows: %w", cerr)
		}
	}()

	var au []*warnly.AssignedUser
	for rows.Next() {
		var a warnly.AssignedUser
		if err := rows.Scan(&a.IssueID, &a.AssignedToUserID); err != nil {
			return nil, fmt.Errorf("mysql issue assignment store: get assigned users, scan: %w", err)
		}
		au = append(au, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql issue assignment store: get assigned users, rows error: %w", err)
	}

	return au, nil
}

func (s *AssingmentStore) ListAssignedFilters(
	ctx context.Context,
	criteria *warnly.GetAssignedFiltersCriteria,
) ([]warnly.Filter, error) {
	placeholders := strings.Repeat("?,", len(criteria.CurrentUserTeamIDs))
	placeholders = strings.TrimSuffix(placeholders, ",")

	query := fmt.Sprintf(`
        SELECT u.username AS assigned_value
        FROM user AS u
        INNER JOIN team_relation AS tr ON u.id = tr.user_id
        WHERE tr.team_id IN (%s)
        GROUP BY u.username
        UNION
        SELECT 'unassigned' AS assigned_value
        UNION ALL
        SELECT 'me' AS assigned_value
        ORDER BY assigned_value;
    `, placeholders)

	args := make([]any, len(criteria.CurrentUserTeamIDs))
	for i := range criteria.CurrentUserTeamIDs {
		args[i] = criteria.CurrentUserTeamIDs[i]
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql issue store: get assigned filters: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			err = cerr
		}
	}()

	var filterItems []warnly.Filter
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("mysql issue store: scan assigned filter: %w", err)
		}

		filterItems = append(filterItems, warnly.Filter{
			Key:   "assigned",
			Value: value,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql issue store: rows error: %w", err)
	}

	return filterItems, nil
}

// makePlaceholders creates a string of placeholders for SQL IN clause and a slice of arguments.
func makePlaceholders[T constraints.Integer](ids []T) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	return strings.Join(placeholders, ","), args
}
