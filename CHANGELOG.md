# Changelog

## [Unreleased]

### Added
- `docs/guides/configuration.md` — full env var reference, actuator endpoints, Spring Cloud Bus config auto-refresh how-to, and three broker-free alternatives (ConfigMap mount, Spring Cloud Kubernetes, Kafka)
- `docs/issues/2026-03-25-rabbitmq-connection-refused.md` — root cause analysis for RabbitMQ CrashLoopBackOff (fixed in shopping-cart-infra PR #22)

### Fixed
- Replace Spring Boot's default RabbitMQ health indicator with a guarded version so /actuator/health stays UP while the rabbitmq-client cache is empty (prevents CrashLoopBackOff until the patched library is released)
- `k8s/base/configmap.yaml`: add `SPRING_RABBITMQ_HOST`, `SPRING_RABBITMQ_PORT`, `SPRING_RABBITMQ_VIRTUAL_HOST` — Spring AMQP auto-configured `CachingConnectionFactory` was defaulting to `localhost:5672`, causing `RabbitHealthIndicator` to return DOWN and the startup probe to fail with 503; custom `ConnectionManager` connected correctly but Spring's health indicator used the auto-configured factory
- CI: switch `lint` and `build` jobs from `PACKAGES_TOKEN` to `GITHUB_TOKEN` in `.github/maven-settings.xml` and `.github/workflows/ci.yml` — resolves 401 when resolving `rabbitmq-client-java` from GitHub Packages; `GITHUB_TOKEN` is automatic and scoped to `packages: read` via workflow permissions
- Align k8s manifests with data-layer: correct DB_PASSWORD, RABBITMQ_PASSWORD, add SPRING_CLOUD_VAULT_ENABLED=false (Vault unreachable from app cluster), reduce resource requests cpu 200m→100m / memory 512Mi→256Mi for t3.medium headroom

### Changed
- Reduce deployment replicas from 2 to 1 for dev/test environment; delete HPA (`minReplicas: 2` was scaling pods back up on single-node cluster); will reintroduce in v1.1.0 EKS
- Bump `rabbitmq-client` dependency from `1.0.0-SNAPSHOT` to `1.0.1`; remove
  `RabbitHealthConfig` workaround — NPE in `ConnectionManager.getStats()` is fixed
  at source in `1.0.1`, eliminating the `CrashLoopBackOff` on pod startup

## [0.1.0] - 2026-03-14

### Added
- Order lifecycle management: PENDING→PAID→PROCESSING→SHIPPED→COMPLETED/CANCELLED
- RabbitMQ event publishing for all 5 order lifecycle transitions
- JWT authentication via Keycloak OAuth2 Resource Server
- Rate limiting (Bucket4j + Caffeine) and input sanitization
- Security headers middleware
- Testcontainers-based integration test infrastructure
- Dockerfile (multi-stage, JRE Alpine, non-root user)
- Kubernetes manifests (Deployment, Service, ConfigMap)
- GitHub Actions CI: Checkstyle + OWASP dependency-check gate + build/test + ghcr.io push
- Branch protection (1 required review + CI status check)
