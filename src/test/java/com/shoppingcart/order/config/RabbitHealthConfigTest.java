package com.shoppingcart.order.config;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.when;

import com.shoppingcart.rabbitmq.connection.ConnectionManager;
import com.shoppingcart.rabbitmq.connection.ConnectionManager.ConnectionStats;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthIndicator;
import org.springframework.boot.actuate.health.Status;

@ExtendWith(MockitoExtension.class)
class RabbitHealthConfigTest {

  @Mock private ConnectionManager connectionManager;

  private RabbitHealthConfig config;

  @BeforeEach
  void setUp() {
    config = new RabbitHealthConfig();
  }

  @Test
  void returnsStatsWhenAvailable() {
    when(connectionManager.getStats())
        .thenReturn(new ConnectionStats(5, 2, 3, true));

    HealthIndicator indicator = config.rabbitHealthIndicator(connectionManager);
    Health health = indicator.health();

    assertThat(health.getStatus()).isEqualTo(Status.UP);
    assertThat(health.getDetails())
        .containsEntry("totalChannels", 5)
        .containsEntry("activeChannels", 2)
        .containsEntry("idleChannels", 3)
        .containsEntry("connected", true);
  }

  @Test
  void returnsFallbackWhenStatsMissing() {
    when(connectionManager.getStats()).thenReturn(null);

    Health health = config.rabbitHealthIndicator(connectionManager).health();

    assertThat(health.getStatus()).isEqualTo(Status.UP);
    assertThat(health.getDetails())
        .containsEntry("totalChannels", 0)
        .containsEntry("activeChannels", 0)
        .containsEntry("idleChannels", 0)
        .containsEntry("connected", false)
        .containsEntry("reason", "statsUnavailable");
  }

  @Test
  void returnsFallbackWhenNullPointerThrown() {
    when(connectionManager.getStats()).thenThrow(new NullPointerException("cache not ready"));

    Health health = config.rabbitHealthIndicator(connectionManager).health();

    assertThat(health.getStatus()).isEqualTo(Status.UP);
    assertThat(health.getDetails())
        .containsEntry("reason", "empty-cache")
        .containsEntry("connected", false);
  }

  @Test
  void returnsDownWhenRealFailureOccurs() {
    when(connectionManager.getStats()).thenThrow(new IllegalStateException("boom"));

    Health health = config.rabbitHealthIndicator(connectionManager).health();

    assertThat(health.getStatus()).isEqualTo(Status.DOWN);
  }
}
