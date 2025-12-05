package warnly

import (
	"context"
	"errors"
)

const (
	QueueTopic = "warnly.queue"
)

// ErrConsumerAlreadyRunning is returned by consumer.Run if it has already
// been called.
var ErrConsumerAlreadyRunning = errors.New("consumer.Run: consumer already running")

// DeliveryType for the consumer. For more details See the supported DeliveryTypes.
type DeliveryType uint8

const (
	// AtMostOnceDeliveryType acknowledges the message as soon as it's received
	// and decoded, without waiting for the message to be processed.
	AtMostOnceDeliveryType DeliveryType = iota
	// AtLeastOnceDeliveryType acknowledges the message after it has been
	// processed. It may or may not create duplicates, depending on how batches
	// are processed by the underlying Processor.
	AtLeastOnceDeliveryType
)

// Consumer wraps the implementation details of the consumer implementation.
// Consumer implementations must support the defined delivery types.
type Consumer interface {
	// Run executes the consumer in a blocking manner. Returns
	// ErrConsumerAlreadyRunning when it has already been called.
	Run(ctx context.Context) error
	// Healthy returns an error if the consumer isn't healthy.
	Healthy(ctx context.Context) error
	// Close closes the consumer.
	Close() error
}

// Topic represents a destination topic where to produce a message/record.
type Topic string

// Producer wraps the producer implementation details. Producer implementations
// must support sync and async production.
type Producer interface {
	// Produce produces N records. If the Producer is synchronous, waits until
	// all records are produced, otherwise, returns as soon as the records are
	// stored in the producer buffer, or when the records are produced to the
	// queue if sync producing is configured.
	// If the context has been enriched with metadata, each entry will be added
	// as a record's header.
	// Produce takes ownership of Record and any modifications after Produce is
	// called may cause an unhandled exception.
	Produce(ctx context.Context, rs ...Record) error
	// Healthy returns an error if the producer isn't healthy.
	Healthy(ctx context.Context) error
	// Close closes the producer.
	Close() error
}

// Record wraps a record's value with the topic where it's produced / consumed.
type Record struct {
	// Topics holds the topic where the record will be produced.
	Topic Topic
	// OrderingKey is an optional field that is hashed to map to a partition.
	// Records with same ordering key are routed to the same partition.
	OrderingKey []byte
	// Value holds the record's content. It must not be mutated after Produce.
	Value []byte
	// Partition identifies the partition ID where the record was polled from.
	// It is optional and only used for consumers.
	// When not specified, the zero value for int32 (0) identifies the only partition.
	Partition int32
}

// Processor defines record processing signature.
type Processor interface {
	// Process processes one or more records within the passed context.
	// Process takes ownership of the passed records, callers must not mutate
	// a record after Process has been called.
	Process(ctx context.Context, record Record) error
}

// ProcessorFunc is a function type that implements the Processor interface.
type ProcessorFunc func(context.Context, Record) error

// Process returns f(ctx, records...).
func (f ProcessorFunc) Process(ctx context.Context, rs Record) error {
	return f(ctx, rs)
}

// TopicConsumer is used to monitor a set of consumer topics.
type TopicConsumer struct {
	// Optional topic to monitor.
	Topic Topic
	// Optional regex expression to match topics for monitoring.
	Regex string
	// Required consumer name.
	Consumer string
}
