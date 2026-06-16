//go:build integration

package order

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	orderEntity := &Order{
		ID:         uuid.New(),
		CustomerID: "integration",
		Status:     OrderStatusPending,
		Currency:   "USD",
		Items: []OrderItem{
			{ID: uuid.New(), ProductID: "tea", ProductName: "Tea", Quantity: 1, UnitPrice: decimal.RequireFromString("2.50")},
		},
	}
	orderEntity.RecalculateTotals()

	if err := store.Create(ctx, orderEntity); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.Get(ctx, orderEntity.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != orderEntity.ID || got.TotalAmount.StringFixed(2) != "2.50" {
		t.Fatalf("unexpected round-trip: %#v", got)
	}
}
