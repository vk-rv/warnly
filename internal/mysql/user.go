package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/vk-rv/warnly/internal/warnly"
)

// UserStore provides user operations.
type UserStore struct {
	db ExtendedDB
}

// NewUserStore is a constructor of UserStore.
func NewUserStore(db ExtendedDB) *UserStore {
	return &UserStore{db: db}
}

// GetUser returns a user by email.
// Returns warnly.ErrNotFound if user with the given email does not exist.
func (s *UserStore) GetUser(ctx context.Context, email string) (*warnly.User, error) {
	const query = `SELECT id, email, name, surname, username FROM user WHERE email = ?`
	user := warnly.User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Name, &user.Surname, &user.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", warnly.ErrNotFound, email)
		}
		return nil, fmt.Errorf("mysql user store: get user by email: %w", err)
	}
	return &user, nil
}

// CreateUser creates a user in the database.
func (s *UserStore) CreateUser(ctx context.Context, email string, hashedPassword []byte) error {
	const query = `INSERT INTO user (name, surname, email, password, username) VALUES (?, ?, ?, ?, ?)`
	atIndex := strings.Index(email, "@")
	if atIndex == -1 {
		return fmt.Errorf("invalid email format: %s", email)
	}
	username := email[:atIndex]
	_, err := s.db.ExecContext(ctx, query, "John", "Doe", email, hashedPassword, username)
	if err != nil {
		return fmt.Errorf("mysql user store: create user: %w", err)
	}
	return nil
}
