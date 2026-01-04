package com.shoppingcart.order.config;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import io.github.bucket4j.Bucket;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockFilterChain;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.mock.web.MockHttpServletResponse;

import java.util.concurrent.TimeUnit;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Unit tests for RateLimitFilter.
 */
class RateLimitFilterTest {

    private RateLimitFilter rateLimitFilter;
    private RateLimitConfig rateLimitConfig;
    private Cache<String, Bucket> bucketCache;

    @BeforeEach
    void setUp() {
        rateLimitConfig = new RateLimitConfig();
        rateLimitConfig.setRequestsPerMinute(10);
        rateLimitConfig.setRequestsPerSecond(5);
        rateLimitConfig.setBurstCapacity(5);

        bucketCache = Caffeine.newBuilder()
            .expireAfterAccess(1, TimeUnit.HOURS)
            .build();

        rateLimitFilter = new RateLimitFilter(bucketCache, rateLimitConfig);
    }

    @Test
    @DisplayName("Should allow requests within rate limit")
    void shouldAllowRequestsWithinLimit() throws Exception {
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.setRemoteAddr("192.168.1.1");
        request.setRequestURI("/api/orders");

        MockHttpServletResponse response = new MockHttpServletResponse();
        MockFilterChain filterChain = new MockFilterChain();

        rateLimitFilter.doFilter(request, response, filterChain);

        assertThat(response.getStatus()).isEqualTo(200);
        assertThat(response.getHeader("X-Rate-Limit-Remaining")).isNotNull();
    }

    @Test
    @DisplayName("Should block requests exceeding rate limit")
    void shouldBlockRequestsExceedingLimit() throws Exception {
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.setRemoteAddr("192.168.1.2");
        request.setRequestURI("/api/orders");

        // Exhaust the rate limit
        for (int i = 0; i < 10; i++) {
            MockHttpServletResponse response = new MockHttpServletResponse();
            MockFilterChain filterChain = new MockFilterChain();
            rateLimitFilter.doFilter(request, response, filterChain);
        }

        // Next request should be blocked
        MockHttpServletResponse response = new MockHttpServletResponse();
        MockFilterChain filterChain = new MockFilterChain();
        rateLimitFilter.doFilter(request, response, filterChain);

        assertThat(response.getStatus()).isEqualTo(429);
        assertThat(response.getHeader("Retry-After")).isNotNull();
        assertThat(response.getContentAsString()).contains("Too many requests");
    }

    @Test
    @DisplayName("Should skip rate limiting for actuator endpoints")
    void shouldSkipActuatorEndpoints() throws Exception {
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.setRemoteAddr("192.168.1.3");
        request.setRequestURI("/actuator/health");

        MockHttpServletResponse response = new MockHttpServletResponse();
        MockFilterChain filterChain = new MockFilterChain();

        rateLimitFilter.doFilter(request, response, filterChain);

        // Should not have rate limit headers (skipped)
        assertThat(response.getStatus()).isEqualTo(200);
    }

    @Test
    @DisplayName("Should use X-Forwarded-For header for client IP")
    void shouldUseForwardedForHeader() throws Exception {
        MockHttpServletRequest request1 = new MockHttpServletRequest();
        request1.setRemoteAddr("10.0.0.1"); // Proxy IP
        request1.addHeader("X-Forwarded-For", "203.0.113.1, 10.0.0.1");
        request1.setRequestURI("/api/orders");

        MockHttpServletRequest request2 = new MockHttpServletRequest();
        request2.setRemoteAddr("10.0.0.1"); // Same proxy IP
        request2.addHeader("X-Forwarded-For", "203.0.113.2, 10.0.0.1"); // Different client
        request2.setRequestURI("/api/orders");

        // Both should be tracked separately
        MockHttpServletResponse response1 = new MockHttpServletResponse();
        MockHttpServletResponse response2 = new MockHttpServletResponse();

        rateLimitFilter.doFilter(request1, response1, new MockFilterChain());
        rateLimitFilter.doFilter(request2, response2, new MockFilterChain());

        // Both should succeed (different client IPs)
        assertThat(response1.getStatus()).isEqualTo(200);
        assertThat(response2.getStatus()).isEqualTo(200);
    }

    @Test
    @DisplayName("Should rate limit per IP independently")
    void shouldRateLimitPerIpIndependently() throws Exception {
        // Exhaust rate limit for IP1
        for (int i = 0; i < 10; i++) {
            MockHttpServletRequest request = new MockHttpServletRequest();
            request.setRemoteAddr("192.168.1.10");
            request.setRequestURI("/api/orders");
            rateLimitFilter.doFilter(request, new MockHttpServletResponse(), new MockFilterChain());
        }

        // IP2 should still be allowed
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.setRemoteAddr("192.168.1.20");
        request.setRequestURI("/api/orders");
        MockHttpServletResponse response = new MockHttpServletResponse();

        rateLimitFilter.doFilter(request, response, new MockFilterChain());

        assertThat(response.getStatus()).isEqualTo(200);
    }
}
