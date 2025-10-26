package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	capoidc "github.com/hashicorp/cap/oidc"
	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

const (
	oidcStateTimeout = 2 * time.Minute
	openid           = "openid"
	sessionID        = "session"
)

const defaultPeriod = "14d"

const (
	// msgInvalidLoginCredentials is the message displayed on the login page when the user provides invalid credentials.
	msgInvalidLoginCredentials = "Invalid login credentials."
	// msgSomethingWentWrong is a generic error message displayed to the user when an unexpected error occurs.
	msgSomethingWentWrong = "Something went wrong. Check application logs for more details."
)

// rootHandler handles HTTP requests related to user sessions and the main page.
type rootHandler struct {
	*BaseHandler

	svc          warnly.SessionService
	projectSvc   warnly.ProjectService
	cookieStore  *session.CookieStore
	oidc         *OIDC
	logger       *slog.Logger
	rememberDays int
}

// newRootHandler creates a new rootHandler instance.
func newRootHandler(
	sessionSvc warnly.SessionService,
	projectSvc warnly.ProjectService,
	cookieStore *session.CookieStore,
	rememberDays int,
	oidc *OIDC,
	logger *slog.Logger,
) *rootHandler {
	return &rootHandler{
		BaseHandler:  NewBaseHandler(logger),
		svc:          sessionSvc,
		projectSvc:   projectSvc,
		rememberDays: rememberDays,
		cookieStore:  cookieStore,
		oidc:         oidc,
		logger:       logger,
	}
}

type claims struct {
	Verified          *bool   `json:"email_verified"`
	Email             *string `json:"email"`
	GivenName         *string `json:"given_name"`
	FamilyName        *string `json:"family_name"`
	PrefferedUsername *string `json:"preferred_username"`
	Name              *string `json:"name"`
	Profile           *string `json:"profile"`
	Picture           *string `json:"picture"`
	Sub               *string `json:"sub"`
	Nonce             *string `json:"nonce"`
}

// UserData returns user data from claims.
func (c *claims) UserData() *warnly.GetOrCreateUserRequest {
	req := &warnly.GetOrCreateUserRequest{}
	if c.GivenName == nil && c.FamilyName == nil && c.Name != nil {
		parts := strings.Split(*c.Name, " ")
		if len(parts) > 1 {
			req.Name = parts[0]
			req.Surname = parts[1]
		}
	}
	if c.GivenName != nil && c.FamilyName != nil {
		req.Name = *c.GivenName
		req.Surname = *c.FamilyName
	}
	if c.PrefferedUsername != nil {
		req.Username = *c.PrefferedUsername
	}
	if c.Email != nil {
		req.Email = *c.Email
	}
	return req
}

func newOIDCRequest(scopes []string, usePKCE bool, redirect string) (capoidc.Request, warnly.OIDCState, error) {
	state, err := capoidc.NewID()
	if err != nil {
		return nil, warnly.OIDCState{}, fmt.Errorf("failed to generate state: %w", err)
	}
	nonce, err := capoidc.NewID()
	if err != nil {
		return nil, warnly.OIDCState{}, fmt.Errorf("failed to generate nonce: %w", err)
	}
	opts := []capoidc.Option{capoidc.WithState(state), capoidc.WithNonce(nonce)}
	if len(scopes) > 0 {
		opts = append(opts, capoidc.WithScopes(scopes...))
	}
	if usePKCE {
		verifier, err := capoidc.NewCodeVerifier()
		if err != nil {
			return nil, warnly.OIDCState{}, fmt.Errorf("failed to make pkce verifier: %w", err)
		}
		opts = append(opts, capoidc.WithPKCE(verifier))
	}

	req, err := capoidc.NewRequest(
		oidcStateTimeout,
		redirect,
		opts...,
	)
	if err != nil {
		return nil, warnly.OIDCState{}, fmt.Errorf("failed to create OIDC request: %w", err)
	}

	oidcState := warnly.OIDCState{
		State:  state,
		Nonce:  nonce,
		Scopes: scopes,
	}
	if usePKCE {
		oidcState.CodeVerifier = req.PKCEVerifier().Verifier()
	}

	return req, oidcState, nil
}

func (h *rootHandler) oidcCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.oidc.ProviderName != r.PathValue("provider_name") {
		h.writeError(ctx, w, http.StatusBadRequest, "oidc callback: invalid provider name", nil)
		return
	}

	code, state := r.URL.Query().Get("code"), r.URL.Query().Get("state")
	if code == "" {
		h.writeError(ctx, w, http.StatusBadRequest, "oidc callback: no code", nil)
		return
	}
	if state == "" {
		h.writeError(ctx, w, http.StatusBadRequest, "oidc callback: no state", nil)
		return
	}

	sess, err := h.cookieStore.Get(r, sessionID)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "oidc callback: get session", err)
		return
	}

	cookieState := sess.Values.OIDCState
	if cookieState.State != state {
		h.writeError(ctx, w, http.StatusBadRequest, "oidc callback: no oidc states in session", nil)
		return
	}

	sess.Values.OIDCState = warnly.OIDCState{}

	opts := []capoidc.Option{capoidc.WithState(cookieState.State), capoidc.WithNonce(cookieState.Nonce)}
	if len(sess.Values.OIDCState.Scopes) > 0 {
		opts = append(opts, capoidc.WithScopes(cookieState.Scopes...))
	}
	if cookieState.CodeVerifier != "" {
		verifier, err := capoidc.NewCodeVerifier(capoidc.WithVerifier(cookieState.CodeVerifier))
		if err != nil {
			h.writeError(ctx, w, http.StatusInternalServerError, "oidc callback: recreate verifier", err)
			return
		}
		opts = append(opts, capoidc.WithPKCE(verifier))
	}
	req, err := capoidc.NewRequest(
		oidcStateTimeout,
		h.oidc.Callback,
		opts...,
	)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "oidc callback: recreate request", err)
		return
	}

	token, err := h.oidc.Provider.Exchange(ctx, req, state, code)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "oidc callback: exchange", err)
		return
	}
	if token.IsExpired() {
		h.writeError(ctx, w, http.StatusUnauthorized, "oidc callback: token expired", nil)
		return
	}

	claims := claims{}
	if err := token.IDToken().Claims(&claims); err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "oidc callback: claims", err)
		return
	}
	if claims.Nonce == nil || *claims.Nonce != cookieState.Nonce {
		h.writeError(ctx, w, http.StatusUnauthorized, "oidc callback: nonce mismatch", nil)
		return
	}

	userData := claims.UserData()
	if userData.Email == "" {
		h.writeError(ctx, w, http.StatusUnauthorized, "oidc callback: email is empty", nil)
		return
	}

	result, err := h.svc.GetOrCreateUser(ctx, userData)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "oidc callback: get or create user", err)
		return
	}

	if err := saveCookie(
		w,
		r,
		h.cookieStore,
		*result.User,
		false,
		h.rememberDays); err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "save cookie oidc callback", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// index handles the HTTP request to render the main page with a list of issues.
func (h *rootHandler) index(w http.ResponseWriter, r *http.Request) {
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

// listTagValues handles the request to list values for a tag.
func (h *rootHandler) listTagValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUser(ctx)

	tag := r.URL.Query().Get("tag")
	projectName := r.URL.Query().Get("project_name")
	period := r.URL.Query().Get("period")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	req := &warnly.ListTagValuesRequest{
		User:        &user,
		Tag:         tag,
		ProjectName: projectName,
		Period:      period,
		Limit:       limit,
	}

	values, err := h.projectSvc.ListTagValues(ctx, req)
	if err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list tag values", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(values); err != nil {
		h.writeError(ctx, w, http.StatusInternalServerError, "list tag values: encode", err)
		return
	}
}

// writeIndex writes the index page to the response writer.
func (h *rootHandler) writeIndex(w http.ResponseWriter, r *http.Request, res *warnly.ListIssuesResult, user *warnly.User) {
	ctx := r.Context()

	target := r.Header.Get("Hx-Target")
	partial := r.URL.Query().Get("partial")

	if partial == "body" {
		query := r.URL.Query()
		query.Del("partial")
		w.Header().Set("Hx-Push-Url", "/?"+query.Encode())
	}

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
func (h *rootHandler) destroy(w http.ResponseWriter, r *http.Request) {
	if err := destroySession(w, r, h.cookieStore); err != nil {
		h.logger.Error("destroy session: destroy", slog.Any("error", err))
		if err = web.Login("", "", h.oidc.ProviderName).Render(r.Context(), w); err != nil {
			h.logger.Error("destroy session: login web render", slog.Any("error", err))
		}
		return
	}

	w.Header().Add("Hx-Redirect", "/")
}

// destroySession removes the session cookie.
func destroySession(w http.ResponseWriter, r *http.Request, cookieStore *session.CookieStore) error {
	sess, err := cookieStore.Get(r, "session")
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
func (h *rootHandler) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authURL := ""

	if h.oidc.Provider != nil {
		req, oidcState, err := newOIDCRequest(h.oidc.Scopes, h.oidc.UsePkce, h.oidc.Callback)
		if err != nil {
			h.writeError(ctx, w, http.StatusInternalServerError, "oidc: new request", err)
			return
		}
		sess, err := h.cookieStore.Get(r, "session")
		if err != nil {
			h.writeError(ctx, w, http.StatusInternalServerError, "oidc: get session", err)
			return
		}
		sess.Values.OIDCState = oidcState
		if err := h.cookieStore.Save(r, w, sess); err != nil {
			h.writeError(ctx, w, http.StatusInternalServerError, "oidc: save session", err)
			return
		}
		url, err := h.oidc.Provider.AuthURL(ctx, req)
		if err != nil {
			h.writeError(ctx, w, http.StatusInternalServerError, "oidc: auth url", err)
			return
		}
		authURL = url
	}

	if err := web.Login("", authURL, h.oidc.ProviderName).Render(ctx, w); err != nil {
		h.logger.Error("get session: hello web render", slog.Any("error", err))
	}
}

// create handles the HTTP request to create a new session.
// It authenticates the user and sets the session cookie.
// If the authentication fails, it renders an error page.
// If the authentication succeeds, it redirects to the main page.
func (h *rootHandler) create(w http.ResponseWriter, r *http.Request) {
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
			if err = web.Login(msgInvalidLoginCredentials, "", h.oidc.ProviderName).Render(ctx, w); err != nil {
				h.logger.Error("create new session: login web render", slog.Any("error", err))
			}
			return
		default:
			h.logger.Error("create new session: sign in", slog.Any("error", err))
			if err = web.Login(msgSomethingWentWrong, "", h.oidc.ProviderName).Render(ctx, w); err != nil {
				h.logger.Error("create new session: login web render", slog.Any("error", err))
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
		if err = web.Login(msgSomethingWentWrong, "", h.oidc.ProviderName).Render(ctx, w); err != nil {
			h.logger.Error("create new session: login web render", slog.Any("error", err))
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
	sess, err := cookieStore.Get(r, "session")
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if sess.Values.OIDCState.State != "" {
		sess.Values.OIDCState = warnly.OIDCState{}
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
