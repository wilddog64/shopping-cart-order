package com.shoppingcart.order.event;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.math.BigDecimal;
import java.util.List;

/**
 * Event published when a new order is created.
 * Routing key: order.created
 */
public record OrderCreatedEvent(
    @JsonProperty("orderId") String orderId,
    @JsonProperty("customerId") String customerId,
    @JsonProperty("items") List<OrderItem> items,
    @JsonProperty("totalAmount") BigDecimal totalAmount,
    @JsonProperty("currency") String currency,
    @JsonProperty("shippingAddress") Address shippingAddress
) {
    public static final String TYPE = "order.created";
    public static final String VERSION = "1.0";

    public record OrderItem(
        @JsonProperty("productId") String productId,
        @JsonProperty("productName") String productName,
        @JsonProperty("quantity") int quantity,
        @JsonProperty("unitPrice") BigDecimal unitPrice
    ) {}

    public record Address(
        @JsonProperty("street") String street,
        @JsonProperty("city") String city,
        @JsonProperty("state") String state,
        @JsonProperty("postalCode") String postalCode,
        @JsonProperty("country") String country
    ) {}

    public EventEnvelope<OrderCreatedEvent> toEnvelope() {
        return EventEnvelope.create(TYPE, VERSION, this);
    }

    public EventEnvelope<OrderCreatedEvent> toEnvelope(String correlationId) {
        return EventEnvelope.create(TYPE, VERSION, this, correlationId);
    }
}
