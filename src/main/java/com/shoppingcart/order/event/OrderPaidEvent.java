package com.shoppingcart.order.event;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.math.BigDecimal;
import java.time.Instant;

/**
 * Event published when payment is confirmed for an order.
 * Routing key: order.paid
 */
public record OrderPaidEvent(
    @JsonProperty("orderId") String orderId,
    @JsonProperty("paymentId") String paymentId,
    @JsonProperty("amount") BigDecimal amount,
    @JsonProperty("currency") String currency,
    @JsonProperty("paymentMethod") String paymentMethod,
    @JsonProperty("paidAt") Instant paidAt
) {
    public static final String TYPE = "order.paid";
    public static final String VERSION = "1.0";

    public EventEnvelope<OrderPaidEvent> toEnvelope() {
        return EventEnvelope.create(TYPE, VERSION, this);
    }

    public EventEnvelope<OrderPaidEvent> toEnvelope(String correlationId) {
        return EventEnvelope.create(TYPE, VERSION, this, correlationId);
    }
}
