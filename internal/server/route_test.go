package server

import (
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vk-rv/warnly/internal/session"
)

func TestNewHandlerReturnsValidHandler(t *testing.T) {
	t.Parallel()

	backend := &Backend{
		Now:                 time.Now,
		SessionStore:        nil,
		UserStore:           nil,
		SessionService:      nil,
		EventService:        nil,
		ProjectService:      nil,
		SystemService:       nil,
		AlertService:        nil,
		NotificationService: nil,
		OIDC: &OIDC{
			ProviderName: "test",
			EmailMatches: []*regexp.Regexp{},
		},
		Reg:                 prometheus.NewRegistry(),
		Logger:              slog.Default(),
		CookieStore:         session.NewCookieStore(time.Now, []byte("test-secret-key")),
		RememberSessionDays: 7,
		IsHTTPS:             true,
		IsDemo:              false,
	}

	handler, err := NewHandler(backend)
	if err != nil {
		t.Fatalf("NewHandler() error = %v, want nil", err)
	}

	if handler == nil {
		t.Fatal("NewHandler() returned nil handler, want *Handler")
	}

	if handler.ServeMux == nil {
		t.Fatal("NewHandler() handler.ServeMux is nil, want *http.ServeMux")
	}
}
