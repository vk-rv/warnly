package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/web"
)

//go:embed asset/static
var Asset embed.FS

// Backend is all services and associated parameters required to construct a Handler.
type Backend struct {
	Now                 func() time.Time
	SessionStore        warnly.SessionStore
	UserStore           warnly.UserStore
	SessionService      warnly.SessionService
	EventService        warnly.EventService
	ProjectService      warnly.ProjectService
	SystemService       warnly.SystemService
	Reg                 *prometheus.Registry
	Logger              *slog.Logger
	CookieStore         *session.CookieStore
	RememberSessionDays int
	IsHTTPS             bool
}

// Handler is a collection of all the service handlers.
type Handler struct {
	*http.ServeMux
}

// NewHandler initialize dependencies and returns router with attached routes.
func NewHandler(b *Backend) (*Handler, error) {
	initValidator()

	mux := http.NewServeMux()

	authenticateMw := newAuthMW(b.CookieStore, b.Logger.With(
		slog.String("middleware", "auth"),
	))
	recoverMw := newRecoverMw(b.Reg, b.Logger.With(
		slog.String("middleware", "recover"),
	))

	prometheusMw := newPrometheusMW(b.Reg, b.Now)

	chainWithoutAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = recoverMw.recover(handler)
		handler = prometheusMw.recordLatency(handler)
		csrfMiddleware := http.NewCrossOriginProtection()
		handler = http.HandlerFunc(csrfMiddleware.Handler(handler).ServeHTTP)
		return handler
	}

	chain := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = authenticateMw.authenticate(handler)
		return chainWithoutAuth(handler)
	}

	systemHandler := newSystemHandler(b.SystemService, b.CookieStore, b.Logger.With(
		slog.String("handler", "system"),
	))
	mux.HandleFunc("GET /system", chain(systemHandler.listSlowQueries))
	mux.HandleFunc("GET /system/schema", chain(systemHandler.listSchemas))
	mux.HandleFunc("GET /system/errors", chain(systemHandler.listErrors))

	settingsHandler := newSettingsHandler(b.Logger.With(
		slog.String("handler", "settings"),
	))
	mux.HandleFunc("GET /settings", chain(settingsHandler.listSettings))

	sessionHandler := newSessionHandler(
		b.SessionService,
		b.ProjectService,
		b.CookieStore,
		b.RememberSessionDays,
		b.Logger.With(
			slog.String("handler", "session"),
		))

	eventAPIHandler := newEventAPIHandler(b.EventService, b.Logger.With(
		slog.String("handler", "event"),
	))

	projectHandler := newProjectHandler(b.ProjectService, b.CookieStore, b.Logger.With(
		slog.String("handler", "project"),
	))

	issueHandler := newIssueHandler(b.ProjectService, b.CookieStore, b.Logger.With(
		slog.String("handler", "issues"),
	))
	mux.HandleFunc("GET /issues", chain(issueHandler.listIssues))

	mux.HandleFunc("GET /notready", chain(func(w http.ResponseWriter, r *http.Request) {
		if err := web.InDevelopment().Render(r.Context(), w); err != nil {
			b.Logger.Error("not ready web render", slog.Any("error", err))
		}
	}))

	mux.HandleFunc("GET /oncall", chain(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(htmxHeader) != "" {
			if err := web.OnCallHtmx().Render(r.Context(), w); err != nil {
				b.Logger.Error("not ready web render", slog.Any("error", err))
			}
		} else {
			user := getUser(r.Context())
			if err := web.OnCall(&user).Render(r.Context(), w); err != nil {
				b.Logger.Error("not ready web render", slog.Any("error", err))
			}
		}
	}))

	mux.HandleFunc("GET /analytics", chain(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(htmxHeader) != "" {
			if err := web.ReportsHtmx().Render(r.Context(), w); err != nil {
				b.Logger.Error("not ready web render", slog.Any("error", err))
			}
		} else {
			user := getUser(r.Context())
			if err := web.Reports(&user).Render(r.Context(), w); err != nil {
				b.Logger.Error("not ready web render", slog.Any("error", err))
			}
		}
	}))

	mux.HandleFunc("GET /settings/projects/{id}", chain(projectHandler.projectSettings))

	mux.HandleFunc("GET /projects/q", chain(projectHandler.searchProjectByName))
	mux.HandleFunc("GET /projects/{id}", chain(projectHandler.projectDetails))
	mux.HandleFunc("GET /projects", chain(projectHandler.listProjects))
	mux.HandleFunc("GET /projects/new", chain(projectHandler.getPlatforms))
	mux.HandleFunc("POST /projects", chain(projectHandler.createProject))
	mux.HandleFunc("GET /projects/{projectID}/getting-started", chain(projectHandler.gettingStarted))
	mux.HandleFunc("DELETE /projects/{id}", chain(projectHandler.deleteProject))
	mux.HandleFunc("GET /projects/{project_id}/issues/{issue_id}", chain(projectHandler.getIssue))
	mux.HandleFunc("GET /projects/{project_id}/issues/{issue_id}/discussions", chain(projectHandler.getDiscussions))
	mux.HandleFunc("POST /projects/{project_id}/issues/{issue_id}/discussions", chain(projectHandler.postMessage))
	mux.HandleFunc("DELETE /projects/{project_id}/issues/{issue_id}/discussions/{message_id}", chain(projectHandler.deleteMessage))
	mux.HandleFunc("GET /projects/{project_id}/issues/{issue_id}/fields", chain(projectHandler.listFields))
	mux.HandleFunc("GET /projects/{project_id}/issues/{issue_id}/events", chain(projectHandler.listEvents))
	mux.HandleFunc("POST /projects/{project_id}/issues/{issue_id}/assignments", chain(projectHandler.assignIssue))
	mux.HandleFunc("DELETE /projects/{project_id}/issues/{issue_id}/assignments", chain(projectHandler.deleteAssignment))

	mux.HandleFunc("GET /error", chain(func(w http.ResponseWriter, r *http.Request) {
		if err := web.ServerError().Render(r.Context(), w); err != nil {
			b.Logger.Error("server error web render", slog.Any("error", err))
		}
	}))

	subFs, err := fs.Sub(Asset, "asset/static")
	if err != nil {
		return nil, fmt.Errorf("sub fs: %w", err)
	}
	embedRoot := http.FileServer(http.FS(subFs))
	fsHandler := func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static", embedRoot).ServeHTTP(w, r)
	}

	mux.HandleFunc("GET /static/", fsHandler)

	mux.HandleFunc("GET /", chain(sessionHandler.index))
	mux.HandleFunc("DELETE /session", chain(sessionHandler.destroy))

	mux.HandleFunc("POST /api/v1/events", chain(eventAPIHandler.ingestEvent))

	mux.HandleFunc("POST /ingest/api/{project_id}/envelope/", eventAPIHandler.ingestEvent)

	mux.HandleFunc("GET /login", chainWithoutAuth(sessionHandler.login))
	mux.HandleFunc("POST /login", chainWithoutAuth(sessionHandler.create))

	return &Handler{ServeMux: mux}, nil
}
