# Tech Context: Order Service

## Language & Runtime

- **Java 21** (LTS) — uses modern Java features: records, switch expressions, text blocks
- Maven artifact: `com.shoppingcart:shopping-cart-order:1.0.0-SNAPSHOT`

## Framework & Core Dependencies (pom.xml)

| Dependency | Version | Purpose |
|---|---|---|
| spring-boot-starter-parent | 3.2.0 | Framework BOM and plugin management |
| spring-boot-starter-web | (managed) | REST API / embedded Tomcat |
| spring-boot-starter-data-jpa | (managed) | Hibernate + Spring Data |
| spring-boot-starter-validation | (managed) | Jakarta Bean Validation |
| spring-boot-starter-actuator | (managed) | Health, metrics, Prometheus |
| spring-boot-starter-security | (managed) | Spring Security filter chain |
| spring-boot-starter-oauth2-resource-server | (managed) | JWT validation via Keycloak JWKS |
| com.shoppingcart:rabbitmq-client | 1.0.0-SNAPSHOT | RabbitMQ publishing (local library) |
| org.postgresql:postgresql | (managed) | PostgreSQL JDBC driver |
| io.micrometer:micrometer-registry-prometheus | (managed) | Prometheus metrics export |
| com.bucket4j:bucket4j-core | 8.7.0 | Rate limiting |
| com.github.ben-manes.caffeine:caffeine | (managed) | In-memory cache for rate limit buckets |
| org.projectlombok:lombok | (managed) | `@Slf4j`, `@RequiredArgsConstructor`, etc. |

## Test Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| spring-boot-starter-test | (managed) | JUnit 5, Mockito, AssertJ |
| spring-security-test | (managed) | Security context mocking |
| com.h2database:h2 | (managed) | In-memory DB for unit tests |
| org.testcontainers:postgresql | 1.19.3 | Real PostgreSQL for integration tests |
| org.testcontainers:rabbitmq | 1.19.3 | Real RabbitMQ for integration tests |
| org.testcontainers:junit-jupiter | 1.19.3 | JUnit 5 Testcontainers integration |

## Infrastructure Dependencies

| Service | Required | Default | Purpose |
|---|---|---|---|
| PostgreSQL 15+ | Yes | localhost:5432/orders | Order persistence |
| RabbitMQ 3.12+ | Yes | localhost:5672 | Event publishing |
| Keycloak | Optional | — | JWT issuer / JWKS |
| HashiCorp Vault | Optional | localhost:8200 | Dynamic RabbitMQ credentials |

## Development Environment Setup

### Prerequisites

- Java 21 (JDK), Maven 3.9+
- PostgreSQL 15+ accessible
- RabbitMQ accessible
- `rabbitmq-client-java` library installed to local Maven repo

### Local Setup

```bash
# 1. Install the local RabbitMQ client library (if not published to a Maven repo)
cd ../rabbitmq-client-java
mvn install
cd ../shopping-cart-order

# 2. Start dependencies (option: Docker Compose)
docker-compose up -d  # if docker-compose.yml exists

# 3. Build
mvn clean package -DskipTests

# 4. Run
mvn spring-boot:run
# Or with dev profile:
mvn spring-boot:run -Dspring-boot.run.profiles=dev
```

### Environment Variables

```
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_NAME=orders
DB_USERNAME=postgres
DB_PASSWORD=postgres
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_VHOST=/
RABBITMQ_USERNAME=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_USE_TLS=false
VAULT_ENABLED=false
VAULT_ADDR=http://localhost:8200
VAULT_ROLE=order-publisher
OAUTH2_ENABLED=false
OAUTH2_ISSUER_URI=http://keycloak.identity.svc.cluster.local/realms/shopping-cart
OAUTH2_JWK_SET_URI=<issuer>/protocol/openid-connect/certs
RATE_LIMIT_PER_MINUTE=100
RATE_LIMIT_PER_SECOND=20
RATE_LIMIT_BURST=50
```

## Application Configuration

Main config file: `src/main/resources/application.yml`
- JPA `ddl-auto: validate` — schema is NOT auto-generated; must be created externally
- Jackson: ISO-8601 dates (not timestamps), ignore unknown properties
- Max request header size: 8KB; max form post: 2MB (security hardening)
- Actuator exposes: health, info, metrics, prometheus

## Build Tooling

- **Makefile** — primary developer interface; `make help` shows all targets
- **Maven wrapper** not present (uses system Maven); JAVA_HOME set in Makefile to `/home/linuxbrew/.linuxbrew/opt/openjdk@21` — adjust for your environment
- **Dockerfile** — multi-stage build
- **Dockerfile.local** — local development variant

## Kubernetes & GitOps

- Manifests in `k8s/base/` managed with Kustomize
- Resources: deployment.yaml, service.yaml, configmap.yaml, serviceaccount.yaml, secret.yaml, namespace.yaml, hpa.yaml, kustomization.yaml
- Namespace: `shopping-cart-apps`
- ArgoCD application name: `order-service`
- Makefile has full ArgoCD target set: status, sync, refresh, diff, history

## Testing Infrastructure

- Unit tests: H2 in-memory database, no Docker required, `mvn test`
- Integration tests: Testcontainers spins up real PostgreSQL + RabbitMQ Docker containers; requires Docker daemon, `mvn verify -Pintegration`
- Security tests run as standard unit tests (no infrastructure needed)
