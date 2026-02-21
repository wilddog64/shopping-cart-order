# System Patterns: Order Service

## Architectural Pattern: Layered Spring Boot

```
HTTP Clients
    │
    ▼
┌──────────────────────────────────────────┐
│  Security Filters                         │
│  - RateLimitFilter (Bucket4j/Caffeine)    │
│  - OAuth2ResourceServerFilter (JWT)       │
│  - SecurityHeaders Filter                 │
└──────────────────┬───────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│  Controller Layer                         │
│  OrderController                          │
│  - Input validation (Jakarta Bean Val.)   │
│  - Request/Response DTO mapping           │
└──────────────────┬───────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│  Service Layer                            │
│  OrderService (@Transactional)            │
│  - Business logic                         │
│  - State machine enforcement              │
│  - Calls OrderEventPublisher              │
└──────────┬───────────────┬───────────────┘
           │               │
           ▼               ▼
┌──────────────┐  ┌──────────────────────┐
│ OrderRepo    │  │  OrderEventPublisher  │
│ (Spring JPA) │  │  (rabbitmq-client)    │
└──────────────┘  └──────────────────────┘
           │               │
           ▼               ▼
    PostgreSQL          RabbitMQ
```

## Domain Model

### Order Entity (JPA aggregate root)
```java
@Entity @Table(name = "orders")
class Order {
    UUID id;                    // @GeneratedValue(UUID)
    String customerId;          // from JWT sub or request
    OrderStatus status;         // @Enumerated(STRING)
    BigDecimal totalAmount;     // precision 10, scale 2
    String currency;            // default "USD"
    List<OrderItem> items;      // @OneToMany cascade ALL, orphanRemoval
    ShippingAddress shipping;   // @Embedded
    String trackingNumber;
    String carrier;
    Instant createdAt;          // set @PrePersist
    Instant updatedAt;          // set @PrePersist and @PreUpdate
    Instant paidAt;
    Instant shippedAt;
    Instant completedAt;
    Instant cancelledAt;
    String cancellationReason;
}
```

### OrderItem Entity
```java
@Entity
class OrderItem {
    UUID id;
    Order order;        // @ManyToOne back-reference
    String productId;
    String productName;
    Integer quantity;
    BigDecimal unitPrice;
    // subtotal computed: quantity * unitPrice
}
```

### OrderStatus Enum
```
PENDING → PAID → PROCESSING → SHIPPED → COMPLETED
    └────────────────────────────────→ CANCELLED
```

### ShippingAddress (Embedded)
```java
@Embeddable
class ShippingAddress {
    String street, city, state, postalCode, country;
}
```

## Order State Machine

Enforced in `OrderService.validateStatusTransition()` using Java 21 switch expression:

```
PENDING    → PAID | CANCELLED
PAID       → PROCESSING | CANCELLED
PROCESSING → SHIPPED | CANCELLED
SHIPPED    → COMPLETED             (cannot be cancelled)
COMPLETED  → (terminal)
CANCELLED  → (terminal)
```

Invalid transitions throw `IllegalStateException`. Each transition is `@Transactional` and atomically saves the entity and publishes the event within the same database transaction.

## Event-Driven Publishing Pattern

`OrderEventPublisher` publishes to RabbitMQ exchange `events` with routing keys:
- `order.created` — published in `createOrder()`
- `order.paid` — published when status transitions to PAID (includes paymentId, paymentMethod)
- `order.shipped` — published when status transitions to SHIPPED (includes trackingNumber, carrier, estimatedDelivery)
- `order.completed` — published when status transitions to COMPLETED
- `order.cancelled` — published when `cancelOrder()` is called (includes cancelledBy, reason)

Common event envelope format (shared across the platform):
```json
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

Event classes (Java records): `OrderCreatedEvent`, `OrderPaidEvent`, `OrderShippedEvent`, `OrderCompletedEvent`, `OrderCancelledEvent` — all wrapped with `EventEnvelope`.

## Security Architecture

### SecurityConfig (always active)
- CSRF disabled (stateless REST API)
- Security headers: X-Content-Type-Options, X-Frame-Options: DENY, X-XSS-Protection, CSP, Referrer-Policy
- CORS configuration
- Endpoint-level authorization rules

### OAuth2SecurityConfig (active when `oauth2.enabled=true`)
- JWT validation via Keycloak JWKS endpoint
- Role extraction from multiple JWT claim paths: `realm_access.roles`, `resource_access.{client}.roles`, `groups`
- `@PreAuthorize` annotations on controller methods

### RateLimitFilter
- Bucket4j token bucket algorithm
- Per-IP rate limiting
- Caffeine cache for IP → bucket mapping
- Config: requests-per-minute, requests-per-second, burst-capacity from `application.yml`
- Returns 429 Too Many Requests when exceeded

### InputSanitizer
- Sanitizes request inputs to prevent XSS and injection attacks

## DTO Pattern

All request and response DTOs are Java **records** (immutable):
```java
public record CreateOrderRequest(
    @NotNull String customerId,
    @NotEmpty List<OrderItemRequest> items,
    ShippingAddressRequest shippingAddress,
    String currency
) {}
```

Response DTOs (`OrderResponse`) map entity data for API consumers.

## Repository Pattern

```java
@Repository
interface OrderRepository extends JpaRepository<Order, UUID> {
    List<Order> findByCustomerId(String customerId);
    // Custom queries via @Query or method naming
}
```

## Configuration Pattern

All settings in `application.yml` with `${ENV_VAR:default}` notation. Environment variables override YAML values. No Spring Cloud Config; configuration is self-contained.

## Observability

- **Logging**: SLF4J + `@Slf4j`; JSON format in production; includes orderId, customerId, status in log context
- **Metrics**: Micrometer + Prometheus at `/actuator/prometheus`
- **Health**: `/actuator/health` (DB connection), `/actuator/health/liveness`, `/actuator/health/readiness`
- **Correlation**: `correlationId` field in all event envelopes; passed through service method parameters
