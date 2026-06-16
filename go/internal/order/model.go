package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusPaid       OrderStatus = "PAID"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusShipped    OrderStatus = "SHIPPED"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

func ParseOrderStatus(value string) (OrderStatus, error) {
	status := OrderStatus(strings.ToUpper(strings.TrimSpace(value)))
	if !status.IsValid() {
		return "", fmt.Errorf("invalid order status: %s", value)
	}
	return status, nil
}

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusPaid, OrderStatusProcessing, OrderStatusShipped, OrderStatusCompleted, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func (s OrderStatus) CanTransitionTo(next OrderStatus) bool {
	switch s {
	case OrderStatusPending:
		return next == OrderStatusPaid || next == OrderStatusCancelled
	case OrderStatusPaid:
		return next == OrderStatusProcessing || next == OrderStatusCancelled
	case OrderStatusProcessing:
		return next == OrderStatusShipped || next == OrderStatusCancelled
	case OrderStatusShipped:
		return next == OrderStatusCompleted
	case OrderStatusCompleted, OrderStatusCancelled:
		return false
	default:
		return false
	}
}

type DateOnly struct {
	time.Time
}

func NewDateOnly(t time.Time) DateOnly {
	return DateOnly{Time: t.UTC()}
}

func ParseDateOnly(value string) (DateOnly, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return DateOnly{}, err
	}
	return DateOnly{Time: parsed.UTC()}, nil
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time.UTC().Format("2006-01-02"))
}

func (d *DateOnly) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.Time = time.Time{}
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value == "" {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return err
	}
	d.Time = parsed.UTC()
	return nil
}

type ShippingAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type OrderItem struct {
	ID          uuid.UUID       `json:"id"`
	OrderID     uuid.UUID       `json:"-"`
	ProductID   string          `json:"productId"`
	ProductName string          `json:"productName"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

func (i *OrderItem) RecalculateSubtotal() {
	i.Subtotal = i.UnitPrice.Mul(decimal.NewFromInt(int64(i.Quantity))).Round(2)
}

type Order struct {
	ID                 uuid.UUID        `json:"id"`
	CustomerID         string           `json:"customerId"`
	Status             OrderStatus      `json:"status"`
	Items              []OrderItem      `json:"items"`
	TotalAmount        decimal.Decimal  `json:"totalAmount"`
	Currency           string           `json:"currency"`
	ShippingAddress    *ShippingAddress `json:"shippingAddress"`
	TrackingNumber     *string          `json:"trackingNumber"`
	Carrier            *string          `json:"carrier"`
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
	PaidAt             *time.Time       `json:"paidAt"`
	ShippedAt          *time.Time       `json:"shippedAt"`
	CompletedAt        *time.Time       `json:"completedAt"`
	CancelledAt        *time.Time       `json:"cancelledAt"`
	CancellationReason *string          `json:"cancellationReason"`
}

func NewOrder(customerID, currency string) *Order {
	now := time.Now().UTC()
	if currency == "" {
		currency = "USD"
	}
	return &Order{
		ID:          uuid.New(),
		CustomerID:  customerID,
		Status:      OrderStatusPending,
		Items:       []OrderItem{},
		TotalAmount: decimal.Zero,
		Currency:    currency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (o *Order) RecalculateTotals() {
	total := decimal.Zero
	for i := range o.Items {
		o.Items[i].RecalculateSubtotal()
		total = total.Add(o.Items[i].Subtotal)
	}
	o.TotalAmount = total.Round(2)
}

func (o *Order) AddItem(item OrderItem) {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	item.OrderID = o.ID
	item.RecalculateSubtotal()
	o.Items = append(o.Items, item)
	o.RecalculateTotals()
}

func (o *Order) UpdateTimestampsOnPersist() {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	o.UpdatedAt = time.Now().UTC()
}

func StringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func TimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	t := value.UTC()
	return &t
}

func DecimalFromString(value string) (decimal.Decimal, error) {
	if strings.TrimSpace(value) == "" {
		return decimal.Zero, errors.New("empty decimal")
	}
	return decimal.NewFromString(value)
}
