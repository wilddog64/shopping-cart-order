package com.shoppingcart.order.dto;

import com.shoppingcart.order.entity.OrderStatus;
import jakarta.validation.constraints.NotNull;
import java.time.LocalDate;

public record UpdateOrderStatusRequest(
    @NotNull OrderStatus status,
    // For PAID status
    String paymentId,
    String paymentMethod,
    // For SHIPPED status
    String trackingNumber,
    String carrier,
    LocalDate estimatedDelivery
) {}
