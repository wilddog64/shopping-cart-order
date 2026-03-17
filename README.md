# Shopping Cart Order Service

Order processing microservice for the Shopping Cart platform with RabbitMQ integration for event-driven architecture.

## Overview

The Order Service handles:
- Order creation and lifecycle management
- Payment processing coordination
- Shipping and fulfillment tracking
- Event publishing for order state changes

## Architecture

See **[Service Architecture](docs/architecture/README.md)** for component diagram, service layers, order status flow, and security architecture.

## Events Published

| Event | Routing Key | Description |
|-------|-------------|-------------|
| OrderCreatedEvent | `order.created` | New order placed |
| OrderPaidEvent | `order.paid` | Payment confirmed |
| OrderShippedEvent | `order.shipped` | Order shipped |
| OrderCompletedEvent | `order.completed` | Order delivered |
| OrderCancelledEvent | `order.cancelled` | Order cancelled |

See [Message Schemas](../shopping-cart-infra/docs/message-schemas.md) for event format details.

## Prerequisites

- Java 21+
- Maven 3.9+
- PostgreSQL 15+
- RabbitMQ 3.12+ (via shopping-cart-infra)
- HashiCorp Vault (optional, for dynamic credentials)

## Quick Start

### 1. Build

```bash
mvn clean package
```

### 2. Run with Docker Compose (Development)

```bash
docker-compose up -d
mvn spring-boot:run
```

### 3. Run with Kubernetes

```bash
# Deploy to k3d cluster with shopping-cart-infra
kubectl apply -f k8s/
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_NAME` | orders | Database name |
| `DB_USERNAME` | postgres | Database username |
| `DB_PASSWORD` | postgres | Database password |
| `RABBITMQ_HOST` | localhost | RabbitMQ host |
| `RABBITMQ_PORT` | 5672 | RabbitMQ AMQP port |
| `VAULT_ENABLED` | false | Enable Vault integration |
| `VAULT_ADDR` | http://localhost:8200 | Vault address |
| `VAULT_ROLE` | order-publisher | Vault role for RabbitMQ |

### Vault Integration

When `VAULT_ENABLED=true`, the service fetches RabbitMQ credentials dynamically from Vault:

```bash
export VAULT_ENABLED=true
export VAULT_ADDR=http://vault.vault.svc.cluster.local:8200
export VAULT_ROLE=order-publisher
```

## API Endpoints

### Create Order
```bash
POST /api/orders
Content-Type: application/json

{
  "customerId": "cust-123",
  "items": [
    {
      "productId": "prod-456",
      "productName": "Widget",
      "quantity": 2,
      "unitPrice": 29.99
    }
  ],
  "shippingAddress": {
    "street": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "postalCode": "62701",
    "country": "US"
  }
}
```

### Get Order
```bash
GET /api/orders/{orderId}
```

### List Orders by Customer
```bash
GET /api/orders?customerId=cust-123
```

### Update Order Status
```bash
PATCH /api/orders/{orderId}/status
Content-Type: application/json

{
  "status": "SHIPPED",
  "trackingNumber": "1Z999AA10123456784",
  "carrier": "UPS"
}
```

### Cancel Order
```bash
POST /api/orders/{orderId}/cancel
Content-Type: application/json

{
  "reason": "Customer requested cancellation"
}
```

## Health & Metrics

| Endpoint | Description |
|----------|-------------|
| `/actuator/health` | Health check |
| `/actuator/metrics` | Metrics overview |
| `/actuator/prometheus` | Prometheus metrics |

## Development

### Run Tests

```bash
# Unit tests
mvn test

# Integration tests (requires Docker)
mvn verify -Pintegration
```

### Code Style

```bash
# Format code
mvn spotless:apply

# Check style
mvn spotless:check
```

## Project Structure

```
shopping-cart-order/
├── src/
│   ├── main/
│   │   ├── java/com/shoppingcart/order/
│   │   │   ├── config/         # Spring configuration
│   │   │   ├── controller/     # REST controllers
│   │   │   ├── dto/            # Request/response DTOs
│   │   │   ├── entity/         # JPA entities
│   │   │   ├── event/          # RabbitMQ events
│   │   │   ├── repository/     # Data repositories
│   │   │   └── service/        # Business logic
│   │   └── resources/
│   │       └── application.yml
│   └── test/
├── pom.xml
├── README.md
└── CLAUDE.md
```

## Documentation

### Architecture
- **[Service Architecture](docs/architecture/README.md)** — component diagram, service layers, order status flow, and security architecture.

### API Reference
- **[API Reference](docs/api/README.md)** — endpoint details, payloads, and error codes.

### Troubleshooting
- **[Troubleshooting Guide](docs/troubleshooting/README.md)** — common issues and debugging tips.

## Related Repositories

- [shopping-cart-infra](../shopping-cart-infra) - Kubernetes infrastructure, RabbitMQ cluster
- [shopping-cart-product-catalog](../shopping-cart-product-catalog) - Product catalog service
- [rabbitmq-client-java](../rabbitmq-client-java) - Java RabbitMQ client library

## License

MIT
