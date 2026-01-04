package com.shoppingcart.order.event;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;

/**
 * Event published when an order is cancelled.
 * Routing key: order.cancelled
 */
public record OrderCancelledEvent(
    @JsonProperty("orderId") String orderId,
    @JsonProperty("reason") String reason,
    @JsonProperty("cancelledBy") String cancelledBy,
    @JsonProperty("cancelledAt") Instant cancelledAt,
    @JsonProperty("refundInitiated") boolean refundInitiated
) {
    public static final String TYPE = "order.cancelled";
    public static final String VERSION = "1.0";

    public EventEnvelope<OrderCancelledEvent> toEnvelope() {
        return EventEnvelope.create(TYPE, VERSION, this);
    }

    public EventEnvelope<OrderCancelledEvent> toEnvelope(String correlationId) {
        return EventEnvelope.create(TYPE, VERSION, this, correlationId);
    }
}
