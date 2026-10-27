package main

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestConsumeStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deliveries := make(chan amqp.Delivery)
	done := make(chan struct{})

	go func() {
		consume(ctx, "test", deliveries, nil, nil, nil, func() {})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("consume blocked after context cancellation")
	}
}
