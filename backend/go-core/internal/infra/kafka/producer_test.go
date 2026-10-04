package kafka

import (
	"context"
	"testing"
)

func TestNoOpProducer(t *testing.T) {
	p := NewNoOpProducer()

	var prod Producer = p
	if err := prod.Publish(context.Background(), "test.topic", "key-1", map[string]string{"foo": "bar"}); err != nil {
		t.Fatalf("expected nil error from NoOpProducer.Publish, got: %v", err)
	}

	if err := prod.Close(); err != nil {
		t.Fatalf("expected nil error from NoOpProducer.Close, got: %v", err)
	}
}

func TestNewEventProducerWithBrokers(t *testing.T) {
	p := NewEventProducerWithBrokers("localhost:9092")
	if p == nil {
		t.Fatal("expected non-nil EventProducer")
	}
	defer p.Close()
}
