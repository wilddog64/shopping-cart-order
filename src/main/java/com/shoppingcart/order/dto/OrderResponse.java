package com.shoppingcart.order.dto;

import com.shoppingcart.order.entity.Order;
import com.shoppingcart.order.entity.OrderItem;
import com.shoppingcart.order.entity.OrderStatus;
import java.math.BigDecimal;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

public record OrderResponse(
    UUID id,
    String customerId,
    OrderStatus status,
    List<OrderItemResponse> items,
    BigDecimal totalAmount,
    String currency,
    AddressResponse shippingAddress,
    String trackingNumber,
    String carrier,
    Instant createdAt,
    Instant updatedAt,
    Instant paidAt,
    Instant shippedAt,
    Instant completedAt,
    Instant cancelledAt,
    String cancellationReason
) {
    public record OrderItemResponse(
        UUID id,
        String productId,
        String productName,
        int quantity,
        BigDecimal unitPrice,
        BigDecimal subtotal
    ) {
        public static OrderItemResponse from(OrderItem item) {
            return new OrderItemResponse(
                item.getId(),
                item.getProductId(),
                item.getProductName(),
                item.getQuantity(),
                item.getUnitPrice(),
                item.getSubtotal()
            );
        }
    }

    public record AddressResponse(
        String street,
        String city,
        String state,
        String postalCode,
        String country
    ) {}

    public static OrderResponse from(Order order) {
        AddressResponse address = null;
        if (order.getShippingAddress() != null) {
            address = new AddressResponse(
                order.getShippingAddress().getStreet(),
                order.getShippingAddress().getCity(),
                order.getShippingAddress().getState(),
                order.getShippingAddress().getPostalCode(),
                order.getShippingAddress().getCountry()
            );
        }

        return new OrderResponse(
            order.getId(),
            order.getCustomerId(),
            order.getStatus(),
            order.getItems().stream().map(OrderItemResponse::from).toList(),
            order.getTotalAmount(),
            order.getCurrency(),
            address,
            order.getTrackingNumber(),
            order.getCarrier(),
            order.getCreatedAt(),
            order.getUpdatedAt(),
            order.getPaidAt(),
            order.getShippedAt(),
            order.getCompletedAt(),
            order.getCancelledAt(),
            order.getCancellationReason()
        );
    }
}
