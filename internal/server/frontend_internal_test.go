package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestFrontendDeepLinksAndAssets(t *testing.T) {
	t.Parallel()
	handler := serveFrontend(fstest.MapFS{
		"dist/index.html":            {Data: []byte("<!doctype html><div>Warnly</div>")},
		"dist/_app/immutable/app.js": {Data: []byte("console.log('app')")},
	})
	for _, path := range []string{"/", "/login", "/projects/123", "/projects/123/issues/456", "/projects/123/issues/456/events", "/alerts/3/edit", "/settings/projects/3", "/system/schema"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, w.Code, path)
		require.Contains(t, w.Body.String(), "<!doctype html>")
		require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	}
	for _, path := range []string{"/missing", "/api/missing", "/_app/missing.js", "/_app/", "/README.txt"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, w.Code, path)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/_app/immutable/app.js", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "javascript")
	require.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	require.Equal(t, "console.log('app')", w.Body.String())
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/projects/123", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Body.String())
}

func TestFrontendMissingBuild(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	serveFrontend(fstest.MapFS{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "make frontend-build")
}

type uiSessionService struct{ warnly.SessionService }

func (uiSessionService) SignIn(_ context.Context, credentials *warnly.Credentials) (*warnly.Session, error) {
	if credentials.Identifier != "admin" || credentials.Password != "password" {
		return nil, warnly.ErrInvalidLoginCredentials
	}
	return &warnly.Session{User: &warnly.User{ID: 1, Email: "admin@example.com", Username: "admin"}}, nil
}

type uiProjectService struct{ warnly.ProjectService }

func (uiProjectService) ListProjects(_ context.Context, criteria *warnly.ListProjectsCriteria, user *warnly.User) (*warnly.ListProjectsResult, error) {
	if user.ID != 1 {
		return nil, errors.New("unexpected user")
	}
	return &warnly.ListProjectsResult{Criteria: criteria, Projects: []warnly.Project{{ID: 1, Name: "api"}}}, nil
}

func TestUISessionAndAPI(t *testing.T) {
	t.Parallel()
	b := &Backend{Now: time.Now, Logger: slog.New(slog.DiscardHandler), Reg: prometheus.NewRegistry(),
		CookieStore:    session.NewCookieStore(time.Now, []byte("01234567890123456789012345678901")),
		OIDC:           &OIDC{EmailMatches: []*regexp.Regexp{regexp.MustCompile(`@example\.com$`)}},
		SessionService: uiSessionService{}, ProjectService: uiProjectService{}, RememberSessionDays: 30,
	}
	handler, err := NewHandler(b)
	require.NoError(t, err)
	request := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	w := request("GET", "/api/projects", "", nil)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.JSONEq(t, `{"error":"Authentication required"}`, w.Body.String())
	w = request("POST", "/api/login", "identifier=admin&password=wrong", nil)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, w.Result().Cookies())
	w = request("POST", "/api/login", url.Values{"identifier": {"admin"}, "password": {"password"}, "remember-me": {"on"}}.Encode(), nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, w.Result().Cookies())
	cookie := w.Result().Cookies()[0]
	require.True(t, cookie.HttpOnly)
	w = request("GET", "/api/session", "", cookie)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"Username":"admin"`)
	w = request("GET", "/api/projects?name=api", "", cookie)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"Name":"api"`)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	w = request("GET", "/api/missing", "", cookie)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "application/json")
	req := httptest.NewRequest("POST", "/api/projects", strings.NewReader("projectName=bad"))
	req.AddCookie(cookie)
	req.Header.Set("Origin", "https://untrusted.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
	w = request("DELETE", "/api/session", "", cookie)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Less(t, w.Result().Cookies()[0].MaxAge, 0)
}
