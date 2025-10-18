package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

const defaultPeriod = "14d"

const (
	// msgInvalidLoginCredentials is the message displayed on the login page when the user provides invalid credentials.
	msgInvalidLoginCredentials = "Invalid login credentials."
	// msgSomethingWentWrong is a generic error message displayed to the user when an unexpected error occurs.
	msgSomethingWentWrong = "Something went wrong. Check application logs for more details."
)

// sessionHandler handles HTTP requests related to user sessions.
type sessionHandler struct {
	*BaseHandler

	svc          warnly.SessionService
	projectSvc   warnly.ProjectService
	cookieStore  *session.CookieStore
	logger       *slog.Logger
	rememberDays int
}

// newSessionHandler creates a new sessionHandler instance.
func newSessionHandler(
	sessionSvc warnly.SessionService,
	projectSvc warnly.ProjectService,
	cookieStore *session.CookieStore,
	rememberDays int,
	logger *slog.Logger,
) *sessionHandler {
	return &sessionHandler{
		BaseHandler:  NewBaseHandler(logger),
		svc:          sessionSvc,
		projectSvc:   projectSvc,
		rememberDays: rememberDays,
		cookieStore:  cookieStore,
		logger:       logger,
	}
}

// index handles the HTTP request to render the main page with a list of issues.
func (h *sessionHandler) index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	period := r.URL.Query().Get("period")
	if period == "" {
		period = defaultPeriod
	}

	offset, err := parseOffset(r.URL.Query().Get("offset"))
	if err != nil {
		h.writeError(ctx, w, http.StatusBadRequest, "list issues: parse offset", err)
		return
	}

	req := &warnly.ListIssuesRequest{
		User:        &user,
		Period:      period,
		Start:       r.URL.Query().Get("start"),
		End:         r.URL.Query().Get("end"),
		Query:       r.URL.Query().Get("query"),
		Filters:     r.URL.Query().Get("filters"),
		ProjectName: r.URL.Query().Get("project_name"),
		Offset:      offset,
		Limit:       50,
	}

	result, err := h.projectSvc.ListIssues(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "index: list issues", err)
		return
	}

	h.writeIndex(w, r, result, &user)
}

// writeIndex writes the index page to the response writer.
func (h *sessionHandler) writeIndex(w http.ResponseWriter, r *http.Request, res *warnly.ListIssuesResult, user *warnly.User) {
	ctx := r.Context()

	target := r.Header.Get("Hx-Target")
	partial := r.URL.Query().Get("partial")

	switch {
	case partial == "body":
		if err := web.IssuesBody(res).Render(ctx, w); err != nil {
			h.logger.Error("print index body web render", slog.Any("error", err))
		}
	case partial == "filters" || (target == "issues-container" && partial != ""):
		if err := web.IssuesFiltersAndBody(res).Render(ctx, w); err != nil {
			h.logger.Error("print index filters web render", slog.Any("error", err))
		}
	case target == "content":
		if err := web.IssuesHtmx(res).Render(ctx, w); err != nil {
			h.logger.Error("print index htmx web render", slog.Any("error", err))
		}
	default:
		if err := web.Index(res, user).Render(ctx, w); err != nil {
			h.logger.Error("print index web render", slog.Any("error", err))
		}
	}
}

// destroy handles the HTTP request to destroy the current session (log out).
func (h *sessionHandler) destroy(w http.ResponseWriter, r *http.Request) {
	if err := destroySession(w, r, h.cookieStore); err != nil {
		h.logger.Error("destroy session: destroy", slog.Any("error", err))
		if err = web.Hello("").Render(r.Context(), w); err != nil {
			h.logger.Error("destroy session: hello web render", slog.Any("error", err))
		}
		return
	}

	w.Header().Add("Hx-Redirect", "/")
}

// destroySession removes the session cookie.
func destroySession(w http.ResponseWriter, r *http.Request, cookieStore *session.CookieStore) error {
	sess, err := cookieStore.Get(r, "sessid")
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	sess.Options.MaxAge = -1
	if err := cookieStore.Save(r, w, sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// login handles the HTTP request to render the login page.
func (h *sessionHandler) login(w http.ResponseWriter, r *http.Request) {
	if err := web.Hello("").Render(r.Context(), w); err != nil {
		h.logger.Error("get session: hello web render", slog.Any("error", err))
	}
}

// create handles the HTTP request to create a new session.
// It authenticates the user and sets the session cookie.
// If the authentication fails, it renders an error page.
// If the authentication succeeds, it redirects to the main page.
func (h *sessionHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	credentials := &warnly.Credentials{
		Email:      r.PostFormValue("email"),
		Password:   r.PostFormValue("password"),
		RememberMe: r.PostFormValue("remember-me") == "on",
	}

	result, err := h.svc.SignIn(ctx, credentials)
	if err != nil {
		switch {
		case errors.Is(err, warnly.ErrInvalidLoginCredentials):
			h.logger.Error("create new session: invalid login credentials",
				slog.Any("error", err),
				slog.String("email", credentials.Email))
			if err = web.Hello(msgInvalidLoginCredentials).Render(ctx, w); err != nil {
				h.logger.Error("create new session: hello web render", slog.Any("error", err))
			}
			return
		default:
			h.logger.Error("create new session: sign in", slog.Any("error", err))
			if err = web.Hello(msgSomethingWentWrong).Render(ctx, w); err != nil {
				h.logger.Error("create new session: hello web render", slog.Any("error", err))
			}
			return
		}
	}

	if err := saveCookie(
		w,
		r,
		h.cookieStore,
		*result.User,
		credentials.RememberMe,
		h.rememberDays); err != nil {
		h.logger.Error("create new session: save cookie", slog.Any("error", err))
		if err = web.Hello(msgSomethingWentWrong).Render(ctx, w); err != nil {
			h.logger.Error("create new session: hello web render", slog.Any("error", err))
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// saveCookie saves the user information in a session cookie.
func saveCookie(
	w http.ResponseWriter,
	r *http.Request,
	cookieStore *session.CookieStore,
	user warnly.User,
	rememberMe bool,
	sessionDays int,
) error {
	sess, err := cookieStore.Get(r, "sessid")
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if rememberMe {
		sess.Options.MaxAge = 86400 * sessionDays
	} else {
		sess.Options.MaxAge = 0
	}

	sess.Values.User = user
	if err := cookieStore.Save(r, w, sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
}
