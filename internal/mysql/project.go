package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/vk-rv/warnly/internal/warnly"
)

// ProjectStore implements warnly.ProjectStore for MySQL.
type ProjectStore struct {
	db ExtendedDB
}

// NewProjectStore is a constructor of ProjectStore.
func NewProjectStore(db ExtendedDB) *ProjectStore {
	return &ProjectStore{db: db}
}

// CreateProject is a method that creates a new project.
func (s *ProjectStore) CreateProject(ctx context.Context, p *warnly.Project) error {
	const query = `INSERT INTO project (created_at, name, user_id, team_id, platform, project_key) VALUES (?, ?, ?, ?, ?, ?)`

	res, err := s.db.ExecContext(ctx, query, p.CreatedAt, p.Name, p.UserID, p.TeamID, p.Platform, p.Key)
	if err != nil {
		return fmt.Errorf("mysql project store: create project: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql project store: create project: %w", err)
	}
	if affected != 1 {
		return errors.New("mysql project store: create project: no rows affected")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql project store: create project: %w", err)
	}
	p.ID = int(id)

	return nil
}

// DeleteProject deletes a project by project unique identifier.
func (s *ProjectStore) DeleteProject(ctx context.Context, projectID int) error {
	const query = `DELETE FROM project WHERE id = ?`

	res, err := s.db.ExecContext(ctx, query, projectID)
	if err != nil {
		return fmt.Errorf("mysql project store: delete project: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql project store: delete project: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("mysql project store: delete project: bad rows affected : %d", affected)
	}

	return nil
}

// GetProject returns a project by unique identifier.
// Returns warnly.ErrProjectNotFound if project does not exist.
func (s *ProjectStore) GetProject(ctx context.Context, projectID int) (*warnly.Project, error) {
	const query = `SELECT id, created_at, name, user_id, team_id, platform, project_key FROM project WHERE id = ?`

	p := &warnly.Project{}
	err := s.db.QueryRowContext(ctx, query, projectID).Scan(&p.ID, &p.CreatedAt, &p.Name, &p.UserID, &p.TeamID, &p.Platform, &p.Key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("mysql project store: get project with id %d: %w", projectID, warnly.ErrProjectNotFound)
		}
		return nil, fmt.Errorf("mysql project store: get project: %w", err)
	}

	return p, nil
}

// GetOptions returns project options by project ID.
func (s *ProjectStore) GetOptions(ctx context.Context, projectID int) (*warnly.ProjectOptions, error) {
	return &warnly.ProjectOptions{}, nil
}

// ListProjects returns a list of projects by team unique identifiers.
func (s *ProjectStore) ListProjects(
	ctx context.Context,
	teamIDs []int,
	name string,
) ([]warnly.Project, error) {
	query, args := buildListProjectsQuery(teamIDs, name)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql project store: list projects: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			err = cerr
		}
	}()

	return scan(rows, scanProject)
}

// scanProject scans a single project from sql.Rows.
func scanProject(rows *sql.Rows) (warnly.Project, error) {
	var p warnly.Project
	err := rows.Scan(&p.ID, &p.Name, &p.Platform)
	if err != nil {
		return warnly.Project{}, err
	}
	return p, nil
}

// scanTeammates scans a single teammate from sql.Rows.
func scanTeammates(rows *sql.Rows) (warnly.Teammate, error) {
	var t warnly.Teammate
	err := rows.Scan(&t.ID, &t.Name, &t.Surname, &t.Email, &t.Username)
	if err != nil {
		return warnly.Teammate{}, err
	}
	return t, nil
}

// scan is a generic function that scans sql.Rows using the provided scanFunc.
func scan[T any](rows *sql.Rows, scanFunc func(*sql.Rows) (T, error)) ([]T, error) {
	var items []T
	for rows.Next() {
		item, err := scanFunc(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql store: scan items: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql store: scan items: %w", err)
	}

	return items, nil
}

// buildListProjectsQuery builds the SQL query and arguments for listing projects.
func buildListProjectsQuery(teamIDs []int, name string) (string, []any) {
	query := `SELECT id, name, platform FROM project WHERE team_id IN (` + strings.Repeat("?,", len(teamIDs)-1) + `?)`
	if name != "" {
		query += ` AND name LIKE ?`
	}

	args := make([]any, 0, len(teamIDs)+1)
	for _, id := range teamIDs {
		args = append(args, id)
	}
	if name != "" {
		args = append(args, "%"+name+"%")
	}

	return query, args
}
