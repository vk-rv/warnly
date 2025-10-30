package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	capoidc "github.com/hashicorp/cap/oidc"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/microcosm-cc/bluemonday"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vk-rv/warnly/internal/ch"
	"github.com/vk-rv/warnly/internal/chprometheus"
	"github.com/vk-rv/warnly/internal/migrator"
	"github.com/vk-rv/warnly/internal/mysql"
	"github.com/vk-rv/warnly/internal/server"
	sessionstore "github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/stdlog"
	"github.com/vk-rv/warnly/internal/svc/alert"
	"github.com/vk-rv/warnly/internal/svc/event"
	"github.com/vk-rv/warnly/internal/svc/project"
	"github.com/vk-rv/warnly/internal/svc/session"
	"github.com/vk-rv/warnly/internal/svc/system"
	"github.com/vk-rv/warnly/internal/svcotel"
	"go.uber.org/automaxprocs/maxprocs"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
)

func main() {
	const failed = 1

	cfg := config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		slog.Error("failed to create config", slog.Any("error", err))
		os.Exit(failed)
	}

	logger := stdlog.NewSlogLogger(cfg.Log.Output, cfg.Log.Text)
	slog.SetDefault(logger)

	if err := run(&cfg, logger); err != nil {
		logger.Error("warnly web server start / shutdown problem", slog.Any("error", err))
		os.Exit(failed)
	}
}

//nolint:gocyclo,cyclop // boring initialization.
func run(cfg *config, logger *slog.Logger) error {
	l := func(format string, a ...any) {
		logger.Info(fmt.Sprintf(strings.TrimPrefix(format, "maxprocs: "), a...))
	}
	opt := maxprocs.Logger(l)
	if _, err := maxprocs.Set(opt); err != nil {
		return fmt.Errorf("maxprocs set error: %w", err)
	}

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt)
	termCtx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-term
		logger.Info("signal was received", slog.String("signal", sig.String()))
		cancel()
	}()

	var tracingProvider svcotel.TracerProvider
	if cfg.Tracing.ReporterURI != "" {
		p, err := startTracing(
			termCtx,
			cfg.Tracing.ServiceName,
			cfg.Tracing.ReporterURI,
			cfg.Tracing.Probability,
		)
		if err != nil {
			return fmt.Errorf("start tracing: %w", err)
		}
		tracingProvider = p
	} else {
		tracingProvider = svcotel.NewNoopProvider()
	}

	db, closeDB, err := mysql.ConnectLoop(termCtx, cfg.Database, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err = closeDB(); err != nil {
			logger.Error("close database connection pool on server shutdown", slog.Any("error", err))
		}
	}()

	clickConn, clickClose, err := ch.ConnectLoop(termCtx, cfg.ClickHouse.DSN, ch.DefaultTimeout, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err = clickClose(); err != nil {
			logger.Error("close clickhouse connection pool on server shutdown", slog.Any("error", err))
		}
	}()

	dbm, err := migrator.NewMigrator(cfg.Database.DSN, logger)
	if err != nil {
		return err
	}
	if err = dbm.Up(cfg.ForceMigrate); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	if sourceErr, err := dbm.Close(); sourceErr != nil || err != nil {
		return fmt.Errorf("close oltp migrator: %w, %w", sourceErr, err)
	}
	olapm, err := migrator.NewAnalyticsMigrator(cfg.ClickHouse.DSN, logger)
	if err != nil {
		return err
	}
	if err = olapm.Up(cfg.ForceMigrate); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	if sourceErr, err := olapm.Close(); sourceErr != nil || err != nil {
		return fmt.Errorf("close olap migrator: %w, %w", sourceErr, err)
	}

	userStore := mysql.NewUserStore(db)
	sessionStore := mysql.NewSessionStore(db)
	projectStore := mysql.NewProjectStore(db)
	teamStore := mysql.NewTeamStore(db)
	issueStore := mysql.NewIssueStore(db)
	messageStore := mysql.NewMessageStore(db)
	mentionStore := mysql.NewMentionStore(db)
	assingmentStore := mysql.NewAssingmentStore(db)
	alertStore := mysql.NewAlertStore(db)

	olap := ch.NewClickhouseStore(clickConn, tracingProvider)

	reg := prometheus.NewRegistry()
	regCollectors := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewDBStatsCollector(db, "oltp"),
		chprometheus.NewClickhouseCollector(clickConn, "olap"),
	}
	for i := range regCollectors {
		if err = reg.Register(regCollectors[i]); err != nil {
			return fmt.Errorf("register prometheus collector: %w", err)
		}
	}

	startUOW := mysql.NewUOW(db, logger.With(slog.String("service", "uow")))

	sanitizerPolicy := bluemonday.UGCPolicy()

	now := time.Now

	var publicBaseURL, publicScheme string
	if cfg.PublicIngestURL == "" {
		publicBaseURL = net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
		publicScheme = cfg.Server.Scheme
	} else {
		u, err := url.Parse(cfg.PublicIngestURL)
		if err != nil {
			return fmt.Errorf("parse public ingest URL: %w", err)
		}
		publicScheme = u.Scheme
		publicBaseURL = u.Host
	}

	sessionService := session.NewSessionService(sessionStore, userStore, teamStore, startUOW, now)
	projectService := project.NewProjectService(
		projectStore,
		assingmentStore,
		teamStore,
		issueStore,
		messageStore,
		mentionStore,
		olap,
		startUOW,
		sanitizerPolicy,
		net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
		cfg.Server.Scheme,
		publicBaseURL,
		publicScheme,
		now,
		logger.With(slog.String("service", "project")))
	systemService := system.NewSystemService(olap, now, logger.With(slog.String("service", "system")))

	memoryCache := cache.New(5*time.Minute, 10*time.Minute)

	eventService := event.NewEventService(projectStore, issueStore, memoryCache, olap, now)

	alertService := alert.NewAlertService(alertStore, logger.With(slog.String("service", "alert")))

	isHTTPS := cfg.Server.Scheme == "https"

	cookieStore := sessionstore.NewCookieStore(now, cfg.SessionKey)

	var (
		oidcProvider *capoidc.Provider
		oidcCallback string
		rgxsEmails   []*regexp.Regexp
	)
	if cfg.OIDCProvider.ProviderName != "" {
		if len(cfg.OIDCProvider.EmailMatches) > 0 {
			rgxsEmails = make([]*regexp.Regexp, 0, len(cfg.OIDCProvider.EmailMatches))
			for _, em := range cfg.OIDCProvider.EmailMatches {
				rgx, err := regexp.Compile(em)
				if err != nil {
					return fmt.Errorf("failed compiling email matches string for pattern: %s: %w", em, err)
				}
				rgxsEmails = append(rgxsEmails, rgx)
			}
		}
		cfg.OIDCProvider.RedirectAddress = strings.TrimSuffix(cfg.OIDCProvider.RedirectAddress, "/")
		oidcCallback = cfg.OIDCProvider.RedirectAddress + "/oidc/" + cfg.OIDCProvider.ProviderName + "/callback"
		oidcCfg, err := capoidc.NewConfig(
			cfg.OIDCProvider.IssuerURL,
			cfg.OIDCProvider.ClientID,
			capoidc.ClientSecret(cfg.OIDCProvider.ClientSecret),
			[]capoidc.Alg{oidc.RS256},
			[]string{oidcCallback},
		)
		if err != nil {
			return fmt.Errorf("create oidc config: %w", err)
		}
		oidcProvider, err = capoidc.NewProvider(oidcCfg)
		if err != nil {
			return fmt.Errorf("create oidc provider: %w", err)
		}
		defer oidcProvider.Done()
	}

	var handler http.Handler
	handler, err = server.NewHandler(&server.Backend{
		SessionStore:        sessionStore,
		UserStore:           userStore,
		SessionService:      sessionService,
		EventService:        eventService,
		ProjectService:      projectService,
		SystemService:       systemService,
		AlertService:        alertService,
		IsHTTPS:             isHTTPS,
		RememberSessionDays: cfg.RemeberSessionDays,
		CookieStore:         cookieStore,
		Reg:                 reg,
		Now:                 now,
		Logger:              logger,
		OIDC: &server.OIDC{
			ProviderName: cfg.OIDCProvider.ProviderName,
			Provider:     oidcProvider,
			UsePkce:      cfg.OIDCProvider.UsePKCE,
			Scopes:       cfg.OIDCProvider.Scopes,
			Callback:     oidcCallback,
			EmailMatches: rgxsEmails,
		},
		IsDemo: cfg.IsDemo,
	})
	if err != nil {
		return err
	}

	handler = otelhttp.NewHandler(handler, "/", otelhttp.WithTracerProvider(tracingProvider))

	err = sessionService.CreateUserIfNotExists(termCtx, cfg.Admin.Email, cfg.Admin.Password)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		Handler:           handler,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	if isHTTPS {
		if cfg.Server.CertFile == "" {
			return errors.New("CERT_FILE is required for https scheme")
		}
		if cfg.Server.CertKey == "" {
			return errors.New("CERT_KEY is required for https scheme")
		}
		if _, err = os.Stat(cfg.Server.CertFile); err != nil {
			return fmt.Errorf("CERT_FILE os stat: %w", err)
		}
		if _, err = os.Stat(cfg.Server.CertKey); err != nil {
			return fmt.Errorf("CERT_KEY os stat: %w", err)
		}
	}

	go func() {
		if isHTTPS {
			err = srv.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.CertKey)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("listen on specified port", slog.Any("error", err))
				cancel()
			}
		} else {
			err = srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("listen on specified port", slog.Any("error", err))
				cancel()
			}
		}
	}()

	var metricsSrv *http.Server
	if cfg.Metrics.Enabled {
		router := http.NewServeMux()
		router.Handle(cfg.Metrics.Path, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
			ErrorLog: slog.NewLogLogger(logger.With(slog.String("service", "prometheus")).
				Handler(), slog.LevelError),
			Timeout: time.Second * 1,
		}))
		metricsSrv = &http.Server{
			Addr:              net.JoinHostPort(cfg.Server.Host, cfg.Metrics.Port),
			Handler:           router,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      15 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
			ErrorLog: slog.NewLogLogger(
				logger.With(slog.String("service", "metrics_server")).
					Handler(), slog.LevelError),
		}
		go func() {
			if isHTTPS {
				err = metricsSrv.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.CertKey)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("listen on specified port for metrics", slog.Any("error", err))
					cancel()
				}
			} else {
				err = metricsSrv.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("listen on specified port for metrics", slog.Any("error", err))
					cancel()
				}
			}
		}()
	}

	metricsPort := ""
	if metricsSrv != nil {
		metricsPort = cfg.Metrics.Port
	}

	logger.Info("server started",
		slog.String("scheme", cfg.Server.Scheme),
		slog.String("host", cfg.Server.Host),
		slog.String("port", cfg.Server.Port),
		slog.String("metrics_port", metricsPort),
		slog.String("runtime", runtime.Version()),
		slog.String("os", runtime.GOOS))

	<-termCtx.Done()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), cfg.Server.CloseTimeout)
	defer cancel()

	if err = srv.Shutdown(ctxShutDown); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	if metricsSrv != nil {
		if err = metricsSrv.Shutdown(ctxShutDown); err != nil {
			return fmt.Errorf("graceful shutdown for metrics failed: %w", err)
		}
	}

	logger.Info("server exited properly")

	return nil
}

// startTracing configure open telemetry to be used.
func startTracing(ctx context.Context, serviceName, reporterURI string, probability float64) (*trace.TracerProvider, error) {
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(reporterURI),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(probability)),
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			),
		),
	)

	// We must set this provider as the global provider for things to work,
	// but we pass this provider around the program where needed to collect
	// our traces.
	otel.SetTracerProvider(traceProvider)

	// Chooses the HTTP header formats we extract incoming trace contexts from,
	// and the headers we set in outgoing requests.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return traceProvider, nil
}

//nolint:tagalign // later
type config struct {
	OIDCProvider struct {
		ProviderName    string   `env:"OIDC_PROVIDER_NAME"`
		IssuerURL       string   `env:"OIDC_ISSUER_URL"`
		ClientID        string   `env:"OIDC_CLIENT_ID"`
		ClientSecret    string   `env:"OIDC_CLIENT_SECRET"`
		RedirectAddress string   `env:"OIDC_REDIRECT_ADDRESS"`
		Scopes          []string `env:"OIDC_SCOPES"`
		EmailMatches    []string `env:"OIDC_EMAIL_MATCHES"`
		UsePKCE         bool     `env:"OIDC_USE_PKCE" env-default:"true"`
	}
	Admin struct {
		Email    string `env:"ADMIN_EMAIL"    env-required:"true"`
		Password string `env:"ADMIN_PASSWORD" env-required:"true"`
	}
	ClickHouse struct {
		DSN string `env:"CLICKHOUSE_DSN" env-required:"true"`
	}
	Server struct {
		Host         string        `env:"SERVER_HOST"   env-default:"localhost"`
		Port         string        `env:"SERVER_PORT"   env-default:"8080"`
		Scheme       string        `env:"SCHEME"        env-default:"http"`
		CertFile     string        `env:"CERT_FILE"`
		CertKey      string        `env:"CERT_KEY"`
		CloseTimeout time.Duration `env:"CLOSE_TIMEOUT" env-default:"5s"`
	}
	Metrics struct {
		Port    string `env:"METRICS_PORT"    env-default:"8081"`
		Path    string `env:"METRICS_PATH"    env-default:"/metrics"`
		Enabled bool   `env:"METRICS_ENABLED" env-default:"false"`
	}
	Log struct {
		Output string `env:"LOG_OUTPUT" env-default:"stderr"`
		Text   bool   `env:"LOG_TEXT"   env-default:"false"`
	}
	Tracing struct {
		ReporterURI string  `env:"TRACING_REPORTER_URI" env-default:""`
		ServiceName string  `env:"TRACING_SERVICE_NAME" env-default:"warnly"`
		Probability float64 `env:"TRACING_PROBABILITY"  env-default:"1.0"`
	}
	PublicIngestURL    string `env:"PUBLIC_INGEST_URL"`
	SessionKey         []byte `env:"SESSION_KEY" env-required:"true"`
	Database           mysql.DBConfig
	RemeberSessionDays int  `env:"REMEMBER_SESSION_DAYS" env-default:"30"`
	ForceMigrate       bool `env:"FORCE_MIGRATE"         env-default:"false"`
	IsDemo             bool `env:"IS_DEMO"               env-default:"false"`
}
