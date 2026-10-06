package server

import (
	"log/slog"
	"net/http"
	"regexp"
	"time"

	capoidc "github.com/hashicorp/cap/oidc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/warnly"
)

// Backend is all services and associated parameters required to construct a Handler.
type Backend struct {
	Now                 func() time.Time
	SessionStore        warnly.SessionStore
	UserStore           warnly.UserStore
	SessionService      warnly.SessionService
	EventService        warnly.EventService
	ProjectService      warnly.ProjectService
	SystemService       warnly.SystemService
	AlertService        warnly.AlertService
	NotificationService warnly.NotificationService
	OIDC                *OIDC
	Reg                 *prometheus.Registry
	Logger              *slog.Logger
	CookieStore         *session.CookieStore
	RememberSessionDays int
	IsHTTPS             bool
	IsDemo              bool
}

type OIDC struct {
	ProviderName string
	Nonce        string
	Callback     string
	Provider     *capoidc.Provider
	Scopes       []string
	EmailMatches []*regexp.Regexp
	UsePkce      bool
}

// Handler is a collection of all the service handlers.
type Handler struct {
	*http.ServeMux
}

// NewHandler initialize dependencies and returns router with attached routes.
func NewHandler(b *Backend) (*Handler, error) {
	mux := http.NewServeMux()
	b.CookieStore.Options.HTTPOnly = true
	b.CookieStore.Options.Secure = b.IsHTTPS
	b.CookieStore.Options.SameSite = http.SameSiteLaxMode

	authenticateMw := newAuthMW(b.CookieStore, b.Logger.With(
		slog.String("middleware", "auth"),
	))
	recoverMw := newRecoverMw(b.Reg, b.Logger.With(
		slog.String("middleware", "recover"),
	))

	prometheusMw := newPrometheusMW(b.Reg, b.Now)

	emailMatcherMw := newEmailMatcherMW(b.OIDC.EmailMatches, b.Logger)

	chainWithoutAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = prometheusMw.recordLatency(handler)
		handler = recoverMw.recover(handler)
		csrfMiddleware := http.NewCrossOriginProtection()
		handler = http.HandlerFunc(csrfMiddleware.Handler(handler).ServeHTTP)
		return handler
	}

	chain := func(handler http.HandlerFunc) http.HandlerFunc {
		if len(b.OIDC.EmailMatches) > 0 {
			handler = emailMatcherMw.emailMatch(handler)
		}
		handler = authenticateMw.authenticate(handler)
		return chainWithoutAuth(handler)
	}

	systemHandler := newSystemHandler(b.SystemService, b.CookieStore, b.Logger.With(
		slog.String("handler", "system"),
	))
	mux.HandleFunc("GET /api/system", chain(systemHandler.listSlowQueries))
	mux.HandleFunc("GET /api/system/schema", chain(systemHandler.listSchemas))
	mux.HandleFunc("GET /api/system/errors", chain(systemHandler.listErrors))

	settingsHandler := newSettingsHandler(b.NotificationService, b.Logger.With(
		slog.String("handler", "settings"),
	))
	mux.HandleFunc("GET /api/settings", chain(settingsHandler.listSettings))

	rootHandler := newRootHandler(
		b.SessionService,
		b.ProjectService,
		b.CookieStore,
		b.RememberSessionDays,
		b.OIDC,
		b.IsDemo,
		b.Logger.With(
			slog.String("handler", "session"),
		))

	eventAPIHandler := NewEventAPIHandler(b.EventService, b.Logger.With(
		slog.String("handler", "event"),
	))

	projectHandler := NewProjectHandler(b.ProjectService, b.Logger.With(
		slog.String("handler", "project"),
	))

	alertsHandler := NewAlertsHandler(b.AlertService, b.Logger.With(
		slog.String("handler", "alerts"),
	))

	notificationHandler := newNotificationHandler(b.NotificationService, b.Logger.With(
		slog.String("handler", "notification"),
	))

	mux.HandleFunc("GET /api/settings/projects/{id}", chain(projectHandler.ProjectSettings))

	mux.HandleFunc("GET /api/projects/q", chain(projectHandler.SearchProjectByName))
	mux.HandleFunc("GET /api/projects/{id}", chain(projectHandler.ProjectDetails))
	mux.HandleFunc("GET /api/projects", chain(projectHandler.ListProjects))
	mux.HandleFunc("GET /api/projects/new", chain(projectHandler.GetPlatforms))
	mux.HandleFunc("POST /api/projects", chain(projectHandler.CreateProject))
	mux.HandleFunc("GET /api/projects/{projectID}/getting-started", chain(projectHandler.GettingStarted))
	mux.HandleFunc("DELETE /api/projects/{id}", chain(projectHandler.DeleteProject))
	mux.HandleFunc("GET /api/projects/{project_id}/issues/{issue_id}", chain(projectHandler.GetIssue))
	mux.HandleFunc("GET /api/projects/{project_id}/issues/{issue_id}/discussions", chain(projectHandler.GetDiscussions))
	mux.HandleFunc("POST /api/projects/{project_id}/issues/{issue_id}/discussions", chain(projectHandler.PostMessage))
	mux.HandleFunc("DELETE /api/projects/{project_id}/issues/{issue_id}/discussions/{message_id}", chain(projectHandler.DeleteMessage))
	mux.HandleFunc("GET /api/projects/{project_id}/issues/{issue_id}/fields", chain(projectHandler.ListFields))
	mux.HandleFunc("GET /api/projects/{project_id}/issues/{issue_id}/events", chain(projectHandler.ListEvents))
	mux.HandleFunc("POST /api/projects/{project_id}/issues/{issue_id}/assignments", chain(projectHandler.AssignIssue))
	mux.HandleFunc("DELETE /api/projects/{project_id}/issues/{issue_id}/assignments", chain(projectHandler.DeleteAssignment))

	mux.HandleFunc("GET /api/alerts", chain(alertsHandler.ListAlerts))
	mux.HandleFunc("GET /api/alerts/new", chain(alertsHandler.CreateAlertGet))
	mux.HandleFunc("POST /api/alerts", chain(alertsHandler.CreateAlert))
	mux.HandleFunc("GET /api/alerts/{id}/edit", chain(alertsHandler.EditAlertGet))
	mux.HandleFunc("PUT /api/alerts/{id}", chain(alertsHandler.UpdateAlert))
	mux.HandleFunc("DELETE /api/alerts/{id}", chain(alertsHandler.DeleteAlert))

	mux.HandleFunc("POST /api/settings/webhook", chain(notificationHandler.SaveWebhook))

	mux.HandleFunc("GET /api/login", chainWithoutAuth(rootHandler.login))
	mux.HandleFunc("POST /api/login", chainWithoutAuth(rootHandler.create))
	mux.HandleFunc("GET /api/issues", chain(rootHandler.index))
	mux.HandleFunc("GET /oidc/{provider_name}/callback", chainWithoutAuth(rootHandler.oidcCallback))
	mux.HandleFunc("GET /api/search/tag-values", chain(rootHandler.listTagValues))
	mux.HandleFunc("DELETE /api/session", chain(rootHandler.destroy))

	mux.HandleFunc("POST /ingest/api/{project_id}/envelope/", chainWithoutAuth(eventAPIHandler.IngestEvent))

	mux.HandleFunc("GET /api/session", chain(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, getUser(r.Context()))
	}))
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
	})
	frontend, err := newFrontendHandler()
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /", frontend)
	return &Handler{ServeMux: mux}, nil
}
