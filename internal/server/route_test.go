package server_test

import (
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/server"
	"github.com/vk-rv/warnly/internal/session"
)

func TestNewHandlerReturnsValidHandler(t *testing.T) {
	t.Parallel()

	backend := &server.Backend{
		Now:                 time.Now,
		SessionStore:        nil,
		UserStore:           nil,
		SessionService:      nil,
		EventService:        nil,
		ProjectService:      nil,
		SystemService:       nil,
		AlertService:        nil,
		NotificationService: nil,
		OIDC: &server.OIDC{
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

	handler, err := server.NewHandler(backend)
	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.ServeMux)
}
