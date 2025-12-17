// Package initializer initializes warnly for the first time.
package initializer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vk-rv/warnly/internal/uow"
	"github.com/vk-rv/warnly/internal/warnly"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultTeamName is the default team name of the first team created.
	DefaultTeamName = "default"
)

// InitService is a service for initializing (seeding) the database with initial data.
type InitService struct {
	userStore warnly.UserStore
	teamStore warnly.TeamStore
	uow       uow.StartUnitOfWork
	now       func() time.Time
}

// NewInitService creates a new InitService instance.
func NewInitService(
	userStore warnly.UserStore,
	teamStore warnly.TeamStore,
	startUow uow.StartUnitOfWork,
	now func() time.Time,
) *InitService {
	return &InitService{
		userStore: userStore,
		teamStore: teamStore,
		uow:       startUow,
		now:       now,
	}
}

// Init initializes the database with initial data.
// It seeds the database with a default team and a default user.
// If the user already exists, it returns nil without doing anything.
func (s *InitService) Init(ctx context.Context, email, password string) error {
	user, err := s.userStore.GetUser(ctx, email)
	if err != nil && !errors.Is(err, warnly.ErrNotFound) {
		return fmt.Errorf("get user: %w", err)
	}
	if user != nil {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	username, err := warnly.UsernameFromEmail(email)
	if err != nil {
		return err
	}

	now := s.now().UTC()

	err = s.uow(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		if err := uw.Users().CreateUser(ctx, email, username, hashedPassword); err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		user, err := uw.Users().GetUser(ctx, email)
		if err != nil {
			return fmt.Errorf("get created user: %w", err)
		}

		if err := uw.Teams().CreateTeam(ctx, warnly.Team{
			CreatedAt: now,
			Name:      DefaultTeamName,
			OwnerID:   int(user.ID),
		}); err != nil {
			return fmt.Errorf("create team: %w", err)
		}

		if err := uw.Teams().AddUserToTeam(ctx, now, user.ID, warnly.DefaultTeamID); err != nil {
			return fmt.Errorf("add user to team: %w", err)
		}

		return nil
	}, s.userStore, s.teamStore)
	if err != nil {
		return err
	}

	return nil
}
