# Order Service Documentation

Welcome to the Order Service documentation. This service handles order management for the Shopping Cart platform.

## Quick Links

| Document | Description |
|----------|-------------|
| [Architecture](architecture/README.md) | System design, components, and data model |
| [API Reference](api/README.md) | REST API endpoints and examples |
| [Troubleshooting](troubleshooting/README.md) | Common issues and debugging guide |

## Overview

The Order Service is a Spring Boot microservice responsible for:
- Creating and managing customer orders
- Order lifecycle management (pending → confirmed → shipped → delivered)
- Publishing order events to RabbitMQ
- Integration with Keycloak for OAuth2/OIDC authentication

## Getting Started

### Prerequisites

- Java 21+
- Maven 3.9+
- PostgreSQL 15+
- RabbitMQ 3.12+ (optional)
- Keycloak (optional, for OAuth2)

### Running Locally

```bash
# Build
mvn clean package -DskipTests

# Run with default settings (OAuth2 disabled)
java -jar target/shopping-cart-order-1.0.0-SNAPSHOT.jar

# Run with OAuth2 enabled
OAUTH2_ENABLED=true \
OAUTH2_ISSUER_URI=http://keycloak:8080/realms/shopping-cart \
java -jar target/shopping-cart-order-1.0.0-SNAPSHOT.jar
```

### Running Tests

```bash
# All tests
mvn test

# Specific test class
mvn test -Dtest="OAuth2SecurityConfigTest"
```

## Configuration

Key environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SPRING_DATASOURCE_URL` | PostgreSQL URL | `jdbc:postgresql://localhost:5432/orders` |
| `SPRING_RABBITMQ_HOST` | RabbitMQ host | `localhost` |
| `OAUTH2_ENABLED` | Enable OAuth2 | `false` |
| `OAUTH2_ISSUER_URI` | Keycloak issuer | - |

See [Architecture > Configuration](architecture/README.md#configuration) for full list.

## API Quick Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/orders` | Create order |
| GET | `/api/orders/{id}` | Get order by ID |
| GET | `/api/orders?customerId={id}` | List customer orders |
| PATCH | `/api/orders/{id}/status` | Update status |
| POST | `/api/orders/{id}/cancel` | Cancel order |
| GET | `/actuator/health` | Health check |

See [API Reference](api/README.md) for complete documentation.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Controller │────▶│   Service   │────▶│ Repository  │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Security   │     │   Events    │     │  PostgreSQL │
│  (OAuth2)   │     │ (RabbitMQ)  │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
```

See [Architecture](architecture/README.md) for detailed design.

## Related Services

- **Product Catalog Service**: Product information management
- **Keycloak**: Identity and access management
- **RabbitMQ**: Message broker for event publishing

## Support

- [Troubleshooting Guide](troubleshooting/README.md)
- [GitHub Issues](https://github.com/your-org/shopping-cart-order/issues)
