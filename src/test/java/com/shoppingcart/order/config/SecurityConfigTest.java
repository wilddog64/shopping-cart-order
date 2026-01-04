package com.shoppingcart.order.config;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.context.annotation.Import;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * Integration tests for security headers configuration.
 * Uses @WebMvcTest for lightweight testing without full application context.
 */
@WebMvcTest(controllers = TestController.class)
@Import({SecurityConfig.class, RateLimitConfig.class})
@TestPropertySource(properties = {
    "oauth2.enabled=false",
    "rate-limit.requests-per-minute=100",
    "rate-limit.requests-per-second=20",
    "rate-limit.burst-capacity=50"
})
class SecurityConfigTest {

    @Autowired
    private MockMvc mockMvc;

    @Test
    @DisplayName("Should include X-XSS-Protection header")
    void shouldIncludeXssProtectionHeader() throws Exception {
        MvcResult result = mockMvc.perform(get("/actuator/health"))
            .andReturn();

        String header = result.getResponse().getHeader("X-XSS-Protection");
        assertThat(header).isNotNull();
        assertThat(header).contains("1");
    }

    @Test
    @DisplayName("Should include X-Frame-Options header")
    void shouldIncludeFrameOptionsHeader() throws Exception {
        mockMvc.perform(get("/actuator/health"))
            .andExpect(header().string("X-Frame-Options", "DENY"));
    }

    @Test
    @DisplayName("Should include X-Content-Type-Options header")
    void shouldIncludeContentTypeOptionsHeader() throws Exception {
        mockMvc.perform(get("/actuator/health"))
            .andExpect(header().string("X-Content-Type-Options", "nosniff"));
    }

    @Test
    @DisplayName("Should include Content-Security-Policy header")
    void shouldIncludeContentSecurityPolicyHeader() throws Exception {
        mockMvc.perform(get("/actuator/health"))
            .andExpect(header().exists("Content-Security-Policy"));
    }

    @Test
    @DisplayName("Should include Referrer-Policy header")
    void shouldIncludeReferrerPolicyHeader() throws Exception {
        mockMvc.perform(get("/actuator/health"))
            .andExpect(header().exists("Referrer-Policy"));
    }

    @Test
    @DisplayName("Should allow health endpoint without authentication")
    void shouldAllowHealthEndpoint() throws Exception {
        mockMvc.perform(get("/actuator/health"))
            .andExpect(status().isOk());
    }

    @Test
    @DisplayName("Should allow API endpoints without authentication")
    void shouldAllowApiEndpoints() throws Exception {
        MvcResult result = mockMvc.perform(get("/api/orders"))
            .andReturn();

        int statusCode = result.getResponse().getStatus();
        // Should not be 401 or 403 when OAuth2 is disabled
        assertThat(statusCode).isNotEqualTo(401);
        assertThat(statusCode).isNotEqualTo(403);
    }

    @Test
    @DisplayName("Should deny unknown endpoints")
    void shouldDenyUnknownEndpoints() throws Exception {
        MvcResult result = mockMvc.perform(get("/unknown/endpoint"))
            .andReturn();

        int statusCode = result.getResponse().getStatus();
        // Unknown paths should return 403 (denied) or 404 (not found)
        assertThat(statusCode).isIn(403, 404);
    }

    @Test
    @DisplayName("Should include rate limit header on API requests")
    void shouldIncludeRateLimitHeader() throws Exception {
        mockMvc.perform(get("/api/orders"))
            .andExpect(header().exists("X-Rate-Limit-Remaining"));
    }
}
