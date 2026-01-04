package com.shoppingcart.order.event;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;

/**
 * Event published when an order is marked as complete (delivered).
 * Routing key: order.completed
 */
public record OrderCompletedEvent(
    @JsonProperty("orderId") String orderId,
    @JsonProperty("completedAt") Instant completedAt
) {
    public static final String TYPE = "order.completed";
    public static final String VERSION = "1.0";

    public EventEnvelope<OrderCompletedEvent> toEnvelope() {
        return EventEnvelope.create(TYPE, VERSION, this);
    }

    public EventEnvelope<OrderCompletedEvent> toEnvelope(String correlationId) {
        return EventEnvelope.create(TYPE, VERSION, this, correlationId);
    }
}
