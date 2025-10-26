// Package session provides the implementation of the SessionService interface,
// which includes methods for user authentication and session management.
package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vk-rv/warnly/internal/uow"
	"github.com/vk-rv/warnly/internal/warnly"
	"golang.org/x/crypto/bcrypt"
)

// SessionService provides user session operations.
type SessionService struct {
	sessionStore warnly.SessionStore
	userStore    warnly.UserStore
	teamStore    warnly.TeamStore
	uow          uow.StartUnitOfWork
	now          func() time.Time
}

// NewSessionService is a constructor of SessionService.
func NewSessionService(
	sessionStore warnly.SessionStore,
	userStore warnly.UserStore,
	teamStore warnly.TeamStore,
	uw uow.StartUnitOfWork,
	now func() time.Time,
) *SessionService {
	return &SessionService{
		sessionStore: sessionStore,
		userStore:    userStore,
		teamStore:    teamStore,
		uow:          uw,
		now:          now,
	}
}

// SignIn authenticates a user by email and password.
func (s *SessionService) SignIn(ctx context.Context, creds *warnly.Credentials) (*warnly.Session, error) {
	user, err := s.userStore.GetUser(ctx, creds.Email)
	if err != nil {
		if errors.Is(err, warnly.ErrNotFound) {
			return nil, warnly.ErrInvalidLoginCredentials
		}
		return nil, err
	}

	if user.AuthMethod == warnly.AuthMethodOIDC {
		return nil, warnly.ErrInvalidAuthMethod
	}

	hashedPassword, err := s.sessionStore.GetHashedPassword(ctx, creds.Email)
	if err != nil {
		if errors.Is(err, warnly.ErrNotFound) {
			return nil, warnly.ErrInvalidLoginCredentials
		}
		return nil, err
	}

	if err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(creds.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, warnly.ErrInvalidLoginCredentials
		}
		return nil, fmt.Errorf("bcrypt compare hash and password: %w", err)
	}

	return &warnly.Session{User: user}, nil
}

// CreateUserIfNotExists creates a user if it does not exist in the database.
func (s *SessionService) CreateUserIfNotExists(ctx context.Context, email, password string) error {
	user, err := s.userStore.GetUser(ctx, email)
	if err != nil {
		if !errors.Is(err, warnly.ErrNotFound) {
			return fmt.Errorf("get user: %w", err)
		}
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

	if err := s.userStore.CreateUser(ctx, email, username, hashedPassword); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// GetOrCreateUser creates a new user if it does not exist in the database.
// If the user exists, it returns the existing user.
func (s *SessionService) GetOrCreateUser(ctx context.Context, req *warnly.GetOrCreateUserRequest) (*warnly.Session, error) {
	user, err := s.userStore.GetUser(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, warnly.ErrNotFound) {
			return nil, err
		}
	}
	if user != nil {
		return &warnly.Session{User: user}, nil
	}

	if req.Username == "" {
		req.Username, err = warnly.UsernameFromEmail(req.Email)
		if err != nil {
			return nil, err
		}
	}

	err = s.uow(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		userID, err := uw.Users().CreateUserOIDC(ctx, req)
		if err != nil {
			return err
		}
		return uw.Teams().AddUserToTeam(ctx, s.now().UTC(), userID, warnly.DefaultTeamID)
	}, s.userStore, s.teamStore)
	if err != nil {
		return nil, err
	}

	user, err = s.userStore.GetUser(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	return &warnly.Session{User: user}, nil
}
