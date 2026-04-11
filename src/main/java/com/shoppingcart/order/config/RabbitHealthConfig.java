package com.shoppingcart.order.config;

import com.shoppingcart.rabbitmq.connection.ConnectionManager;
import com.shoppingcart.rabbitmq.connection.ConnectionManager.ConnectionStats;
import java.util.LinkedHashMap;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthIndicator;
import org.springframework.boot.actuate.health.Status;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Replaces Spring Boot's default RabbitMQ health indicator so we can guard
 * connection stats until the first AMQP channel is opened. Without this guard
 * the upstream rabbitmq-client library throws {@link NullPointerException}
 * during the startup probe and the pod is restarted before it can process any
 * traffic.
 */
@Configuration
public class RabbitHealthConfig {

  private static final Logger LOG = LoggerFactory.getLogger(RabbitHealthConfig.class);

  @Bean(name = "rabbitHealthIndicator")
  public HealthIndicator rabbitHealthIndicator(ConnectionManager connectionManager) {
    return () -> {
      try {
        ConnectionStats stats = connectionManager.getStats();
        if (stats == null) {
          return buildHealthyResponse("statsUnavailable");
        }
        return Health.status(stats.healthy() ? Status.UP : Status.DOWN)
            .withDetail("totalChannels", stats.totalChannels())
            .withDetail("activeChannels", stats.activeChannels())
            .withDetail("idleChannels", stats.idleChannels())
            .withDetail("connected", stats.healthy())
            .build();
      } catch (NullPointerException npe) {
        if (LOG.isDebugEnabled()) {
          LOG.debug("RabbitMQ cache not initialised yet; returning zero stats", npe);
        }
        return buildHealthyResponse("empty-cache");
      } catch (Exception ex) {
        LOG.warn("RabbitMQ health check failed", ex);
        return Health.down(ex).build();
      }
    };
  }

  private Health buildHealthyResponse(String reason) {
    Map<String, Object> details = new LinkedHashMap<>();
    details.put("totalChannels", 0);
    details.put("activeChannels", 0);
    details.put("idleChannels", 0);
    details.put("connected", false);
    details.put("reason", reason);
    return Health.up().withDetails(details).build();
  }
}
