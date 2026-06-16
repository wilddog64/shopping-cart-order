package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestEnvelopeAndEventPayloads(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	created := NewOrderCreatedEvent(
		"11111111-1111-1111-1111-111111111111",
		"cust-123",
		[]OrderItem{
			{
				ProductID:   "tea",
				ProductName: "Tea",
				Quantity:    2,
				UnitPrice:   decimal.RequireFromString("9.99"),
			},
		},
		decimal.RequireFromString("19.99"),
		"USD",
		&Address{
			Street:     "1 Main",
			City:       "Portland",
			State:      "OR",
			PostalCode: "97201",
			Country:    "US",
		},
	)

	payload, err := json.Marshal(created.ToEnvelope("corr-1"))
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if got["source"] != SourceName {
		t.Fatalf("source = %v, want %s", got["source"], SourceName)
	}
	if got["correlationId"] != "corr-1" {
		t.Fatalf("correlationId = %v, want corr-1", got["correlationId"])
	}

	data := got["data"].(map[string]any)
	if _, ok := data["totalAmount"].(float64); !ok {
		t.Fatalf("order.created totalAmount type = %T, want number", data["totalAmount"])
	}
	if _, ok := data["shippingAddress"].(map[string]any); !ok {
		t.Fatalf("order.created shippingAddress type = %T, want object", data["shippingAddress"])
	}

	paidPayload, err := json.Marshal(NewOrderPaidEvent(
		"11111111-1111-1111-1111-111111111111",
		"pay-1",
		decimal.RequireFromString("19.99"),
		"USD",
		"CARD",
		now,
	))
	if err != nil {
		t.Fatalf("marshal paid event: %v", err)
	}
	if !jsonContainsKey(paidPayload, "amount") || jsonContainsKey(paidPayload, "totalAmount") {
		t.Fatalf("paid event json = %s, want amount and no totalAmount", string(paidPayload))
	}

	shipEvent := NewOrderShippedEvent(
		"11111111-1111-1111-1111-111111111111",
		"TRK-1",
		"UPS",
		now,
		NewDateOnly(now.AddDate(0, 0, 5)),
	)
	shipPayload, err := json.Marshal(shipEvent)
	if err != nil {
		t.Fatalf("marshal shipped event: %v", err)
	}
	if !jsonContainsKey(shipPayload, "estimatedDelivery") {
		t.Fatalf("shipped event json = %s, want estimatedDelivery", string(shipPayload))
	}

	cancelledPayload, err := json.Marshal(NewOrderCancelledEvent(
		"11111111-1111-1111-1111-111111111111",
		"changed mind",
		"system",
		now,
		true,
	))
	if err != nil {
		t.Fatalf("marshal cancelled event: %v", err)
	}
	if !jsonContainsKey(cancelledPayload, "refundInitiated") {
		t.Fatalf("cancelled event json = %s, want refundInitiated", string(cancelledPayload))
	}
}

func TestEnvelopeGeneratesCorrelationID(t *testing.T) {
	envelope := NewEnvelope(OrderCompletedType, OrderCompletedVersion, OrderCompletedEvent{OrderID: uuid.NewString()}, "")
	if envelope.CorrelationID == "" {
		t.Fatal("expected generated correlation id")
	}
}

func jsonContainsKey(payload []byte, key string) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return false
	}
	_, ok := raw[key]
	return ok
}
