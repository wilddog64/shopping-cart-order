package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wilddog64/shopping-cart-order/internal/events"
	"go.uber.org/zap"
)

var (
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrCancelNotAllowed  = errors.New("cancel not allowed")
	ErrOrderValidation   = errors.New("order validation failed")
)

type EventPublisher interface {
	Publish(ctx context.Context, envelope events.Envelope) error
}

type Service struct {
	store     Store
	publisher EventPublisher
	logger    *zap.Logger
}

func NewService(store Store, publisher EventPublisher, logger *zap.Logger) *Service {
	return &Service{
		store:     store,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest, correlationID string) (*Order, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	orderEntity := NewOrder(req.CustomerID, currency)
	orderEntity.Items = make([]OrderItem, 0, len(req.Items))
	for _, itemReq := range req.Items {
		item := OrderItem{
			ID:          uuid.New(),
			ProductID:   itemReq.ProductID,
			ProductName: itemReq.ProductName,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
		}
		item.RecalculateSubtotal()
		orderEntity.Items = append(orderEntity.Items, item)
	}
	orderEntity.RecalculateTotals()
	if req.ShippingAddress != nil {
		orderEntity.ShippingAddress = &ShippingAddress{
			Street:     req.ShippingAddress.Street,
			City:       req.ShippingAddress.City,
			State:      req.ShippingAddress.State,
			PostalCode: req.ShippingAddress.PostalCode,
			Country:    req.ShippingAddress.Country,
		}
	}

	if err := s.store.Create(ctx, orderEntity); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, events.NewOrderCreatedEvent(
			orderEntity.ID.String(),
			orderEntity.CustomerID,
			toEventItems(orderEntity.Items),
			orderEntity.TotalAmount,
			orderEntity.Currency,
			toEventAddress(orderEntity.ShippingAddress),
		).ToEnvelope(correlationID)); err != nil {
			s.logger.Warn("failed to publish order.created event",
				zap.String("orderId", orderEntity.ID.String()),
				zap.String("correlationId", correlationID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("created order",
		zap.String("orderId", orderEntity.ID.String()),
		zap.String("customerId", orderEntity.CustomerID),
		zap.String("correlationId", correlationID),
	)

	return orderEntity, nil
}

func (s *Service) GetOrder(ctx context.Context, id uuid.UUID) (*Order, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) ListOrdersByCustomer(ctx context.Context, customerID string) ([]*Order, error) {
	return s.store.ListByCustomer(ctx, customerID)
}

func (s *Service) UpdateOrderStatus(ctx context.Context, id uuid.UUID, req UpdateOrderStatusRequest, correlationID string) (*Order, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	orderEntity, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	nextStatus := req.Status
	currentStatus := orderEntity.Status
	if !currentStatus.CanTransitionTo(nextStatus) {
		return nil, &InvalidTransitionError{From: orderEntity.Status, To: nextStatus}
	}

	now := time.Now().UTC()
	orderEntity.Status = nextStatus
	orderEntity.UpdatedAt = now

	switch nextStatus {
	case OrderStatusPaid:
		orderEntity.PaidAt = &now
	case OrderStatusShipped:
		orderEntity.ShippedAt = &now
		orderEntity.TrackingNumber = stringPtr(req.TrackingNumber)
		orderEntity.Carrier = stringPtr(req.Carrier)
	case OrderStatusCompleted:
		orderEntity.CompletedAt = &now
	}

	if err := s.store.Update(ctx, orderEntity); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		switch nextStatus {
		case OrderStatusPaid:
			if err := s.publisher.Publish(ctx, events.NewOrderPaidEvent(
				orderEntity.ID.String(),
				req.PaymentID,
				orderEntity.TotalAmount,
				orderEntity.Currency,
				req.PaymentMethod,
				valueOrZero(orderEntity.PaidAt),
			).ToEnvelope(correlationID)); err != nil {
				s.logger.Warn("failed to publish order.paid event",
					zap.String("orderId", orderEntity.ID.String()),
					zap.String("correlationId", correlationID),
					zap.Error(err),
				)
			}
		case OrderStatusShipped:
			estimatedDelivery := req.EstimatedDelivery
			if estimatedDelivery == nil {
				today := time.Now().UTC().AddDate(0, 0, 5)
				estimatedDelivery = &DateOnly{Time: today}
			}
			var trackingNumber string
			if orderEntity.TrackingNumber != nil {
				trackingNumber = *orderEntity.TrackingNumber
			}
			var carrier string
			if orderEntity.Carrier != nil {
				carrier = *orderEntity.Carrier
			}
			if err := s.publisher.Publish(ctx, events.NewOrderShippedEvent(
				orderEntity.ID.String(),
				trackingNumber,
				carrier,
				valueOrZero(orderEntity.ShippedAt),
				events.NewDateOnly(estimatedDelivery.Time),
			).ToEnvelope(correlationID)); err != nil {
				s.logger.Warn("failed to publish order.shipped event",
					zap.String("orderId", orderEntity.ID.String()),
					zap.String("correlationId", correlationID),
					zap.Error(err),
				)
			}
		case OrderStatusCompleted:
			if err := s.publisher.Publish(ctx, events.NewOrderCompletedEvent(
				orderEntity.ID.String(),
				valueOrZero(orderEntity.CompletedAt),
			).ToEnvelope(correlationID)); err != nil {
				s.logger.Warn("failed to publish order.completed event",
					zap.String("orderId", orderEntity.ID.String()),
					zap.String("correlationId", correlationID),
					zap.Error(err),
				)
			}
		}
	}

	s.logger.Info("updated order status",
		zap.String("orderId", orderEntity.ID.String()),
		zap.String("from", string(currentStatus)),
		zap.String("to", string(nextStatus)),
		zap.String("correlationId", correlationID),
	)

	return orderEntity, nil
}

func (s *Service) CancelOrder(ctx context.Context, id uuid.UUID, reason, cancelledBy, correlationID string) (*Order, error) {
	if cancelledBy == "" {
		cancelledBy = "system"
	}

	orderEntity, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if orderEntity.Status == OrderStatusShipped || orderEntity.Status == OrderStatusCompleted {
		return nil, &CancelNotAllowedError{Status: orderEntity.Status}
	}

	now := time.Now().UTC()
	orderEntity.Status = OrderStatusCancelled
	orderEntity.CancelledAt = &now
	orderEntity.CancellationReason = stringPtr(reason)
	orderEntity.UpdatedAt = now

	if err := s.store.Update(ctx, orderEntity); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, events.NewOrderCancelledEvent(
			orderEntity.ID.String(),
			stringValue(orderEntity.CancellationReason),
			cancelledBy,
			valueOrZero(orderEntity.CancelledAt),
			true,
		).ToEnvelope(correlationID)); err != nil {
			s.logger.Warn("failed to publish order.cancelled event",
				zap.String("orderId", orderEntity.ID.String()),
				zap.String("correlationId", correlationID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("cancelled order",
		zap.String("orderId", orderEntity.ID.String()),
		zap.String("correlationId", correlationID),
		zap.String("cancelledBy", cancelledBy),
	)

	return orderEntity, nil
}

type InvalidTransitionError struct {
	From OrderStatus
	To   OrderStatus
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("Invalid status transition from %s to %s", e.From, e.To)
}

type CancelNotAllowedError struct {
	Status OrderStatus
}

func (e *CancelNotAllowedError) Error() string {
	return fmt.Sprintf("Cannot cancel order in status: %s", e.Status)
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func toEventItems(items []OrderItem) []events.OrderItem {
	result := make([]events.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, events.OrderItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		})
	}
	return result
}

func toEventAddress(address *ShippingAddress) *events.Address {
	if address == nil {
		return nil
	}
	return &events.Address{
		Street:     address.Street,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}
}

func valueOrZero(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
