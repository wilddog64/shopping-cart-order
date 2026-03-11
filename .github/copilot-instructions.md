# Copilot Instructions — Order Service

## Service Overview

Java 21 / Spring Boot 3.2 microservice handling order processing.
PostgreSQL for persistence, RabbitMQ for event-driven communication.

---

## Architecture Guardrails

### Layer Boundaries — Never Cross These
- **Controller**: HTTP request/response only — no business logic, no direct repository calls
- **Service**: business logic and orchestration — no HTTP concerns
- **Repository**: Spring Data JPA only — no business logic
- A controller must never call a repository directly. Always through the service layer.

### Order State Machine — Valid Transitions Only
```
PENDING → PAID → PROCESSING → SHIPPED → COMPLETED
PENDING → CANCELLED
```
- Never skip a state or go backwards
- Every state transition must be `@Transactional`
- Any code that sets `OrderStatus` directly (bypassing the transition logic) is a bug
- If a requested transition is invalid, throw a domain exception — never silently ignore

### RabbitMQ Ownership
- This service publishes events: `order.created`, `order.paid`, `order.shipped`, `order.completed`, `order.cancelled`
- This service consumes: `cart.checkout` events (triggers order creation)
- Never consume queues owned by other services without an explicit spec
- Always use the common `EventEnvelope` format — never publish raw domain objects
- `correlationId` must be propagated from inbound `cart.checkout` event to outbound `order.created`

### Service Isolation
- This service owns the `orders` database — no other service may connect to it directly
- Never call another service's REST API directly from this service (use events)
- Never import classes from other service repos

---

## Security Rules (treat violations as bugs)

### Secrets (OWASP A02)
- Never hardcode database credentials, RabbitMQ passwords, or any secrets
- All secrets injected via environment variables from ESO/Vault
- Never log credential values — not even partially
- `vault read rabbitmq/creds/order-publisher` output must never appear in code or tests

### Injection (OWASP A03)
- Never build JPQL/SQL by string concatenation — always use Spring Data query methods or `@Query` with named parameters
- Validate all inbound `cart.checkout` event payloads before processing — malformed events must be dead-lettered, not crash the consumer

### Access Control (OWASP A01)
- Order retrieval must always be scoped to the authenticated customer ID
- Never allow one customer to read another customer's orders
- Customer ID comes from JWT claims — never from request body or query params

### Cryptographic Failures (OWASP A02)
- Never disable TLS on RabbitMQ or PostgreSQL connections in non-test code

---

## Code Quality Rules

### Testing
- All new service logic requires JUnit 5 unit tests
- Integration tests use Testcontainers — never mock PostgreSQL in integration tests
- Never delete or comment out existing tests
- Never weaken an assertion
- Run `mvn test` before every commit; must pass clean

### Code Style
- Use records for immutable DTOs and events
- Prefer constructor injection over `@Autowired` field injection
- Add `@Transactional` to every service method that modifies state
- Log at appropriate levels: DEBUG for details, INFO for state transitions, WARN/ERROR for failures
- Never use `System.out.println` in production code

### Dependencies
- Never add a Maven dependency without justification in the PR description
- The `rabbitmq-client-java` library version must stay pinned — never use `-SNAPSHOT` in production

---

## Completion Report Requirements

Before marking any task complete, the agent must provide:
- `mvn test` output (must be clean)
- Confirmation that no test was deleted or weakened
- Confirmation that no credential appears in any changed file
- Confirmation that all state transitions remain valid
- List of exact files modified

---

## What NOT To Do

- Do not refactor code outside the scope of the current task
- Do not add new RabbitMQ event types without updating `shopping-cart-infra` message schemas
- Do not change `OrderStatus` enum values without a migration plan
- Do not add direct REST calls to other services — use events
- Do not use `@Transactional(readOnly = false)` as a workaround for missing transaction boundaries
