package com.shoppingcart.order.config;

import io.github.bucket4j.Bandwidth;
import io.github.bucket4j.Bucket;
import io.github.bucket4j.Refill;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;

import java.time.Duration;
import java.util.concurrent.TimeUnit;

/**
 * Rate limiting configuration using Bucket4j.
 *
 * Provides protection against DDoS and abuse by limiting:
 * - Requests per IP address
 * - Requests per API endpoint
 */
@Configuration
@ConfigurationProperties(prefix = "rate-limit")
public class RateLimitConfig {

    private int requestsPerMinute = 100;
    private int requestsPerSecond = 20;
    private int burstCapacity = 50;

    /**
     * Cache to store rate limit buckets per IP address.
     * Buckets expire after 1 hour of inactivity.
     */
    @Bean
    public Cache<String, Bucket> rateLimitBucketCache() {
        return Caffeine.newBuilder()
            .expireAfterAccess(1, TimeUnit.HOURS)
            .maximumSize(100_000) // Max 100k unique IPs
            .build();
    }

    /**
     * Creates a new rate limit bucket for an IP address.
     *
     * Uses a token bucket algorithm with:
     * - Burst capacity for handling spikes
     * - Per-second refill for sustained traffic
     * - Per-minute limit for overall cap
     */
    public Bucket createNewBucket() {
        return Bucket.builder()
            // Allow burst of requests
            .addLimit(Bandwidth.classic(burstCapacity, Refill.greedy(burstCapacity, Duration.ofSeconds(1))))
            // Sustained rate limit per second
            .addLimit(Bandwidth.classic(requestsPerSecond, Refill.greedy(requestsPerSecond, Duration.ofSeconds(1))))
            // Overall limit per minute
            .addLimit(Bandwidth.classic(requestsPerMinute, Refill.intervally(requestsPerMinute, Duration.ofMinutes(1))))
            .build();
    }

    /**
     * Creates a stricter bucket for sensitive endpoints (e.g., order creation).
     */
    public Bucket createStrictBucket() {
        return Bucket.builder()
            .addLimit(Bandwidth.classic(10, Refill.greedy(10, Duration.ofSeconds(1))))
            .addLimit(Bandwidth.classic(30, Refill.intervally(30, Duration.ofMinutes(1))))
            .build();
    }

    // Getters and setters for configuration properties
    public int getRequestsPerMinute() { return requestsPerMinute; }
    public void setRequestsPerMinute(int requestsPerMinute) { this.requestsPerMinute = requestsPerMinute; }

    public int getRequestsPerSecond() { return requestsPerSecond; }
    public void setRequestsPerSecond(int requestsPerSecond) { this.requestsPerSecond = requestsPerSecond; }

    public int getBurstCapacity() { return burstCapacity; }
    public void setBurstCapacity(int burstCapacity) { this.burstCapacity = burstCapacity; }
}
