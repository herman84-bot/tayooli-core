package kafka

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
	kafkago "github.com/segmentio/kafka-go"
)

// Producer defines the interface for publishing events and closing the producer.
type Producer interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
	Close() error
}

// EventProducer publishes domain events to Kafka topics.
type EventProducer struct {
	writer *kafkago.Writer
}

// NewEventProducer creates a producer connected to KAFKA_BROKERS env var (default: localhost:9092).
func NewEventProducer() *EventProducer {
	return NewEventProducerWithBrokers(os.Getenv("KAFKA_BROKERS"))
}

// NewEventProducerWithBrokers creates a producer connected to the specified Kafka brokers.
func NewEventProducerWithBrokers(brokers string) *EventProducer {
	if brokers == "" {
		brokers = "localhost:9092"
		log.Warn().Str("default_broker", brokers).Msg("KAFKA_BROKERS not set, using default — set this in production")
	}
	return &EventProducer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers),
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireAll,
			Async:        false,
		},
	}
}

// Close flushes and closes the underlying writer.
func (p *EventProducer) Close() error {
	return p.writer.Close()
}

// Publish serialises payload as JSON and writes it to the given Kafka topic.
// key should be a stable entity identifier (e.g. invoice UUID or tenant UUID)
// so that related events are routed to the same partition, enabling ordered
// consumption per entity while distributing load across partitions.
func (p *EventProducer) Publish(ctx context.Context, topic string, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	})
	if err != nil {
		log.Error().Err(err).Str("topic", topic).Str("key", key).Msg("kafka publish failed")
		return err
	}
	return nil
}

// NoOpProducer is an event producer that safely drops messages without errors,
// used when running in standalone mode without Kafka.
type NoOpProducer struct{}

// NewNoOpProducer creates a new NoOpProducer.
func NewNoOpProducer() *NoOpProducer {
	return &NoOpProducer{}
}

// Publish implements Producer. It safely drops the message and returns nil.
func (n *NoOpProducer) Publish(ctx context.Context, topic string, key string, payload any) error {
	return nil
}

// Close implements Producer. It returns nil.
func (n *NoOpProducer) Close() error {
	return nil
}
