# Progress: Order Service

## What's Built

### Core Application
- [x] `Order` entity with full field set (UUID PK, status, items, shipping, all lifecycle timestamps)
- [x] `OrderItem` entity with `@ManyToOne` back-reference and subtotal
- [x] `ShippingAddress` embedded entity
- [x] `OrderStatus` enum: PENDING, PAID, PROCESSING, SHIPPED, COMPLETED, CANCELLED
- [x] `OrderRepository` (Spring Data JPA) with `findByCustomerId`
- [x] `OrderService` — createOrder, getOrder, getOrdersByCustomer, updateOrderStatus, cancelOrder
- [x] State machine enforcement in `validateStatusTransition()` (Java 21 switch expression)
- [x] `OrderEventPublisher` — publishes all 5 lifecycle events to RabbitMQ
- [x] `EventEnvelope` — common event wrapper
- [x] Event records: `OrderCreatedEvent`, `OrderPaidEvent`, `OrderShippedEvent`, `OrderCompletedEvent`, `OrderCancelledEvent`
- [x] `OrderController` — REST endpoints: POST /api/orders, GET /api/orders/{id}, GET /api/orders?customerId=, PATCH /api/orders/{id}/status, POST /api/orders/{id}/cancel
- [x] DTOs (Java records): `CreateOrderRequest`, `UpdateOrderStatusRequest`, `CancelOrderRequest`, `OrderResponse`
- [x] `SecurityConfig` — base security headers, CSRF off, CORS
- [x] `OAuth2SecurityConfig` — Keycloak JWT validation, role extraction
- [x] `RateLimitConfig` + `RateLimitFilter` — Bucket4j per-IP rate limiting
- [x] `InputSanitizer` — request sanitization
- [x] `OrderServiceApplication` — Spring Boot main class

### Testing
- [x] `InputSanitizerTest` — sanitization unit test
- [x] `OAuth2SecurityConfigTest` — OAuth2 config unit test
- [x] `OAuth2SecurityIntegrationTest` — OAuth2 integration test
- [x] `RateLimitFilterTest` — rate limit unit test
- [x] `SecurityConfigTest` — base security unit test
- [x] `TestController` — test helper for security tests

### Infrastructure & Operations
- [x] `pom.xml` — full dependency configuration including Testcontainers BOM
- [x] `application.yml` — complete configuration with environment variable overrides
- [x] Dockerfile — multi-stage build
- [x] Dockerfile.local — local development variant
- [x] Makefile — comprehensive targets including ArgoCD, Kubernetes, native image, Flyway
- [x] k8s/base/deployment.yaml
- [x] k8s/base/service.yaml
- [x] k8s/base/configmap.yaml
- [x] k8s/base/serviceaccount.yaml
- [x] k8s/base/secret.yaml
- [x] k8s/base/namespace.yaml
- [x] k8s/base/hpa.yaml
- [x] k8s/base/kustomization.yaml
- [x] .gitignore
- [x] CI workflow pin updated to `.github/workflows/ci.yml@999f8d7` for multi-arch builds (2026-03-17)

### Documentation
- [x] CLAUDE.md — AI assistant guidance
- [x] README.md — setup and usage guide
- [x] docs/README.md
- [x] docs/api/README.md
- [x] docs/architecture/README.md
- [x] docs/troubleshooting/README.md

## In Flight

- [ ] **PR #17** — `docs/configuration-guide` — configuration guide + Spring Boot refresh how-to; open, Copilot tagged, waiting for review
- [ ] **Fix GitHub Packages 401** — after PR #17 merges; add `packages: read` permission to CI workflow + `settings.xml` auth — tracked in `docs/issues/2026-03-17-ci-github-packages-auth.md`

## What's Pending / Known Gaps

- [ ] **Database schema migration** — no Flyway or Liquibase present; `ddl-auto: validate` will fail on a fresh database with no schema; schema creation script needs to be provided or Flyway added
- [ ] **RabbitMQ consumer** — the service publishes events but there is no visible consumer setup for `cart.checkout` events from the Basket Service (may be planned or handled elsewhere)
- [ ] **rabbitmq-client-java local install** — the `1.0.0-SNAPSHOT` dependency is not in any public Maven repo; build breaks without the local install step
- [ ] **Flyway migrations** — Makefile has `db-migrate` target but Flyway is not in pom.xml dependencies
- [ ] **JAVA_HOME path** — Makefile has Linux/Homebrew path, requires override on other platforms
- [ ] **OrderService unit tests** — no `OrderServiceTest` file visible (only security config tests present); business logic tests may be pending
- [ ] **Integration tests for OrderService/OrderController** — Testcontainers dependencies are in pom.xml but no integration test classes visible
- [ ] Load/performance testing
- [ ] Contract tests (e.g., Pact) for RabbitMQ event schema compatibility

## Recent Work

- 2026-03-17 — Dockerfile uses BuildKit `GH_TOKEN` secret to authenticate to GitHub Packages and copies `checkstyle.xml` alongside the pom (commit `cb663a2`); CI run 23175038080 succeeded.

## API Endpoints Summary

| Method | Path | Description | Status |
|--------|------|-------------|--------|
| POST | `/api/orders` | Create order | Implemented |
| GET | `/api/orders/{orderId}` | Get order by ID | Implemented |
| GET | `/api/orders?customerId=` | List orders by customer | Implemented |
| PATCH | `/api/orders/{orderId}/status` | Update order status | Implemented |
| POST | `/api/orders/{orderId}/cancel` | Cancel order | Implemented |
| GET | `/actuator/health` | Health check | Implemented |
| GET | `/actuator/prometheus` | Prometheus metrics | Implemented |

## Events Published Summary

| Routing Key | Trigger | Status |
|---|---|---|
| `order.created` | `createOrder()` | Implemented |
| `order.paid` | `updateOrderStatus()` → PAID | Implemented |
| `order.shipped` | `updateOrderStatus()` → SHIPPED | Implemented |
| `order.completed` | `updateOrderStatus()` → COMPLETED | Implemented |
| `order.cancelled` | `cancelOrder()` | Implemented |
