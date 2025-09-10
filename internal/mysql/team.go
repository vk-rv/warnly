package mysql

import (
	"context"
	"fmt"

	"github.com/vk-rv/warnly/internal/warnly"
)

// TeamStore provides team operations.
type TeamStore struct {
	db ExtendedDB
}

// NewTeamStore is a constructor of TeamStore.
func NewTeamStore(db ExtendedDB) *TeamStore {
	return &TeamStore{db: db}
}

// CreateTeam creates a new team.
func (s *TeamStore) CreateTeam(ctx context.Context, t warnly.Team) error {
	const query = `INSERT INTO team (created_at, name, owner_id) VALUES (?, ?, ?)`

	res, err := s.db.ExecContext(ctx, query, t.CreatedAt, t.Name, t.OwnerID)
	if err != nil {
		return fmt.Errorf("mysql team store: create team: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql team store: create team: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("mysql team store: create team: bad rows affected : %d", affected)
	}

	return nil
}

// ListTeammates returns a list of teammates for the given team identifiers.
func (s *TeamStore) ListTeammates(ctx context.Context, teamIDs []int) ([]warnly.Teammate, error) {
	placeholders := ""
	args := make([]any, len(teamIDs))
	for i, id := range teamIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT u.id, u.name, u.surname, u.email, u.username 
		FROM team_relation AS tr JOIN user AS u ON tr.user_id = u.id 
		WHERE tr.team_id IN (%s)`, placeholders)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql team store: list teammates: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return scan(rows, scanTeammates)
}

// ListTeams returns a list of teams by user unique identifier.
func (s *TeamStore) ListTeams(ctx context.Context, userID int) ([]warnly.Team, error) {
	const query = `SELECT t.id, t.created_at, t.name, t.owner_id FROM team_relation AS tr JOIN team AS t WHERE tr.user_id = ?`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("mysql team store: list teams: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var teams []warnly.Team
	for rows.Next() {
		var t warnly.Team
		if err := rows.Scan(&t.ID, &t.CreatedAt, &t.Name, &t.OwnerID); err != nil {
			return nil, fmt.Errorf("mysql team store: list teams scan: %w", err)
		}
		teams = append(teams, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql team store: list teams: %w", err)
	}

	return teams, nil
}
