package com.shoppingcart.order.service;

import com.shoppingcart.order.entity.Order;
import com.shoppingcart.order.entity.OrderItem;
import com.shoppingcart.order.event.*;
import com.shoppingcart.rabbitmq.publisher.Publisher;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.time.LocalDate;
import java.util.List;

/**
 * Service responsible for publishing order events to RabbitMQ.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class OrderEventPublisher {

    private static final String EVENTS_EXCHANGE = "events";

    private final Publisher publisher;

    /**
     * Publishes an order.created event when a new order is created.
     */
    public void publishOrderCreated(Order order, String correlationId) {
        List<OrderCreatedEvent.OrderItem> items = order.getItems().stream()
            .map(this::toEventItem)
            .toList();

        OrderCreatedEvent.Address address = null;
        if (order.getShippingAddress() != null) {
            address = new OrderCreatedEvent.Address(
                order.getShippingAddress().getStreet(),
                order.getShippingAddress().getCity(),
                order.getShippingAddress().getState(),
                order.getShippingAddress().getPostalCode(),
                order.getShippingAddress().getCountry()
            );
        }

        OrderCreatedEvent event = new OrderCreatedEvent(
            order.getId().toString(),
            order.getCustomerId(),
            items,
            order.getTotalAmount(),
            order.getCurrency(),
            address
        );

        EventEnvelope<OrderCreatedEvent> envelope = event.toEnvelope(correlationId);

        log.info("Publishing order.created event: orderId={}, correlationId={}",
            order.getId(), envelope.correlationId());

        publisher.publish(EVENTS_EXCHANGE, OrderCreatedEvent.TYPE, envelope);
    }

    /**
     * Publishes an order.paid event when payment is confirmed.
     */
    public void publishOrderPaid(Order order, String paymentId, String paymentMethod, String correlationId) {
        OrderPaidEvent event = new OrderPaidEvent(
            order.getId().toString(),
            paymentId,
            order.getTotalAmount(),
            order.getCurrency(),
            paymentMethod,
            order.getPaidAt() != null ? order.getPaidAt() : Instant.now()
        );

        EventEnvelope<OrderPaidEvent> envelope = event.toEnvelope(correlationId);

        log.info("Publishing order.paid event: orderId={}, paymentId={}, correlationId={}",
            order.getId(), paymentId, envelope.correlationId());

        publisher.publish(EVENTS_EXCHANGE, OrderPaidEvent.TYPE, envelope);
    }

    /**
     * Publishes an order.shipped event when order is shipped.
     */
    public void publishOrderShipped(Order order, LocalDate estimatedDelivery, String correlationId) {
        OrderShippedEvent event = new OrderShippedEvent(
            order.getId().toString(),
            order.getTrackingNumber(),
            order.getCarrier(),
            order.getShippedAt() != null ? order.getShippedAt() : Instant.now(),
            estimatedDelivery
        );

        EventEnvelope<OrderShippedEvent> envelope = event.toEnvelope(correlationId);

        log.info("Publishing order.shipped event: orderId={}, trackingNumber={}, correlationId={}",
            order.getId(), order.getTrackingNumber(), envelope.correlationId());

        publisher.publish(EVENTS_EXCHANGE, OrderShippedEvent.TYPE, envelope);
    }

    /**
     * Publishes an order.completed event when order is delivered.
     */
    public void publishOrderCompleted(Order order, String correlationId) {
        OrderCompletedEvent event = new OrderCompletedEvent(
            order.getId().toString(),
            order.getCompletedAt() != null ? order.getCompletedAt() : Instant.now()
        );

        EventEnvelope<OrderCompletedEvent> envelope = event.toEnvelope(correlationId);

        log.info("Publishing order.completed event: orderId={}, correlationId={}",
            order.getId(), envelope.correlationId());

        publisher.publish(EVENTS_EXCHANGE, OrderCompletedEvent.TYPE, envelope);
    }

    /**
     * Publishes an order.cancelled event when order is cancelled.
     */
    public void publishOrderCancelled(Order order, String cancelledBy, String correlationId) {
        OrderCancelledEvent event = new OrderCancelledEvent(
            order.getId().toString(),
            order.getCancellationReason(),
            cancelledBy,
            order.getCancelledAt() != null ? order.getCancelledAt() : Instant.now(),
            true // refundInitiated - would typically be determined by business logic
        );

        EventEnvelope<OrderCancelledEvent> envelope = event.toEnvelope(correlationId);

        log.info("Publishing order.cancelled event: orderId={}, reason={}, correlationId={}",
            order.getId(), order.getCancellationReason(), envelope.correlationId());

        publisher.publish(EVENTS_EXCHANGE, OrderCancelledEvent.TYPE, envelope);
    }

    private OrderCreatedEvent.OrderItem toEventItem(OrderItem item) {
        return new OrderCreatedEvent.OrderItem(
            item.getProductId(),
            item.getProductName(),
            item.getQuantity(),
            item.getUnitPrice()
        );
    }
}
