package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/vk-rv/warnly/internal/warnly"
)

type clientOptsFn func(opts *clientOpts)

type clientOpts struct {
	reg prometheus.Registerer
}

// CompressionCodec configures how records are compressed before being sent.
type CompressionCodec = kgo.CompressionCodec

// NoCompression is a compression option that avoid compression. This can
// alway be used as a fallback compression.
func NoCompression() CompressionCodec { return kgo.NoCompression() }

// GzipCompression enables gzip compression with the default compression level.
func GzipCompression() CompressionCodec { return kgo.GzipCompression() }

// SnappyCompression enables snappy compression.
func SnappyCompression() CompressionCodec { return kgo.SnappyCompression() }

// Lz4Compression enables lz4 compression with the fastest compression level.
func Lz4Compression() CompressionCodec { return kgo.Lz4Compression() }

// ZstdCompression enables zstd compression with the default compression level.
func ZstdCompression() CompressionCodec { return kgo.ZstdCompression() }

// BatchWriteListener specifies a callback function that is invoked after a batch is
// successfully produced to a Kafka broker. It is invoked with the corresponding topic and the
// amount of bytes written to that topic (taking compression into account, when applicable).
type BatchWriteListener func(topic string, bytesWritten int)

// OnProduceBatchWritten implements the kgo.HookProduceBatchWritten interface.
func (l BatchWriteListener) OnProduceBatchWritten(_ kgo.BrokerMetadata,
	topic string, _ int32, m kgo.ProduceBatchMetrics,
) {
	l(topic, m.CompressedBytes)
}

// ProducerConfig holds configuration for publishing events to Kafka.
//
// Defaults follow Franz-go library recommendations for high write throughput.
//
//nolint:govet // we want to align the struct fields
type ProducerConfig struct {
	CommonConfig

	RecordPartitioner      kgo.Partitioner
	Reg                    prometheus.Registerer
	ProduceCallback        func(*kgo.Record, error)
	BatchListener          BatchWriteListener
	CompressionCodec       []CompressionCodec
	MaxBufferedRecords     int
	ProducerBatchMaxBytes  int32
	ManualFlushing         bool
	Sync                   bool
	AllowAutoTopicCreation bool
}

// finalize ensures the configuration is valid, setting default values from
// environment variables and Franz-go recommendations as described in doc comments,
// returning an error if any configuration is invalid.
func (cfg *ProducerConfig) finalize() error {
	cfg.CommonConfig.finalize()

	// Apply recommended defaults for high-throughput producer
	if cfg.MaxBufferedRecords == 0 {
		cfg.MaxBufferedRecords = 1_000_000
	}
	if cfg.ProducerBatchMaxBytes == 0 {
		cfg.ProducerBatchMaxBytes = 16_000_000
	}
	if cfg.MetadataMaxAge == 0 {
		cfg.MetadataMaxAge = 60 * time.Second
	}
	if cfg.RecordPartitioner == nil {
		cfg.RecordPartitioner = kgo.UniformBytesPartitioner(1_000_000, false, false, nil)
	}

	var errs []error
	if cfg.MaxBufferedRecords < 0 {
		errs = append(errs, fmt.Errorf("kafka: max buffered records cannot be negative: %d", cfg.MaxBufferedRecords))
	}
	if cfg.ProducerBatchMaxBytes < 0 {
		errs = append(errs, fmt.Errorf("kafka: producer batch max bytes cannot be negative: %d", cfg.ProducerBatchMaxBytes))
	}
	return errors.Join(errs...)
}

// Producer publishes events to Kafka. Implements the Producer interface.
type Producer struct {
	cfg    *ProducerConfig
	client *kgo.Client
	mu     sync.RWMutex
}

// NewProducer returns a new Producer with the given config.
func NewProducer(cfg *ProducerConfig) (*Producer, error) {
	if err := cfg.finalize(); err != nil {
		return nil, fmt.Errorf("kafka: invalid producer config: %w", err)
	}
	var opts []kgo.Opt
	if len(cfg.CompressionCodec) > 0 {
		opts = append(opts, kgo.ProducerBatchCompression(cfg.CompressionCodec...))
	}
	if cfg.MaxBufferedRecords != 0 {
		opts = append(opts, kgo.MaxBufferedRecords(cfg.MaxBufferedRecords))
	}
	if cfg.ProducerBatchMaxBytes != 0 {
		opts = append(opts, kgo.ProducerBatchMaxBytes(cfg.ProducerBatchMaxBytes))
	}
	if cfg.ManualFlushing {
		opts = append(opts, kgo.ManualFlushing())
	}
	if cfg.BatchListener != nil {
		opts = append(opts, kgo.WithHooks(cfg.BatchListener))
	}
	if cfg.RecordPartitioner != nil {
		opts = append(opts, kgo.RecordPartitioner(cfg.RecordPartitioner))
	}
	if cfg.AllowAutoTopicCreation {
		opts = append(opts, kgo.AllowAutoTopicCreation())
	}
	client, err := cfg.newClientWithOpts(
		[]clientOptsFn{
			func(opts *clientOpts) {
				opts.reg = cfg.Reg
			},
		},
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: failed creating producer: %w", err)
	}

	return &Producer{
		cfg:    cfg,
		client: client,
	}, nil
}

// Close stops the producer
//
// This call is blocking and will cause the underlying client to stop
// producing. If producing is asynchronous, it'll block until all messages
// have been produced. After Close() is called, Producer cannot be reused.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.client.Flush(context.Background()); err != nil {
		return fmt.Errorf("cannot flush on close: %w", err)
	}
	p.client.Close()
	return nil
}

// Healthy returns an error if the Kafka client fails to reach a discovered
// broker.
func (p *Producer) Healthy(ctx context.Context) error {
	if err := p.client.Ping(ctx); err != nil {
		return fmt.Errorf("health probe: %w", err)
	}
	return nil
}

// Produce produces N records. If the Producer is synchronous, waits until
// all records are produced, otherwise, returns as soon as the records are
// stored in the producer buffer, or when the records are produced to the
// queue if sync producing is configured.
// If the context has been enriched with metadata, each entry will be added
// as a record's header.
// Produce takes ownership of Record and any modifications after Produce is
// called may cause an unhandled exception.
func (p *Producer) Produce(ctx context.Context, rs ...warnly.Record) error {
	if len(rs) == 0 {
		return nil
	}

	// Take a read lock to prevent Close from closing the client
	// while we're attempting to produce records.
	p.mu.RLock()
	defer p.mu.RUnlock()

	var headers []kgo.RecordHeader
	if m, ok := MetadataFromContext(ctx); ok {
		headers = make([]kgo.RecordHeader, 0, len(m))
		for k, v := range m {
			headers = append(headers, kgo.RecordHeader{
				Key: k, Value: []byte(v),
			})
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(rs))
	if !p.cfg.Sync {
		ctx = DetachedContext(ctx)
	}
	namespacePrefix := p.cfg.namespacePrefix()

	var errs []error
	var mu sync.Mutex

	for i := range rs {
		kgoRecord := &kgo.Record{
			Headers: headers,
			Topic:   fmt.Sprintf("%s%s", namespacePrefix, rs[i].Topic),
			Key:     rs[i].OrderingKey,
			Value:   rs[i].Value,
		}
		p.client.Produce(ctx, kgoRecord, func(r *kgo.Record, err error) {
			defer wg.Done()
			// kotel already marks spans as errors. No need to handle it here.
			if err != nil {
				topicName := strings.TrimPrefix(r.Topic, namespacePrefix)

				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to produce message: %w", err))
				mu.Unlock()

				p.cfg.Logger.Error("failed producing message",
					slog.Any("error", err),
					slog.String("topic", topicName),
					slog.Int64("offset", r.Offset),
					slog.Int("partition", int(r.Partition)),
					slog.Any("headers", headers),
				)
			}
			if p.cfg.ProduceCallback != nil {
				p.cfg.ProduceCallback(r, err)
			}
		})
	}
	if p.cfg.Sync {
		wg.Wait()
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}
	return nil
}
