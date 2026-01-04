package com.shoppingcart.order.config;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Minimal test controller for security integration tests.
 * This controller provides endpoints to test security configuration
 * without requiring the full application context.
 */
@RestController
class TestController {

    @GetMapping("/api/test")
    public String testEndpoint() {
        return "test";
    }

    @GetMapping("/api/orders")
    public String ordersEndpoint() {
        return "orders";
    }

    @GetMapping("/actuator/health")
    public String healthEndpoint() {
        return "{\"status\":\"UP\"}";
    }

    @GetMapping("/actuator/info")
    public String infoEndpoint() {
        return "{}";
    }
}
