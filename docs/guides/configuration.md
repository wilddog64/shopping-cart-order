# Configuration Guide — order-service

All configuration is supplied via environment variables. The service reads them at startup
from `application.yml` using Spring `${VAR:default}` placeholders. No runtime reload is
wired up — a restart is required to pick up config changes (see
[How to enable config auto-refresh](#how-to-enable-config-auto-refresh) below).

---

## Environment Variables

### Database

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `orders` | Database name |
| `DB_USERNAME` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password — use Vault ESO in cluster |

Schema is managed by Flyway. `ddl-auto: validate` — Hibernate validates against the schema
but never modifies it.

### Server

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP listener port |

Request size limits are hardcoded: 8KB max header, 2MB max form/body.

### Rate Limiting

| Variable | Default | Description |
|---|---|---|
| `RATE_LIMIT_PER_MINUTE` | `100` | Max requests per minute per client |
| `RATE_LIMIT_PER_SECOND` | `20` | Max requests per second per client |
| `RATE_LIMIT_BURST` | `50` | Burst capacity |

Backed by `RateLimitConfig` (`@ConfigurationProperties(prefix = "rate-limit")`). Changes
require a restart — `RateLimitConfig` is not annotated `@RefreshScope`.

### RabbitMQ

| Variable | Default | Description |
|---|---|---|
| `RABBITMQ_HOST` | `localhost` | Broker host |
| `RABBITMQ_PORT` | `5672` | Broker port |
| `RABBITMQ_VHOST` | `/` | Virtual host |
| `RABBITMQ_USE_TLS` | `false` | Enable TLS for broker connection |
| `RABBITMQ_USERNAME` | `guest` | Static credentials (when Vault disabled) |
| `RABBITMQ_PASSWORD` | `guest` | Static credentials (when Vault disabled) |
| `RABBITMQ_POOL_SIZE` | `10` | Channel pool size |
| `RABBITMQ_PREFETCH_COUNT` | `10` | Consumer prefetch count |

**Retry:**

| Variable | Default | Description |
|---|---|---|
| `RABBITMQ_RETRY_MAX_ATTEMPTS` | `3` | Max publish retry attempts |
| `RABBITMQ_RETRY_INITIAL_INTERVAL` | `1000` | Initial retry backoff (ms) |
| `RABBITMQ_RETRY_MAX_INTERVAL` | `30000` | Max retry backoff (ms) |
| `RABBITMQ_RETRY_MULTIPLIER` | `2.0` | Backoff multiplier |

**Circuit breaker:**

| Variable | Default | Description |
|---|---|---|
| `RABBITMQ_CIRCUIT_BREAKER_ENABLED` | `true` | Enable circuit breaker |
| `RABBITMQ_CIRCUIT_BREAKER_FAILURE_THRESHOLD` | `5` | Open circuit after N consecutive failures |
| `RABBITMQ_CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | `2` | Close circuit after N consecutive successes |
| `RABBITMQ_CIRCUIT_BREAKER_TIMEOUT` | `30000` | Half-open retry timeout (ms) |

### Vault (RabbitMQ dynamic credentials)

When `VAULT_ENABLED=true`, RabbitMQ credentials are fetched from Vault at startup instead
of using the static `RABBITMQ_USERNAME` / `RABBITMQ_PASSWORD` values.

| Variable | Default | Description |
|---|---|---|
| `VAULT_ENABLED` | `false` | Enable Vault credential fetch |
| `VAULT_ADDR` | `http://localhost:8200` | Vault server address |
| `VAULT_ROLE` | `order-publisher` | Vault role for RabbitMQ credentials |
| `VAULT_RABBITMQ_PATH` | `rabbitmq` | Vault secrets engine mount path |

When enabled, the service fetches RabbitMQ credentials from Vault at startup using a
Kubernetes ServiceAccount token for authentication. The static `RABBITMQ_USERNAME` /
`RABBITMQ_PASSWORD` values are ignored when `VAULT_ENABLED=true`. The k8s manifests
currently deploy with `VAULT_ENABLED=false` and static credentials.

### OAuth2 / Keycloak

| Variable | Default | Description |
|---|---|---|
| `OAUTH2_ENABLED` | `false` | Enable JWT bearer token validation |
| `OAUTH2_ISSUER_URI` | `http://keycloak.identity.svc.cluster.local/realms/shopping-cart` | Keycloak issuer URI |
| `OAUTH2_JWK_SET_URI` | `.../protocol/openid-connect/certs` | JWK set endpoint for JWT verification |

When `OAUTH2_ENABLED=false`, API endpoints under `/api/**` are accessible without
authentication, but all other paths remain blocked except for the explicitly permitted
actuator endpoints. Set to `true` in staging/prod with the correct Keycloak issuer URI.

### Logging

| Variable | Default | Description |
|---|---|---|
| `SECURITY_LOG_LEVEL` | `INFO` | Log level for `org.springframework.security` |
| `LOGGING_LEVEL_ROOT` | `INFO` | Root logger level (Spring Boot standard override) |
| `LOGGING_LEVEL_COM_SHOPPINGCART` | `DEBUG` | Log level for `com.shoppingcart.order` (Spring Boot standard override) |

The `LOGGING_LEVEL_*` variables follow the standard Spring Boot externalized config pattern
and override the defaults set in `application.yml`.

---

## Actuator Endpoints

The following management endpoints are exposed by Spring Boot Actuator at `/actuator/*`:

| Endpoint | Path | Reachable by default |
|---|---|---|
| Health (with details for authorized users) | `/actuator/health` | ✅ |
| Info | `/actuator/info` | ✅ |
| Prometheus scrape | `/actuator/prometheus` | ✅ |
| Metrics | `/actuator/metrics` | ❌ exposed by Actuator but blocked by `SecurityConfig` |

With the default security configuration, only `health`, `info`, and `prometheus` are
reachable. `/actuator/metrics` is exposed by Actuator but denied by `SecurityConfig` unless
you explicitly permit it. `/actuator/refresh` is **not** exposed — see below.

---

## How to Enable Config Auto-Refresh

The service currently reads all configuration at startup. To support live config reload
without a pod restart, add Spring Cloud Config Bus backed by RabbitMQ (already present in
the cluster).

### What this enables

- `/actuator/refresh` triggers a `@RefreshScope` bean reload on the local pod
- A Spring Cloud Bus event broadcasts the refresh to all running pods simultaneously
- Useful for changing rate limits, log levels, or feature flags without rolling restarts

### Step 1 — Add dependencies to `pom.xml`

```xml
<!-- Spring Cloud BOM — pin to the version matching your Spring Boot version -->
<dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-bus-amqp</artifactId>
</dependency>
```

Add the BOM to `<dependencyManagement>`:

```xml
<dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>org.springframework.cloud</groupId>
      <artifactId>spring-cloud-dependencies</artifactId>
      <version>2023.0.1</version>   <!-- match Spring Boot 3.x -->
      <type>pom</type>
      <scope>import</scope>
    </dependency>
  </dependencies>
</dependencyManagement>
```

### Step 2 — Expose the refresh endpoint in `application.yml`

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health,info,metrics,prometheus,refresh,busrefresh
```

### Step 3 — Annotate beans that hold refreshable config

```java
import org.springframework.cloud.context.config.annotation.RefreshScope;

@RefreshScope
@ConfigurationProperties(prefix = "rate-limit")
public class RateLimitConfig {
    // existing fields unchanged
}
```

`@Value`-annotated beans need `@RefreshScope` on the enclosing `@Component` or `@Bean`.

### Step 4 — Trigger a refresh

**Single pod (manual):**
```bash
curl -X POST http://<pod-ip>:8080/actuator/refresh
```

**All pods via Bus (requires `spring-cloud-starter-bus-amqp`):**
```bash
curl -X POST http://<any-pod>:8080/actuator/busrefresh
```

Spring Cloud Bus uses Spring AMQP with its own `spring.rabbitmq.*` connection factory
(including credentials and TLS). It will **not** automatically reuse the custom `rabbitmq.*`
configuration or the existing `rabbitmq-client` connection, but it talks to the same broker.
You must add `spring.rabbitmq.*` properties (host, port, credentials) alongside the existing
`rabbitmq.*` config — no separate broker needed.

### What does NOT auto-refresh

- `spring.datasource.*` — datasource is created once at startup; changing DB credentials
  requires a restart (or Vault lease rotation via ESO, which already handles this)
- `server.port` — cannot change without restart
- Beans not annotated `@RefreshScope`

### Security note

`/actuator/refresh` and `/actuator/busrefresh` must be protected. Add them to the
management security config or expose only on the internal management port:

```yaml
management:
  server:
    port: 8081   # separate port, not exposed via Kubernetes Service
```

> **Note:** If you move Actuator to a separate management port you **must** also update:
> - Kubernetes liveness/readiness probes (currently target `/actuator/health` on port 8080)
> - Prometheus scrape config (currently targets `/actuator/prometheus` on port 8080)
>
> Otherwise health checks and metrics scraping will fail after the port change.

---

## Alternatives: config refresh without RabbitMQ

The Spring Cloud Bus approach above requires a running RabbitMQ broker. Three alternatives
exist if a broker is unavailable or undesirable.

### Option A — Kubernetes ConfigMap mount (no broker required)

Store environment variables in a ConfigMap and mount it as a file into the pod. Kubernetes
updates the mounted file in-place (eventually consistent, typically within 60 s) without
restarting the pod.

**1. Create the ConfigMap:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: order-service-config
  namespace: order-service
data:
  RATE_LIMIT_PER_MINUTE: "100"
  RATE_LIMIT_PER_SECOND: "5"
  RATE_LIMIT_BURST: "20"
```

**2. Mount it in the Deployment:**
```yaml
spec:
  containers:
    - name: order-service
      envFrom:
        - configMapRef:
            name: order-service-config   # injected as env vars at startup
      volumeMounts:
        - name: config-vol
          mountPath: /config             # mounted as files for live reload
  volumes:
    - name: config-vol
      configMap:
        name: order-service-config
```

**3. Point Spring Boot at the mounted path:**
```yaml
spring:
  config:
    import: "optional:configtree:/config/"
```

> Kubernetes mounts each ConfigMap key as a separate file (e.g. `/config/RATE_LIMIT_PER_MINUTE`).
> Spring Boot's `configtree` import reads this layout correctly — `file:` import expects a single
> `application.yml`/`.properties` file and will silently resolve nothing against individual key files.

**Limitation:** Spring Boot does not watch mounted files by default. The mounted file updates
automatically, but the running JVM still holds the old values. You must either:
- Restart the pod (simplest — ArgoCD/Helm rollout on ConfigMap hash change), or
- Add `spring-cloud-starter-kubernetes-client-config` (see Option B).

### Option B — Spring Cloud Kubernetes Config (no broker, live reload via k8s API)

`spring-cloud-starter-kubernetes-client-config` watches ConfigMaps and Secrets directly
via the Kubernetes API and triggers `@RefreshScope` beans without any message broker.

**Dependency:**
```xml
<dependency>
  <groupId>org.springframework.cloud</groupId>
  <artifactId>spring-cloud-starter-kubernetes-client-config</artifactId>
</dependency>
```

**Enable polling in `application.yml`:**
```yaml
spring:
  cloud:
    kubernetes:
      config:
        enabled: true
        name: order-service-config
        namespace: order-service
      reload:
        enabled: true
        strategy: refresh          # or: restart_context | shutdown
        mode: polling              # or: event (requires watch RBAC)
        period: 15000              # poll every 15 s
```

**RBAC** — the pod's ServiceAccount needs `get`/`list`/`watch` on `configmaps` in its namespace.
The `restart_context` strategy is safer than `refresh` if beans are not annotated `@RefreshScope`.

**When to use:** running on Kubernetes and a message broker is not available or is overkill
for the refresh use case alone.

### Option C — Kafka broker (drop-in swap for Spring Cloud Bus)

If Kafka is already in the stack, replace the RabbitMQ bus dependency with the Kafka variant.
The rest of the setup (Steps 2–4 above) is identical.

```xml
<!-- remove: spring-cloud-starter-bus-amqp -->
<dependency>
  <groupId>org.springframework.cloud</groupId>
  <artifactId>spring-cloud-starter-bus-kafka</artifactId>
</dependency>
```

Configure the Kafka broker:
```yaml
spring:
  kafka:
    bootstrap-servers: ${KAFKA_BOOTSTRAP_SERVERS:localhost:9092}
```

`/actuator/busrefresh` works the same way — it publishes a refresh event to the Kafka topic
`springCloudBus`, and all subscribed instances pick it up.
