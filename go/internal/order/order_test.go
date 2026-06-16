package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wilddog64/shopping-cart-order/internal/events"
	"go.uber.org/zap"
)

type memoryStore struct {
	mu     sync.Mutex
	orders map[uuid.UUID]*Order
}

func newMemoryStore() *memoryStore {
	return &memoryStore{orders: make(map[uuid.UUID]*Order)}
}

func (s *memoryStore) Create(_ context.Context, order *Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ID] = cloneOrder(order)
	return nil
}

func (s *memoryStore) Get(_ context.Context, id uuid.UUID) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return cloneOrder(order), nil
}

func (s *memoryStore) ListByCustomer(_ context.Context, customerID string) ([]*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var orders []*Order
	for _, order := range s.orders {
		if order.CustomerID == customerID {
			orders = append(orders, cloneOrder(order))
		}
	}
	sort.Slice(orders, func(i, j int) bool {
		if orders[i].CreatedAt.Equal(orders[j].CreatedAt) {
			return orders[i].ID.String() < orders[j].ID.String()
		}
		return orders[i].CreatedAt.Before(orders[j].CreatedAt)
	})
	return orders, nil
}

func (s *memoryStore) Update(_ context.Context, order *Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[order.ID]; !ok {
		return ErrOrderNotFound
	}
	s.orders[order.ID] = cloneOrder(order)
	return nil
}

func (s *memoryStore) Close() {}

func (s *memoryStore) Ping(context.Context) error { return nil }

type recordingPublisher struct {
	mu        sync.Mutex
	envelopes []events.Envelope
}

func (p *recordingPublisher) Publish(_ context.Context, envelope events.Envelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.envelopes = append(p.envelopes, envelope)
	return nil
}

func (p *recordingPublisher) Close() error { return nil }

func cloneOrder(order *Order) *Order {
	if order == nil {
		return nil
	}
	cloned := *order
	cloned.Items = make([]OrderItem, len(order.Items))
	copy(cloned.Items, order.Items)
	if order.ShippingAddress != nil {
		addr := *order.ShippingAddress
		cloned.ShippingAddress = &addr
	}
	cloned.TrackingNumber = cloneString(order.TrackingNumber)
	cloned.Carrier = cloneString(order.Carrier)
	cloned.PaidAt = cloneTime(order.PaidAt)
	cloned.ShippedAt = cloneTime(order.ShippedAt)
	cloned.CompletedAt = cloneTime(order.CompletedAt)
	cloned.CancelledAt = cloneTime(order.CancelledAt)
	cloned.CancellationReason = cloneString(order.CancellationReason)
	return &cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func TestOrderStatusTransitions(t *testing.T) {
	cases := []struct {
		name    string
		from    OrderStatus
		to      OrderStatus
		allowed bool
	}{
		{"pending to paid", OrderStatusPending, OrderStatusPaid, true},
		{"pending to cancelled", OrderStatusPending, OrderStatusCancelled, true},
		{"pending to shipped", OrderStatusPending, OrderStatusShipped, false},
		{"paid to processing", OrderStatusPaid, OrderStatusProcessing, true},
		{"paid to cancelled", OrderStatusPaid, OrderStatusCancelled, true},
		{"processing to shipped", OrderStatusProcessing, OrderStatusShipped, true},
		{"processing to completed", OrderStatusProcessing, OrderStatusCompleted, false},
		{"shipped to completed", OrderStatusShipped, OrderStatusCompleted, true},
		{"completed to cancelled", OrderStatusCompleted, OrderStatusCancelled, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.from.CanTransitionTo(tc.to)
			if got != tc.allowed {
				t.Fatalf("transition %s -> %s = %v, want %v", tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestOrderTotalsAndDateOnlyJSON(t *testing.T) {
	order := NewOrder("cust-1", "USD")
	order.AddItem(OrderItem{ProductID: "p1", ProductName: "Tea", Quantity: 2, UnitPrice: decimal.RequireFromString("3.50")})
	order.AddItem(OrderItem{ProductID: "p2", ProductName: "Coffee", Quantity: 1, UnitPrice: decimal.RequireFromString("9.99")})

	if got := order.TotalAmount.StringFixed(2); got != "16.99" {
		t.Fatalf("total amount = %s, want 16.99", got)
	}

	dateOnly := NewDateOnly(time.Date(2026, 6, 15, 12, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60)))
	payload, err := json.Marshal(dateOnly)
	if err != nil {
		t.Fatalf("marshal dateonly: %v", err)
	}
	if got := string(payload); got != `"2026-06-15"` {
		t.Fatalf("dateonly json = %s, want \"2026-06-15\"", got)
	}

	var decoded DateOnly
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal dateonly: %v", err)
	}
	if got := decoded.Time.UTC().Format("2006-01-02"); got != "2026-06-15" {
		t.Fatalf("dateonly decoded = %s, want 2026-06-15", got)
	}
}

func TestServiceCreateUpdateAndCancel(t *testing.T) {
	store := newMemoryStore()
	publisher := &recordingPublisher{}
	service := NewService(store, publisher, zaptestLogger(t))

	created, err := service.CreateOrder(context.Background(), CreateOrderRequest{
		CustomerID: "cust-123",
		Items: []CreateOrderItemRequest{
			{ProductID: "tea", ProductName: "Tea", Quantity: 2, UnitPrice: decimal.RequireFromString("3.25")},
			{ProductID: "coffee", ProductName: "Coffee", Quantity: 1, UnitPrice: decimal.RequireFromString("4.50")},
		},
		ShippingAddress: &CreateOrderAddressRequest{
			Street:     "1 Main",
			City:       "Portland",
			State:      "OR",
			PostalCode: "97201",
			Country:    "US",
		},
	}, "corr-1")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if got := created.TotalAmount.StringFixed(2); got != "11.00" {
		t.Fatalf("created total = %s, want 11.00", got)
	}
	if len(publisher.envelopes) != 1 || publisher.envelopes[0].Type != events.OrderCreatedType {
		t.Fatalf("unexpected created envelopes: %#v", publisher.envelopes)
	}

	paid, err := service.UpdateOrderStatus(context.Background(), created.ID, UpdateOrderStatusRequest{
		Status:        OrderStatusPaid,
		PaymentID:     "pay-1",
		PaymentMethod: "CARD",
	}, "corr-2")
	if err != nil {
		t.Fatalf("paid transition: %v", err)
	}
	if paid.Status != OrderStatusPaid || paid.PaidAt == nil {
		t.Fatalf("paid order not updated correctly: %#v", paid)
	}
	if got := publisher.envelopes[len(publisher.envelopes)-1].Type; got != events.OrderPaidType {
		t.Fatalf("last event = %s, want %s", got, events.OrderPaidType)
	}

	processing, err := service.UpdateOrderStatus(context.Background(), created.ID, UpdateOrderStatusRequest{
		Status: OrderStatusProcessing,
	}, "corr-3")
	if err != nil {
		t.Fatalf("processing transition: %v", err)
	}
	if processing.Status != OrderStatusProcessing {
		t.Fatalf("processing order not updated correctly: %#v", processing)
	}

	shipped, err := service.UpdateOrderStatus(context.Background(), created.ID, UpdateOrderStatusRequest{
		Status:         OrderStatusShipped,
		TrackingNumber: "TRK-1",
		Carrier:        "UPS",
	}, "corr-4")
	if err != nil {
		t.Fatalf("shipped transition: %v", err)
	}
	if shipped.Status != OrderStatusShipped || shipped.ShippedAt == nil || shipped.TrackingNumber == nil || shipped.Carrier == nil {
		t.Fatalf("shipped order not updated correctly: %#v", shipped)
	}
	if got := publisher.envelopes[len(publisher.envelopes)-1].Type; got != events.OrderShippedType {
		t.Fatalf("last event = %s, want %s", got, events.OrderShippedType)
	}

	completed, err := service.UpdateOrderStatus(context.Background(), created.ID, UpdateOrderStatusRequest{
		Status: OrderStatusCompleted,
	}, "corr-5")
	if err != nil {
		t.Fatalf("completed transition: %v", err)
	}
	if completed.Status != OrderStatusCompleted || completed.CompletedAt == nil {
		t.Fatalf("completed order not updated correctly: %#v", completed)
	}
	if got := publisher.envelopes[len(publisher.envelopes)-1].Type; got != events.OrderCompletedType {
		t.Fatalf("last event = %s, want %s", got, events.OrderCompletedType)
	}

	canceledStore := newMemoryStore()
	canceledService := NewService(canceledStore, publisher, zaptestLogger(t))
	canceledOrder, err := canceledService.CreateOrder(context.Background(), CreateOrderRequest{
		CustomerID: "cust-999",
		Items:      []CreateOrderItemRequest{{ProductID: "tea", ProductName: "Tea", Quantity: 1, UnitPrice: decimal.RequireFromString("3.25")}},
	}, "corr-4")
	if err != nil {
		t.Fatalf("create cancel target: %v", err)
	}
	cancelled, err := canceledService.CancelOrder(context.Background(), canceledOrder.ID, "changed mind", "", "corr-5")
	if err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	if cancelled.Status != OrderStatusCancelled || cancelled.CancellationReason == nil {
		t.Fatalf("cancelled order not updated correctly: %#v", cancelled)
	}
	if got := publisher.envelopes[len(publisher.envelopes)-1].Type; got != events.OrderCancelledType {
		t.Fatalf("last event = %s, want %s", got, events.OrderCancelledType)
	}

	cancelTarget := cloneOrder(completed)
	cancelTarget.Status = OrderStatusCompleted
	canceledStore.mu.Lock()
	canceledStore.orders[completed.ID] = cancelTarget
	canceledStore.mu.Unlock()

	_, err = canceledService.CancelOrder(context.Background(), completed.ID, "nope", "system", "corr-6")
	if err == nil || !strings.Contains(err.Error(), "Cannot cancel order in status: COMPLETED") {
		t.Fatalf("expected cancel not allowed error, got %v", err)
	}
}

func TestServiceRejectsInvalidTransitions(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, &recordingPublisher{}, zaptestLogger(t))

	orderEntity, err := service.CreateOrder(context.Background(), CreateOrderRequest{
		CustomerID: "cust-1",
		Items:      []CreateOrderItemRequest{{ProductID: "tea", ProductName: "Tea", Quantity: 1, UnitPrice: decimal.RequireFromString("1.00")}},
	}, "corr")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	_, err = service.UpdateOrderStatus(context.Background(), orderEntity.ID, UpdateOrderStatusRequest{
		Status: OrderStatusShipped,
	}, "corr")
	if err == nil {
		t.Fatalf("expected invalid transition error")
	}
	var invalidTransition *InvalidTransitionError
	if !errors.As(err, &invalidTransition) {
		t.Fatalf("expected InvalidTransitionError, got %T", err)
	}
}

func TestHandleServiceErrorMapsResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", ErrOrderNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"invalid transition", &InvalidTransitionError{From: OrderStatusPending, To: OrderStatusShipped}, http.StatusBadRequest, "INVALID_STATE"},
		{"cancel not allowed", &CancelNotAllowedError{Status: OrderStatusShipped}, http.StatusBadRequest, "INVALID_STATE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			handleServiceError(ctx, tc.err)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			if body["code"] != tc.wantCode {
				t.Fatalf("code = %s, want %s", body["code"], tc.wantCode)
			}
		})
	}
}

func TestToOrderResponse(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	orderEntity := &Order{
		ID:          uuid.New(),
		CustomerID:  "cust",
		Status:      OrderStatusProcessing,
		TotalAmount: decimal.RequireFromString("5.50"),
		Currency:    "USD",
		ShippingAddress: &ShippingAddress{
			Street:     "1 Main",
			City:       "Portland",
			State:      "OR",
			PostalCode: "97201",
			Country:    "US",
		},
		TrackingNumber:     stringPtr("TRK"),
		Carrier:            stringPtr("UPS"),
		CreatedAt:          now,
		UpdatedAt:          now,
		PaidAt:             &now,
		ShippedAt:          &now,
		CompletedAt:        &now,
		CancelledAt:        &now,
		CancellationReason: stringPtr("cancelled"),
		Items: []OrderItem{
			{ID: uuid.New(), ProductID: "tea", ProductName: "Tea", Quantity: 1, UnitPrice: decimal.RequireFromString("5.50"), Subtotal: decimal.RequireFromString("5.50")},
		},
	}

	response := toOrderResponse(orderEntity)
	if response.CustomerID != "cust" || response.Status != OrderStatusProcessing || response.ShippingAddress == nil {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response.Items[0].Subtotal.StringFixed(2) != "5.50" {
		t.Fatalf("unexpected subtotal: %#v", response.Items[0])
	}
}

func zaptestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}
