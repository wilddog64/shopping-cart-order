package com.shoppingcart.order.event;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;
import java.util.UUID;

/**
 * Common envelope format for all events.
 * Matches the schema defined in message-schemas.md
 */
public record EventEnvelope<T>(
    @JsonProperty("id") String id,
    @JsonProperty("type") String type,
    @JsonProperty("version") String version,
    @JsonProperty("timestamp") Instant timestamp,
    @JsonProperty("source") String source,
    @JsonProperty("correlationId") String correlationId,
    @JsonProperty("data") T data
) {
    private static final String SOURCE = "order-service";

    public static <T> EventEnvelope<T> create(String type, String version, T data, String correlationId) {
        return new EventEnvelope<>(
            UUID.randomUUID().toString(),
            type,
            version,
            Instant.now(),
            SOURCE,
            correlationId != null ? correlationId : UUID.randomUUID().toString(),
            data
        );
    }

    public static <T> EventEnvelope<T> create(String type, String version, T data) {
        return create(type, version, data, null);
    }
}
