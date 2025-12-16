// Package kafka provides a Kafka integration for Warnly.
package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"github.com/twmb/franz-go/plugin/kotel"
	"github.com/twmb/franz-go/plugin/kprom"
	"github.com/twmb/franz-go/plugin/kslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	MetricsPrefix = "warnly_kafka_client"
)

type KafkaConfig struct {
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

// SASLMechanism type alias to sasl.Mechanism.
type SASLMechanism = sasl.Mechanism

// CommonConfig defines common configuration for Kafka consumers, producers,
// and managers.
type CommonConfig struct {
	SASL                  SASLMechanism
	TracerProvider        trace.TracerProvider
	Logger                *slog.Logger
	Dialer                func(ctx context.Context, network, address string) (net.Conn, error)
	TLS                   *tls.Config
	ClientID              string
	Version               string
	Namespace             string
	Brokers               []string
	hooks                 []kgo.Hook
	MetadataMaxAge        time.Duration
	DisableTelemetry      bool
	EnableKafkaHistograms bool
}

// finalize ensures the configuration is valid.
func (cfg *CommonConfig) finalize() {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Namespace != "" {
		cfg.Logger = cfg.Logger.With(slog.String("namespace", cfg.Namespace))
	}
}

func (cfg *CommonConfig) newClientWithOpts(
	reg prometheus.Registerer,
	clientOptsFn []clientOptsFn,
	additionalOpts ...kgo.Opt,
) (*kgo.Client, error) {
	clOpts := &clientOpts{
		reg: reg,
	}
	for _, opt := range clientOptsFn {
		opt(clOpts)
	}

	opts := []kgo.Opt{
		kgo.WithLogger(kslog.New(cfg.Logger)),
		kgo.SeedBrokers(cfg.Brokers...),
	}
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
		if cfg.Version != "" {
			opts = append(opts, kgo.SoftwareNameAndVersion(
				cfg.ClientID, cfg.Version,
			))
		}
	}
	if cfg.Dialer != nil {
		opts = append(opts, kgo.Dialer(cfg.Dialer))
	} else if cfg.TLS != nil {
		opts = append(opts, kgo.DialTLSConfig(cfg.TLS.Clone()))
	}
	if cfg.SASL != nil {
		opts = append(opts, kgo.SASL(cfg.SASL))
	}
	opts = append(opts, additionalOpts...)
	if !cfg.DisableTelemetry {
		opts = append(opts, kgo.WithHooks(
			kotel.NewTracer(
				kotel.TracerProvider(cfg.tracerProvider()),
			),
		))
		metrics := NewClientMetrics("warnly.store-events", clOpts.reg, cfg.EnableKafkaHistograms)
		opts = append(opts, kgo.WithHooks(metrics))
	}
	if cfg.MetadataMaxAge > 0 {
		opts = append(opts, kgo.MetadataMaxAge(cfg.MetadataMaxAge))
	}
	if len(cfg.hooks) != 0 {
		opts = append(opts, kgo.WithHooks(cfg.hooks...))
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka: failed creating kafka client: %w", err)
	}

	// Issue a metadata refresh request on construction, so the broker list is populated.
	client.ForceMetadataRefresh()

	return client, nil
}

func (cfg *CommonConfig) namespacePrefix() string {
	if cfg.Namespace == "" {
		return ""
	}
	return cfg.Namespace + "-"
}

func (cfg *CommonConfig) tracerProvider() trace.TracerProvider {
	if cfg.TracerProvider != nil {
		return cfg.TracerProvider
	}
	return otel.GetTracerProvider()
}

// NewClientMetrics returns a new instance of `kprom.Metrics` (used to monitor Kafka interactions), provided
// the `MetricsPrefix` as the `Namespace` for the default set of Prometheus metrics.
func NewClientMetrics(component string, reg prometheus.Registerer, enableKafkaHistograms bool) *kprom.Metrics {
	return kprom.NewMetrics(MetricsPrefix,
		kprom.Registerer(WrapPrometheusRegisterer(component, reg)),
		// Do not export the client ID, because we use it to specify options to the backend.
		kprom.FetchAndProduceDetail(kprom.Batches, kprom.Records, kprom.CompressedBytes, kprom.UncompressedBytes), //  kprom.ByTopic?
		enableKafkaHistogramMetrics(enableKafkaHistograms),
	)
}

// WrapPrometheusRegisterer returns a prometheus.Registerer with labels applied
//
// This Registerer is used internally by the reader/writer Kafka clients to collect *kprom.Metrics (or any custom metrics
// passed by a calling service).
func WrapPrometheusRegisterer(component string, reg prometheus.Registerer) prometheus.Registerer {
	return prometheus.WrapRegistererWith(prometheus.Labels{
		"component": component,
	}, reg)
}

func enableKafkaHistogramMetrics(enable bool) kprom.Opt {
	histogramOpts := []kprom.HistogramOpts{}
	if enable {
		histogramOpts = append(histogramOpts,
			kprom.HistogramOpts{
				Enable:  kprom.ReadTime,
				Buckets: prometheus.DefBuckets,
			}, kprom.HistogramOpts{
				Enable:  kprom.ReadWait,
				Buckets: prometheus.DefBuckets,
			}, kprom.HistogramOpts{
				Enable:  kprom.WriteTime,
				Buckets: prometheus.DefBuckets,
			}, kprom.HistogramOpts{
				Enable:  kprom.WriteWait,
				Buckets: prometheus.DefBuckets,
			})
	}
	return kprom.HistogramsFromOpts(histogramOpts...)
}

//nolint:nilnil // nil is a valid value for configuration tls.
func BuildKafkaTLSConfig(cfg *KafkaConfig) (*tls.Config, error) {
	if !cfg.TLS.Enabled {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load kafka tls key pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	if cfg.TLS.CAFile != "" {
		caCert, err := os.ReadFile(cfg.TLS.CAFile)
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
func BuildKafkaSASLMechanism(cfg *KafkaConfig) (SASLMechanism, error) {
	if cfg.SASL.Plain.Enabled {
		return plain.Auth{
			User: cfg.SASL.Plain.User,
			Pass: cfg.SASL.Plain.Pass,
		}.AsMechanism(), nil
	}
	if cfg.SASL.SCRAM.Enabled {
		var auth scram.Auth
		auth.User = cfg.SASL.SCRAM.User
		auth.Pass = cfg.SASL.SCRAM.Pass
		switch cfg.SASL.SCRAM.Algorithm {
		case "SCRAM-SHA-256":
			return auth.AsSha256Mechanism(), nil
		case "SCRAM-SHA-512":
			return auth.AsSha512Mechanism(), nil
		default:
			return nil, fmt.Errorf("unsupported SCRAM algorithm: %s (use SCRAM-SHA-256 or SCRAM-SHA-512)", cfg.SASL.SCRAM.Algorithm)
		}
	}
	return nil, nil
}
