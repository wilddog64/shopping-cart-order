package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

const (
	ExchangeName = "events"
	SourceName   = "order-service"
	VersionV1    = "1.0"
)

type Envelope struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"`
	Version       string      `json:"version"`
	Timestamp     time.Time   `json:"timestamp"`
	Source        string      `json:"source"`
	CorrelationID string      `json:"correlationId"`
	Data          interface{} `json:"data"`
}

func NewEnvelope(eventType, version string, data interface{}, correlationID string) Envelope {
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	return Envelope{
		ID:            uuid.NewString(),
		Type:          eventType,
		Version:       version,
		Timestamp:     time.Now().UTC(),
		Source:        SourceName,
		CorrelationID: correlationID,
		Data:          data,
	}
}

type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type OrderItem struct {
	ProductID   string          `json:"productId"`
	ProductName string          `json:"productName"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
}

type DateOnly struct {
	time.Time
}

func NewDateOnly(t time.Time) DateOnly {
	return DateOnly{Time: t.UTC()}
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
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

type OrderCreatedEvent struct {
	OrderID         string          `json:"orderId"`
	CustomerID      string          `json:"customerId"`
	Items           []OrderItem     `json:"items"`
	TotalAmount     decimal.Decimal `json:"totalAmount"`
	Currency        string          `json:"currency"`
	ShippingAddress *Address        `json:"shippingAddress"`
}

const (
	OrderCreatedType    = "order.created"
	OrderCreatedVersion = VersionV1
)

func NewOrderCreatedEvent(orderID, customerID string, items []OrderItem, totalAmount decimal.Decimal, currency string, shippingAddress *Address) OrderCreatedEvent {
	return OrderCreatedEvent{
		OrderID:         orderID,
		CustomerID:      customerID,
		Items:           items,
		TotalAmount:     totalAmount,
		Currency:        currency,
		ShippingAddress: shippingAddress,
	}
}

func (e OrderCreatedEvent) ToEnvelope(correlationID string) Envelope {
	return NewEnvelope(OrderCreatedType, OrderCreatedVersion, e, correlationID)
}

type OrderPaidEvent struct {
	OrderID       string          `json:"orderId"`
	PaymentID     string          `json:"paymentId"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	PaymentMethod string          `json:"paymentMethod"`
	PaidAt        time.Time       `json:"paidAt"`
}

const (
	OrderPaidType    = "order.paid"
	OrderPaidVersion = VersionV1
)

func NewOrderPaidEvent(orderID, paymentID string, amount decimal.Decimal, currency, paymentMethod string, paidAt time.Time) OrderPaidEvent {
	if paidAt.IsZero() {
		paidAt = time.Now().UTC()
	}
	return OrderPaidEvent{
		OrderID:       orderID,
		PaymentID:     paymentID,
		Amount:        amount,
		Currency:      currency,
		PaymentMethod: paymentMethod,
		PaidAt:        paidAt.UTC(),
	}
}

func (e OrderPaidEvent) ToEnvelope(correlationID string) Envelope {
	return NewEnvelope(OrderPaidType, OrderPaidVersion, e, correlationID)
}

type OrderShippedEvent struct {
	OrderID           string    `json:"orderId"`
	TrackingNumber    string    `json:"trackingNumber"`
	Carrier           string    `json:"carrier"`
	ShippedAt         time.Time `json:"shippedAt"`
	EstimatedDelivery DateOnly  `json:"estimatedDelivery"`
}

const (
	OrderShippedType    = "order.shipped"
	OrderShippedVersion = VersionV1
)

func NewOrderShippedEvent(orderID, trackingNumber, carrier string, shippedAt time.Time, estimatedDelivery DateOnly) OrderShippedEvent {
	if shippedAt.IsZero() {
		shippedAt = time.Now().UTC()
	}
	return OrderShippedEvent{
		OrderID:           orderID,
		TrackingNumber:    trackingNumber,
		Carrier:           carrier,
		ShippedAt:         shippedAt.UTC(),
		EstimatedDelivery: estimatedDelivery,
	}
}

func (e OrderShippedEvent) ToEnvelope(correlationID string) Envelope {
	return NewEnvelope(OrderShippedType, OrderShippedVersion, e, correlationID)
}

type OrderCompletedEvent struct {
	OrderID     string    `json:"orderId"`
	CompletedAt time.Time `json:"completedAt"`
}

const (
	OrderCompletedType    = "order.completed"
	OrderCompletedVersion = VersionV1
)

func NewOrderCompletedEvent(orderID string, completedAt time.Time) OrderCompletedEvent {
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	return OrderCompletedEvent{
		OrderID:     orderID,
		CompletedAt: completedAt.UTC(),
	}
}

func (e OrderCompletedEvent) ToEnvelope(correlationID string) Envelope {
	return NewEnvelope(OrderCompletedType, OrderCompletedVersion, e, correlationID)
}

type OrderCancelledEvent struct {
	OrderID         string    `json:"orderId"`
	Reason          string    `json:"reason"`
	CancelledBy     string    `json:"cancelledBy"`
	CancelledAt     time.Time `json:"cancelledAt"`
	RefundInitiated bool      `json:"refundInitiated"`
}

const (
	OrderCancelledType    = "order.cancelled"
	OrderCancelledVersion = VersionV1
)

func NewOrderCancelledEvent(orderID, reason, cancelledBy string, cancelledAt time.Time, refundInitiated bool) OrderCancelledEvent {
	if cancelledAt.IsZero() {
		cancelledAt = time.Now().UTC()
	}
	return OrderCancelledEvent{
		OrderID:         orderID,
		Reason:          reason,
		CancelledBy:     cancelledBy,
		CancelledAt:     cancelledAt.UTC(),
		RefundInitiated: refundInitiated,
	}
}

func (e OrderCancelledEvent) ToEnvelope(correlationID string) Envelope {
	return NewEnvelope(OrderCancelledType, OrderCancelledVersion, e, correlationID)
}

type Publisher interface {
	Publish(ctx context.Context, envelope Envelope) error
	Close() error
}

type RabbitPublisher struct {
	uri      string
	logger   *zap.Logger
	mu       sync.Mutex
	pubMu    sync.Mutex
	conn     *amqp.Connection
	channel  *amqp.Channel
	declared bool
}

func NewRabbitPublisher(uri string, logger *zap.Logger) *RabbitPublisher {
	return &RabbitPublisher{uri: uri, logger: logger}
}

func (p *RabbitPublisher) Publish(ctx context.Context, envelope Envelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	channel, err := p.ensureChannel(ctx)
	if err != nil {
		return err
	}

	p.pubMu.Lock()
	pubErr := channel.PublishWithContext(ctx, ExchangeName, envelope.Type, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})
	p.pubMu.Unlock()

	if pubErr != nil {
		_ = p.reset()
		return fmt.Errorf("publish %s: %w", envelope.Type, pubErr)
	}

	return nil
}

func (p *RabbitPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked()
}

func (p *RabbitPublisher) ensureChannel(ctx context.Context) (*amqp.Channel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil && !p.channel.IsClosed() {
		return p.channel, nil
	}

	if p.uri == "" {
		return nil, fmt.Errorf("rabbitmq uri not configured")
	}

	if err := p.closeLocked(); err != nil {
		return nil, err
	}

	conn, err := amqp.DialConfig(p.uri, amqp.Config{
		Dial: amqp.DefaultDial(5 * time.Second),
	})
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := channel.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	p.conn = conn
	p.channel = channel
	p.declared = true
	return p.channel, nil
}

func (p *RabbitPublisher) reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked()
}

func (p *RabbitPublisher) closeLocked() error {
	var firstErr error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.channel = nil
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.conn = nil
	}
	p.declared = false
	return firstErr
}
