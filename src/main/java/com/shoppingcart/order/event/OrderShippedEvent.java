package com.shoppingcart.order.event;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;
import java.time.LocalDate;

/**
 * Event published when an order is shipped.
 * Routing key: order.shipped
 */
public record OrderShippedEvent(
    @JsonProperty("orderId") String orderId,
    @JsonProperty("trackingNumber") String trackingNumber,
    @JsonProperty("carrier") String carrier,
    @JsonProperty("shippedAt") Instant shippedAt,
    @JsonProperty("estimatedDelivery") LocalDate estimatedDelivery
) {
    public static final String TYPE = "order.shipped";
    public static final String VERSION = "1.0";

    public EventEnvelope<OrderShippedEvent> toEnvelope() {
        return EventEnvelope.create(TYPE, VERSION, this);
    }

    public EventEnvelope<OrderShippedEvent> toEnvelope(String correlationId) {
        return EventEnvelope.create(TYPE, VERSION, this, correlationId);
    }
}
