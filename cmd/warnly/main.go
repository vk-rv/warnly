package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
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
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"github.com/vk-rv/warnly/internal/ch"
	"github.com/vk-rv/warnly/internal/chprometheus"
	"github.com/vk-rv/warnly/internal/kafka"
	"github.com/vk-rv/warnly/internal/migrator"
	"github.com/vk-rv/warnly/internal/mysql"
	"github.com/vk-rv/warnly/internal/notifier"
	"github.com/vk-rv/warnly/internal/server"
	sessionstore "github.com/vk-rv/warnly/internal/session"
	"github.com/vk-rv/warnly/internal/stdlog"
	"github.com/vk-rv/warnly/internal/svc/alert"
	"github.com/vk-rv/warnly/internal/svc/event"
	"github.com/vk-rv/warnly/internal/svc/notification"
	"github.com/vk-rv/warnly/internal/svc/project"
	"github.com/vk-rv/warnly/internal/svc/session"
	"github.com/vk-rv/warnly/internal/svc/system"
	"github.com/vk-rv/warnly/internal/svcotel"
	"github.com/vk-rv/warnly/internal/warnly"
	"github.com/vk-rv/warnly/internal/worker"
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
	const (
		failed = 1
		stdout = "stdout"
	)

	cfg := config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		slog.Error("failed to create config", slog.Any("error", err))
		os.Exit(failed)
	}

	var w io.Writer = os.Stderr
	if cfg.Log.Output == stdout {
		w = os.Stdout
	}

	logger := stdlog.NewSlogLogger(w, cfg.Log.Text)
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

	reg := prometheus.NewRegistry()

	var kafkaProducer warnly.Producer
	if len(cfg.Kafka.Brokers) > 0 {
		logger.Info("kafka producer enabled, connecting to brokers", slog.String("brokers", strings.Join(cfg.Kafka.Brokers, ",")))

		kafkaTLS, err := buildKafkaTLSConfig(cfg)
		if err != nil {
			return fmt.Errorf("build kafka tls config: %w", err)
		}
		kafkaSASL, err := buildKafkaSASLMechanism(cfg)
		if err != nil {
			return fmt.Errorf("build kafka sasl mechanism: %w", err)
		}

		kafkaProducer, err = kafka.NewProducer(&kafka.ProducerConfig{
			CommonConfig: kafka.CommonConfig{
				TracerProvider:        tracingProvider,
				Namespace:             cfg.Kafka.Namespace,
				Brokers:               cfg.Kafka.Brokers,
				ClientID:              cfg.Kafka.ClientID,
				Logger:                logger.With(slog.String("service", "kafka_producer")),
				DisableTelemetry:      cfg.Kafka.DisableTelemetry,
				MetadataMaxAge:        cfg.Kafka.MetadataMaxAge,
				EnableKafkaHistograms: true,
				TLS:                   kafkaTLS,
				SASL:                  kafkaSASL,
			},
			Reg:  reg,
			Sync: cfg.Kafka.ProducerSync,
		})
		if err != nil {
			return fmt.Errorf("failed creating kafka producer: %w", err)
		}
		defer func() {
			if err = kafkaProducer.Close(); err != nil {
				logger.Error("close kafka producer on server shutdown", slog.Any("error", err))
			}
		}()
	}

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
	notificationStore := mysql.NewNotificationStore(db)

	olap := ch.NewClickhouseStore(clickConn, tracingProvider)

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

	eventService := event.NewEventService(
		projectStore,
		issueStore,
		memoryCache,
		olap,
		event.Queue{
			Enabled:  len(cfg.Kafka.Brokers) > 0,
			Producer: kafkaProducer,
		},
		now)

	alertService := alert.NewAlertService(alertStore, projectStore, teamStore, now, logger.With(slog.String("service", "alert")))

	webhookNotifier := notifier.NewWebhookNotifier(
		notificationStore,
		cfg.NotificationEncryptionKey,
		&http.Client{
			Timeout: 10 * time.Second,
		},
		now,
		logger.With(slog.String("service", "webhook_notifier")),
	)

	notificationService := notification.NewNotificationService(
		notificationStore,
		teamStore,
		webhookNotifier,
		now,
		logger.With(slog.String("service", "notification")),
	)

	alertWorker := worker.NewAlertWorker(
		alertStore,
		olap,
		issueStore,
		notificationStore,
		webhookNotifier,
		now,
		cfg.AlertWorkerInterval,
		warnly.NewUUID().String(),
		logger.With(slog.String("service", "alert_worker")),
	)
	defer alertWorker.Stop()

	go alertWorker.Start(termCtx)

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
		NotificationService: notificationService,
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
		if cfg.Metrics.PprofEnabled {
			router.HandleFunc("/debug/pprof/", pprof.Index)
			router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			router.HandleFunc("/debug/pprof/profile", pprof.Profile)
			router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			router.HandleFunc("/debug/pprof/trace", pprof.Trace)
			router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
			router.Handle("/debug/pprof/block", pprof.Handler("block"))
			router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
			router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		}
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

//nolint:nilnil // nil is a valid value for configuration tls.
func buildKafkaTLSConfig(cfg *config) (*tls.Config, error) {
	if !cfg.Kafka.TLS.Enabled {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if cfg.Kafka.TLS.CertFile != "" && cfg.Kafka.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Kafka.TLS.CertFile, cfg.Kafka.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load kafka tls key pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	if cfg.Kafka.TLS.CAFile != "" {
		caCert, err := os.ReadFile(cfg.Kafka.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read kafka tls ca file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to append kafka CA certificate")
		}
		tlsCfg.RootCAs = caCertPool
	}
	return tlsCfg, nil
}

//nolint:nilnil // nil is a valid value for configuration sasl.
func buildKafkaSASLMechanism(cfg *config) (kafka.SASLMechanism, error) {
	if cfg.Kafka.SASL.Plain.Enabled {
		return plain.Auth{
			User: cfg.Kafka.SASL.Plain.User,
			Pass: cfg.Kafka.SASL.Plain.Pass,
		}.AsMechanism(), nil
	}
	if cfg.Kafka.SASL.SCRAM.Enabled {
		var auth scram.Auth
		auth.User = cfg.Kafka.SASL.SCRAM.User
		auth.Pass = cfg.Kafka.SASL.SCRAM.Pass
		switch cfg.Kafka.SASL.SCRAM.Algorithm {
		case "SCRAM-SHA-256":
			return auth.AsSha256Mechanism(), nil
		case "SCRAM-SHA-512":
			return auth.AsSha512Mechanism(), nil
		default:
			return nil, fmt.Errorf("unsupported SCRAM algorithm: %s (use SCRAM-SHA-256 or SCRAM-SHA-512)", cfg.Kafka.SASL.SCRAM.Algorithm)
		}
	}
	return nil, nil
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
		Port         string `env:"METRICS_PORT"         env-default:"8081"`
		Path         string `env:"METRICS_PATH"         env-default:"/metrics"`
		Enabled      bool   `env:"METRICS_ENABLED"      env-default:"false"`
		PprofEnabled bool   `env:"METRICS_PPROF_ENABLED" env-default:"false"`
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
	Kafka struct {
		TLS struct {
			CertFile string `env:"KAFKA_TLS_CERT_FILE"`
			KeyFile  string `env:"KAFKA_TLS_KEY_FILE"`
			CAFile   string `env:"KAFKA_TLS_CA_FILE"`
			Enabled  bool   `env:"KAFKA_TLS_ENABLED"   env-default:"false"`
		}
		SASL struct {
			Plain struct {
				User    string `env:"KAFKA_SASL_PLAIN_USER"`
				Pass    string `env:"KAFKA_SASL_PLAIN_PASS"`
				Enabled bool   `env:"KAFKA_SASL_PLAIN_ENABLED" env-default:"false"`
			}
			SCRAM struct {
				Algorithm string `env:"KAFKA_SASL_SCRAM_ALGORITHM"` // "SCRAM-SHA-256" or "SCRAM-SHA-512"
				User      string `env:"KAFKA_SASL_SCRAM_USER"`
				Pass      string `env:"KAFKA_SASL_SCRAM_PASS"`
				Enabled   bool   `env:"KAFKA_SASL_SCRAM_ENABLED"   env-default:"false"`
			}
		}
		Namespace        string        `env:"KAFKA_NAMESPACE"         env-default:"warnly"`
		ClientID         string        `env:"KAFKA_CLIENT_ID"         env-default:"warnly"`
		Brokers          []string      `env:"KAFKA_BROKERS"`
		MetadataMaxAge   time.Duration `env:"KAFKA_METADATA_MAX_AGE"  env-default:"60s"`
		ProducerSync     bool          `env:"KAFKA_PRODUCER_SYNC"     env-default:"false"`
		DisableTelemetry bool          `env:"KAFKA_DISABLE_TELEMETRY" env-default:"false"`
	}
	PublicIngestURL           string `env:"PUBLIC_INGEST_URL"`
	SessionKey                []byte `env:"SESSION_KEY" env-required:"true"`
	NotificationEncryptionKey []byte `env:"NOTIFICATION_ENCRYPTION_KEY" env-required:"true"`
	Database                  mysql.DBConfig
	AlertWorkerInterval       time.Duration `env:"ALERT_WORKER_INTERVAL" env-default:"1m"`
	RemeberSessionDays        int           `env:"REMEMBER_SESSION_DAYS" env-default:"30"`
	ForceMigrate              bool          `env:"FORCE_MIGRATE"         env-default:"false"`
	IsDemo                    bool          `env:"IS_DEMO"               env-default:"false"`
}
