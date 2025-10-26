package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	const query = `SELECT id, email, name, surname, username, auth_method FROM user WHERE email = ?`
	user := warnly.User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Surname,
		&user.Username,
		&user.AuthMethod)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", warnly.ErrNotFound, email)
		}
		return nil, fmt.Errorf("mysql user store: get user by email: %w", err)
	}
	return &user, nil
}

// GetUserByIdentifier returns a user by identifier (email or username).
// Returns warnly.ErrNotFound if user with the given identifier does not exist.
func (s *UserStore) GetUserByIdentifier(ctx context.Context, identifier warnly.UserIdentifier) (*warnly.User, error) {
	var query string
	if identifier.IsEmail {
		query = `SELECT id, email, name, surname, username, auth_method FROM user WHERE email = ?`
	} else {
		query = `SELECT id, email, name, surname, username, auth_method FROM user WHERE username = ?`
	}
	user := warnly.User{}
	err := s.db.QueryRowContext(ctx, query, identifier.Value).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Surname,
		&user.Username,
		&user.AuthMethod)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", warnly.ErrNotFound, identifier.Value)
		}
		return nil, fmt.Errorf("mysql user store: get user by identifier: %w", err)
	}
	return &user, nil
}

// CreateUser creates a user in the database.
func (s *UserStore) CreateUser(ctx context.Context, email, username string, hashedPassword []byte) error {
	const query = `INSERT INTO user (name, surname, email, password, username) VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query, "John", "Doe", email, hashedPassword, username)
	if err != nil {
		return fmt.Errorf("mysql user store: create user: %w", err)
	}
	return nil
}

func (s *UserStore) CreateUserOIDC(ctx context.Context, r *warnly.GetOrCreateUserRequest) (int64, error) {
	const query = `INSERT INTO user (name, surname, email, username, auth_method) 
				   VALUES (?, ?, ?, ?, ?)`

	res, err := s.db.ExecContext(
		ctx,
		query,
		r.Name,
		r.Surname,
		r.Email,
		r.Username,
		warnly.AuthMethodOIDC)
	if err != nil {
		return 0, fmt.Errorf("mysql user store: create user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("mysql user store: create user: %w", err)
	}

	return id, nil
}
