# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

**Shopping Cart Order Service** is a Spring Boot microservice that handles order processing for the shopping cart platform. It publishes order lifecycle events to RabbitMQ for event-driven communication with other services.

## Technology Stack

- **Language**: Java 21
- **Framework**: Spring Boot 3.2
- **Database**: PostgreSQL 15+
- **Messaging**: RabbitMQ (via rabbitmq-client-java library)
- **Build**: Maven
- **Testing**: JUnit 5, Testcontainers

## Key Patterns

### Event-Driven Architecture

Orders publish events to RabbitMQ on state transitions:
- `order.created` - New order placed
- `order.paid` - Payment confirmed
- `order.shipped` - Order shipped
- `order.completed` - Delivery confirmed
- `order.cancelled` - Order cancelled

Events use a common envelope format defined in `EventEnvelope.java`:
```java
{
  "id": "uuid",
  "type": "order.created",
  "version": "1.0",
  "timestamp": "2025-01-01T00:00:00Z",
  "source": "order-service",
  "correlationId": "uuid",
  "data": { ... }
}
```

### Domain Entities

- `Order` - Main aggregate root with status lifecycle
- `OrderItem` - Line items with product references
- `ShippingAddress` - Embedded shipping details
- `OrderStatus` - Enum: PENDING, PAID, PROCESSING, SHIPPED, COMPLETED, CANCELLED

## Directory Structure

```
src/main/java/com/shoppingcart/order/
├── OrderServiceApplication.java   # Main application
├── config/                        # Spring configuration
├── controller/                    # REST endpoints
├── dto/                          # Request/response DTOs
├── entity/                       # JPA entities
├── event/                        # RabbitMQ event classes
├── repository/                   # Spring Data repositories
└── service/                      # Business logic
```

## Common Development Tasks

### Build and Run

```bash
# Build
mvn clean package

# Run locally
mvn spring-boot:run

# Run with specific profile
mvn spring-boot:run -Dspring-boot.run.profiles=local
```

### Testing

```bash
# Unit tests only
mvn test

# Integration tests (requires Docker)
mvn verify -Pintegration

# Single test class
mvn test -Dtest=OrderServiceTest
```

### Database

```bash
# Generate migration (if using Flyway)
# Migrations go in src/main/resources/db/migration/

# Connect to local PostgreSQL
psql -h localhost -U postgres -d orders
```

## Configuration

Key configuration in `application.yml`:

| Property | Description |
|----------|-------------|
| `spring.datasource.*` | PostgreSQL connection |
| `rabbitmq.*` | RabbitMQ settings |
| `rabbitmq.vault.enabled` | Enable Vault credentials |

Environment variables override application.yml values.

## Dependencies

### RabbitMQ Client Library

Uses `rabbitmq-client-java` for RabbitMQ integration:
```xml
<dependency>
    <groupId>com.shoppingcart</groupId>
    <artifactId>rabbitmq-client</artifactId>
    <version>1.0.0-SNAPSHOT</version>
</dependency>
```

Install locally if not in Maven repo:
```bash
cd ../rabbitmq-client-java
mvn install
```

## Event Publishing

Example of publishing an order event:

```java
@Service
public class OrderService {
    private final Publisher publisher;

    public Order createOrder(CreateOrderRequest request) {
        Order order = // create order
        orderRepository.save(order);

        // Publish event
        OrderCreatedEvent event = new OrderCreatedEvent(
            order.getId().toString(),
            order.getCustomerId(),
            // ... other fields
        );
        publisher.publish("events", "order.created", event.toEnvelope());

        return order;
    }
}
```

## Related Documentation

- [Message Schemas](../shopping-cart-infra/docs/message-schemas.md) - Event contracts
- [RabbitMQ Operations](../shopping-cart-infra/docs/rabbitmq-operations.md) - Queue management
- [rabbitmq-client-java](../rabbitmq-client-java/README.md) - Client library docs

## Troubleshooting

### RabbitMQ Connection Issues

```bash
# Check RabbitMQ is running
kubectl get pods -n shopping-cart-data -l app=rabbitmq

# Verify credentials (if using Vault)
vault read rabbitmq/creds/order-publisher
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
kubectl get pods -n shopping-cart-data -l app=postgresql-orders

# Test connection
psql -h localhost -p 5432 -U postgres -d orders -c "SELECT 1"
```

## Code Style

- Follow standard Java conventions
- Use records for immutable DTOs and events
- Prefer constructor injection over field injection
- Add `@Transactional` to service methods that modify state
- Log at appropriate levels (DEBUG for details, INFO for operations, WARN/ERROR for issues)
