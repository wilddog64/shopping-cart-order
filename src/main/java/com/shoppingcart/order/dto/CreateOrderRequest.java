package com.shoppingcart.order.dto;

import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import java.math.BigDecimal;
import java.util.List;

public record CreateOrderRequest(
    @NotBlank String customerId,
    @NotEmpty @Valid List<OrderItemRequest> items,
    @Valid AddressRequest shippingAddress,
    String currency
) {
    public record OrderItemRequest(
        @NotBlank String productId,
        @NotBlank String productName,
        @Positive int quantity,
        @NotNull @Positive BigDecimal unitPrice
    ) {}

    public record AddressRequest(
        @NotBlank String street,
        @NotBlank String city,
        @NotBlank String state,
        @NotBlank String postalCode,
        @NotBlank String country
    ) {}
}
