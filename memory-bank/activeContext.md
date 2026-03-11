# Active Context: Order Service

## Current Status

Active development — the core order lifecycle is fully implemented with a complete set of unit and integration tests. The service is structured and ready for Kubernetes deployment via ArgoCD.

## Implemented Features

- Full order CRUD: create, get by ID, list by customer
- Order status lifecycle with state machine enforcement (PENDING → PAID → PROCESSING → SHIPPED → COMPLETED / CANCELLED)
- Timestamp tracking for each transition (paidAt, shippedAt, completedAt, cancelledAt)
- Tracking number and carrier storage on shipment
- RabbitMQ event publishing for all 5 order lifecycle events
- JWT authentication via Keycloak OAuth2 Resource Server (optional)
- Rate limiting with Bucket4j + Caffeine
- Input sanitization
- Security headers
- Spring Actuator: health, metrics, Prometheus
- Unit tests for security config (OAuth2, SecurityConfig, RateLimitFilter, InputSanitizer)
- Integration test infrastructure with Testcontainers

## Active Development Notes

### Critical Dependency: rabbitmq-client-java
The `com.shoppingcart:rabbitmq-client:1.0.0-SNAPSHOT` dependency is a local library not published to any Maven repository. The pom.xml TODO comment confirms this:
```xml
<!-- TODO: Replace with Maven coordinates once published -->
```
Anyone building this project must first: `cd ../rabbitmq-client-java && mvn install`

### Database Schema
The service uses `ddl-auto: validate` — Hibernate validates the schema but does NOT create it. There is no Flyway or Liquibase migration in this service (unlike the Payment Service which has Flyway). Schema must be created manually or via a separate migration tool. This is a known gap.

### JAVA_HOME in Makefile
The Makefile hardcodes `JAVA_HOME=/home/linuxbrew/.linuxbrew/opt/openjdk@21`. This is a Linux/Homebrew path that will not work on macOS (Apple Silicon) or standard Linux environments. Override with:
```bash
JAVA_HOME=/path/to/your/java21 make build
```

## Integration Points

- **Basket Service**: Consumes `cart.checkout` events from RabbitMQ (consumer setup not visible in current code — may be pending or handled at infra level)
- **Payment Service**: Publishes `order.paid` event after payment confirmation; may consume `payment.*` events
- **RabbitMQ**: Exchange `events`; routing keys `order.*`
- **PostgreSQL**: Database `orders`; K8s service `postgresql-orders.shopping-cart-data.svc.cluster.local`
- **Keycloak**: JWKS at `http://keycloak.identity.svc.cluster.local/realms/shopping-cart/protocol/openid-connect/certs`
- **Vault**: Optional dynamic RabbitMQ credentials at path `rabbitmq/creds/order-publisher`

## Kubernetes Deployment Target

- Namespace: `shopping-cart-apps`
- ArgoCD application name: `order-service`
- K8s service: `order-service` (ClusterIP, port 80 → 8080)
- HPA configured (k8s/base/hpa.yaml)
- Secret manifest (k8s/base/secret.yaml) — requires population with real values before deployment

## Recent Test Coverage

Tests present in `src/test/java/com/shoppingcart/order/config/`:
- `InputSanitizerTest` — request sanitization
- `OAuth2SecurityConfigTest` — OAuth2 configuration
- `OAuth2SecurityIntegrationTest` — full OAuth2 integration test
- `RateLimitFilterTest` — rate limiting behavior
- `SecurityConfigTest` — base security configuration
- `TestController` — helper controller for security tests

## CI Blocker — OPEN (2026-03-11)

**Failing since:** 2026-03-09
**Failing step:** `Build & Test` → `Build with Maven`

**Error:**
```
[ERROR] Could not resolve dependencies for project com.shoppingcart:shopping-cart-order
[ERROR] dependency: com.shoppingcart:rabbitmq-client:jar:1.0.0-SNAPSHOT
[ERROR] Could not find artifact com.shoppingcart:rabbitmq-client:jar:1.0.0-SNAPSHOT
```

**Root cause:** `rabbitmq-client-java` is declared as a Maven dependency but never published to any accessible repository. CI has no way to resolve it.

**Fix options (pick one):**
1. Publish `rabbitmq-client-java` to GitHub Packages and configure CI to install from there
2. Restructure as a multi-module Maven project (parent pom + modules)
3. Add a CI step that builds and installs `rabbitmq-client-java` before building the order service

**Note from CLAUDE.md:** `rabbitmq-client-java` version must stay pinned — never use `-SNAPSHOT` in production.

**Priority:** P2 — assigned to v0.8.0 milestone. See `k3d-manager/docs/issues/2026-03-11-shopping-cart-ci-failures.md`.
