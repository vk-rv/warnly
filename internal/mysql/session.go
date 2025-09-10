package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vk-rv/warnly/internal/warnly"
)

// SessionStore encapsulates user session operations.
type SessionStore struct {
	db ExtendedDB
}

// NewSessionStore is a constructor of SessionStore.
func NewSessionStore(db ExtendedDB) *SessionStore {
	return &SessionStore{db: db}
}

// GetHashedPassword returns a hashed password by email.
// Returns warnly.ErrNotFound if user with the given email does not exist.
func (s *SessionStore) GetHashedPassword(ctx context.Context, email string) ([]byte, error) {
	const query = `SELECT password FROM user WHERE email = ?`
	var hashedPassword []byte
	err := s.db.QueryRowContext(ctx, query, email).Scan(&hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", warnly.ErrNotFound, email)
		}
		return nil, fmt.Errorf("mysql session store: get hashed password by email: %w", err)
	}
	return hashedPassword, nil
}
