// Package session provides the implementation of the SessionService interface,
// which includes methods for user authentication and session management.
package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/vk-rv/warnly/internal/warnly"
	"golang.org/x/crypto/bcrypt"
)

// SessionService provides user session operations.
type SessionService struct {
	sessionStore warnly.SessionStore
	userStore    warnly.UserStore
}

// NewSessionService is a constructor of SessionService.
func NewSessionService(sessionStore warnly.SessionStore, userStore warnly.UserStore) *SessionService {
	return &SessionService{sessionStore: sessionStore, userStore: userStore}
}

// SignIn authenticates a user by email and password.
func (s *SessionService) SignIn(ctx context.Context, creds *warnly.Credentials) (*warnly.Session, error) {
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

	user, err := s.userStore.GetUser(ctx, creds.Email)
	if err != nil {
		if errors.Is(err, warnly.ErrNotFound) {
			return nil, warnly.ErrInvalidLoginCredentials
		}
		return nil, err
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

	if err := s.userStore.CreateUser(ctx, email, hashedPassword); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}
