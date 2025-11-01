package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AlertStore implements warnly.AlertStore interface.
type AlertStore struct {
	db ExtendedDB
}

// NewAlertStore is a constructor of AlertStore.
func NewAlertStore(db ExtendedDB) *AlertStore {
	return &AlertStore{db: db}
}

// ListAlerts returns a list of alerts for the given criteria.
func (s *AlertStore) ListAlerts(
	ctx context.Context,
	teamIDs []int,
	projectName string,
	offset,
	limit int,
) ([]warnly.Alert, int, error) {
	var (
		alerts     []warnly.Alert
		totalCount int
	)

	var (
		conditions []string
		args       []any
	)

	if len(teamIDs) > 0 {
		placeholders := make([]string, len(teamIDs))
		for i, teamID := range teamIDs {
			placeholders[i] = "?"
			args = append(args, teamID)
		}
		conditions = append(conditions, fmt.Sprintf("a.team_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if projectName != "" {
		conditions = append(conditions, "p.name = ?")
		args = append(args, projectName)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM alert a
		JOIN project p ON a.project_id = p.id
		%s
	`, whereClause)

	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("mysql: count alerts: %w", err)
	}

	paginationArgs := make([]any, len(args))
	copy(paginationArgs, args)
	paginationArgs = append(paginationArgs, limit, offset) //nolint:makezero // dont care about it

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.created_at, a.updated_at, a.last_triggered_at,
			a.rule_name, a.description, a.status,
			a.project_id, a.team_id, a.threshold, a.cond, a.timeframe, a.is_high_priority
		FROM alert a
		JOIN project p ON a.project_id = p.id
		%s
		ORDER BY a.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, paginationArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query alerts: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	for rows.Next() {
		var (
			alert           warnly.Alert
			lastTriggeredAt sql.NullTime
			description     sql.NullString
		)
		err := rows.Scan(
			&alert.ID,
			&alert.CreatedAt,
			&alert.UpdatedAt,
			&lastTriggeredAt,
			&alert.RuleName,
			&description,
			&alert.Status,
			&alert.ProjectID,
			&alert.TeamID,
			&alert.Threshold,
			&alert.Condition,
			&alert.Timeframe,
			&alert.HighPriority,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("mysql: scan alert: %w", err)
		}

		if lastTriggeredAt.Valid {
			alert.LastTriggeredAt = &lastTriggeredAt.Time
		}
		if description.Valid {
			alert.Description = description.String
		}

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("mysql: iterate alerts, rows.err: %w", err)
	}

	return alerts, totalCount, nil
}

// CreateAlert creates a new alert.
func (s *AlertStore) CreateAlert(ctx context.Context, alert *warnly.Alert) error {
	const query = `
		INSERT INTO alert (
			created_at, updated_at, rule_name, description, status,
			project_id, team_id, threshold, cond, timeframe, is_high_priority) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		alert.CreatedAt,
		alert.UpdatedAt,
		alert.RuleName,
		alert.Description,
		alert.Status,
		alert.ProjectID,
		alert.TeamID,
		alert.Threshold,
		alert.Condition,
		alert.Timeframe,
		alert.HighPriority,
	)
	if err != nil {
		return fmt.Errorf("mysql: insert alert: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql: get last insert id: %w", err)
	}

	alert.ID = int(id)

	return nil
}

// UpdateAlert updates an existing alert.
func (s *AlertStore) UpdateAlert(ctx context.Context, alert *warnly.Alert) error {
	const query = `
		UPDATE alert
		SET updated_at = ?, rule_name = ?, description = ?, status = ?,
			threshold = ?, cond = ?, timeframe = ?, is_high_priority = ?,
			last_triggered_at = ?
		WHERE id = ?
	`

	res, err := s.db.ExecContext(
		ctx,
		query,
		alert.UpdatedAt,
		alert.RuleName,
		alert.Description,
		alert.Status,
		alert.Threshold,
		alert.Condition,
		alert.Timeframe,
		alert.HighPriority,
		alert.LastTriggeredAt,
		alert.ID,
	)
	if err != nil {
		return fmt.Errorf("mysql: update alert: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql: get rows affected: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("mysql: rows affected not equal to 1: %d", rowsAffected)
	}

	return nil
}

// DeleteAlert deletes an alert by ID.
func (s *AlertStore) DeleteAlert(ctx context.Context, alertID int) error {
	const query = `DELETE FROM alert WHERE id = ?`

	res, err := s.db.ExecContext(ctx, query, alertID)
	if err != nil {
		return fmt.Errorf("mysql: delete alert: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql: get rows affected: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("mysql: rows affected not equal to 1: %d", rowsAffected)
	}

	return nil
}

// GetAlert returns an alert by ID.
func (s *AlertStore) GetAlert(ctx context.Context, alertID int) (*warnly.Alert, error) {
	const query = `
		SELECT 
			id, created_at, updated_at, last_triggered_at,
			rule_name, description, status,
			project_id, team_id, threshold, cond, timeframe, is_high_priority
		FROM alert
		WHERE id = ?
	`

	var (
		alert           warnly.Alert
		lastTriggeredAt sql.NullTime
		description     sql.NullString
	)

	err := s.db.QueryRowContext(ctx, query, alertID).Scan(
		&alert.ID,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&lastTriggeredAt,
		&alert.RuleName,
		&description,
		&alert.Status,
		&alert.ProjectID,
		&alert.TeamID,
		&alert.Threshold,
		&alert.Condition,
		&alert.Timeframe,
		&alert.HighPriority,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("mysql: alert id %d not found: %w", alertID, warnly.ErrNotFound)
		}
		return nil, fmt.Errorf("mysql: get alert: %w", err)
	}

	if lastTriggeredAt.Valid {
		alert.LastTriggeredAt = &lastTriggeredAt.Time
	}
	if description.Valid {
		alert.Description = description.String
	}

	return &alert, nil
}

// ListAlertsByProject returns alerts for a project.
func (s *AlertStore) ListAlertsByProject(ctx context.Context, projectID int) ([]warnly.Alert, error) {
	const query = `
		SELECT 
			id, created_at, updated_at, last_triggered_at,
			rule_name, description, status,
			project_id, team_id, threshold, cond, timeframe, is_high_priority
		FROM alert
		WHERE project_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("mysql: list alerts by project: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var alerts []warnly.Alert
	for rows.Next() {
		var (
			alert           warnly.Alert
			lastTriggeredAt sql.NullTime
			description     sql.NullString
		)
		err := rows.Scan(
			&alert.ID,
			&alert.CreatedAt,
			&alert.UpdatedAt,
			&lastTriggeredAt,
			&alert.RuleName,
			&description,
			&alert.Status,
			&alert.ProjectID,
			&alert.TeamID,
			&alert.Threshold,
			&alert.Condition,
			&alert.Timeframe,
			&alert.HighPriority,
		)
		if err != nil {
			return nil, fmt.Errorf("mysql: scan alert: %w", err)
		}

		if lastTriggeredAt.Valid {
			alert.LastTriggeredAt = &lastTriggeredAt.Time
		}
		if description.Valid {
			alert.Description = description.String
		}

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: iterate alerts: %w", err)
	}

	return alerts, nil
}
