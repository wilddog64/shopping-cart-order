package com.shoppingcart.order.dto;

import jakarta.validation.constraints.NotBlank;

public record CancelOrderRequest(
    @NotBlank String reason
) {}
