// Package warnly provides the core types and interfaces for the Warnly application.
// It is a so-called root package, other layers speak using this package, not each other.
package warnly

import (
	"context"
	"errors"
)

// ErrInvalidLoginCredentials is returned when the provided login credentials are invalid.
var ErrInvalidLoginCredentials = errors.New("invalid login credentials")

// ErrNotFound is returned when an entity is not found in the database.
// It overrides sql.ErrNoRows to avoid leaking database implementation details.
var ErrNotFound = errors.New("entity was not found in database")

// User represents a user in the system.
type User struct {
	Email    string `cbor:"email"`
	Name     string `cbor:"name"`
	Surname  string `cbor:"surname"`
	Username string `cbor:"username"`
	ID       int64  `cbor:"id"`
}

// AvatarInitials returns the initials of the user for avatar display.
func (u *User) AvatarInitials() string {
	return string(u.Name[0]) + string(u.Surname[0])
}

// FullName returns the full name of the user.
func (u *User) FullName() string {
	return u.Name + " " + u.Surname
}

// UserStore defines methods for user data management.
type UserStore interface {
	// GetUser retrieves a user by email.
	GetUser(ctx context.Context, email string) (*User, error)
	// CreateUser creates a new user with the provided email and hashed password.
	CreateUser(ctx context.Context, email string, hashedPassword []byte) error
}

// Session represents a user session, including the authenticated user.
type Session struct {
	User *User
}

// SessionStore defines methods for session data management.
type SessionStore interface {
	// GetHashedPassword retrieves the hashed password for a given email.
	GetHashedPassword(ctx context.Context, email string) ([]byte, error)
}

// SessionService defines methods for session management.
type SessionService interface {
	// SignIn authenticates a user by email and password.
	SignIn(ctx context.Context, credentials *Credentials) (*Session, error)
}

// Credentials represents user login credentials.
type Credentials struct {
	Email      string
	Password   string
	Token      string
	RememberMe bool
}
