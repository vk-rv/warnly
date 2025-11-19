package mock

import (
	"context"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// TeamStore is a mock implementation of warnly.TeamStore.
type TeamStore struct {
	ListTeamsFn     func(ctx context.Context, userID int) ([]warnly.Team, error)
	ListTeammatesFn func(ctx context.Context, teamIDs []int) ([]warnly.Teammate, error)
	CreateTeamFn    func(ctx context.Context, team warnly.Team) error
	AddUserToTeamFn func(ctx context.Context, createdAt time.Time, userID int64, teamID int) error
}

func (m *TeamStore) ListTeams(ctx context.Context, userID int) ([]warnly.Team, error) {
	return m.ListTeamsFn(ctx, userID)
}

func (m *TeamStore) ListTeammates(ctx context.Context, teamIDs []int) ([]warnly.Teammate, error) {
	return m.ListTeammatesFn(ctx, teamIDs)
}

func (m *TeamStore) CreateTeam(ctx context.Context, team warnly.Team) error {
	return m.CreateTeamFn(ctx, team)
}

func (m *TeamStore) AddUserToTeam(ctx context.Context, createdAt time.Time, userID int64, teamID int) error {
	return m.AddUserToTeamFn(ctx, createdAt, userID, teamID)
}
