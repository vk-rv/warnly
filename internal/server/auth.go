package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
)

// contextKey is a type for context keys defined in this package.
type contextKey string

// userContextKey is the key for user values in Contexts. It is used to retrieve the user from the context.
const userContextKey contextKey = "user"

// authMw is a middleware for authentication.
type authMw struct {
	cookieStore *session.CookieStore
	logger      *slog.Logger
}

// newAuthMW is a constructor of authMw.
func newAuthMW(cookieStore *session.CookieStore, logger *slog.Logger) *authMw {
	return &authMw{
		cookieStore: cookieStore,
		logger:      logger,
	}
}

// authenticate middleware: adds user to context if found in session cookie,
// otherwise redirects to login page.
func (mw *authMw) authenticate(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := mw.getUser(r)
		if err != nil {
			mw.logger.Error("authenticate: get user, redirecting to login",
				slog.Any("error", err),
				slog.String("method", r.Method),
				slog.String("url", r.URL.String()))
			if r.Header.Get(htmxHeader) != "" {
				w.Header().Add("Hx-Redirect", "/login")
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}
		handler.ServeHTTP(w, r.WithContext(NewContextWithUser(r.Context(), user)))
	}
}

// NewContextWithUser is a helper function to add user to context.
func NewContextWithUser(ctx context.Context, user warnly.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// getUser retrieves user from the session cookie.
func (mw *authMw) getUser(r *http.Request) (warnly.User, error) {
	sess, err := mw.cookieStore.Get(r, "session")
	if err != nil {
		return warnly.User{}, fmt.Errorf("get session: %w", err)
	}
	if sess.Values.User.ID == 0 {
		return warnly.User{}, errors.New("session user is nil")
	}
	return sess.Values.User, nil
}

// getUser retrieves user from context which was set by authentication middleware.
// and we want panic if it is not set.
//
//nolint:forcetypeassert // This is safe because we ensure that the user is set in the context
func getUser(ctx context.Context) warnly.User { return ctx.Value(userContextKey).(warnly.User) }
