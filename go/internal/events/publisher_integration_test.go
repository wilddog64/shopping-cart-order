//go:build integration

package events

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestRabbitPublisherPublishesOrderCreated(t *testing.T) {
	uri := os.Getenv("TEST_RABBITMQ_URI")
	if uri == "" {
		t.Skip("TEST_RABBITMQ_URI not set")
	}

	publisher := NewRabbitPublisher(uri, zap.NewNop())
	defer publisher.Close()

	if err := publisher.Publish(context.Background(), NewOrderCreatedEvent(
		uuid.NewString(),
		"integration",
		[]OrderItem{{ProductID: "tea", ProductName: "Tea", Quantity: 1, UnitPrice: decimal.RequireFromString("1.23")}},
		decimal.RequireFromString("1.23"),
		"USD",
		nil,
	).ToEnvelope("")); err != nil {
		t.Fatalf("publish: %v", err)
	}
}
