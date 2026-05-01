# Changelog

## [Unreleased]

### Added
- `docs/guides/configuration.md` — full env var reference, actuator endpoints, Spring Cloud Bus config auto-refresh how-to, and three broker-free alternatives (ConfigMap mount, Spring Cloud Kubernetes, Kafka)
- `docs/issues/2026-03-25-rabbitmq-connection-refused.md` — root cause analysis for RabbitMQ CrashLoopBackOff (fixed in shopping-cart-infra PR #22)
- `docs/issues/2026-04-11-copilot-pr25-review-findings.md` — stale status date in activeContext.md, inaccurate CHANGELOG entry for README Issue Logs
- README: update Issue Logs to 5 most recent entries — Copilot PR #25 findings, Copilot PR #24 findings, RabbitMQ connection refused, Rate limiting distributed state, Multi-arch workflow pin

### Fixed
- `src/main/java/.../SecurityConfig.java`: add `/actuator/health/**` to `permitAll` requestMatchers — probes hitting `/actuator/health/readiness` and `/actuator/health/liveness` were getting `401 Unauthorized`, keeping pods at `0/1 Ready`
- CI: bump pinned `build-push-deploy.yml` SHA in `publish` job from `999f8d70` (`trivy-action@0.30.0`, invalid) to `39c3072` (`trivy-action@v0.35.0`) — `Build, Scan & Push` was failing on every main push since the Trivy action version didn't exist
- `k8s/base/configmap.yaml`: add `SPRING_RABBITMQ_HOST`, `SPRING_RABBITMQ_PORT`, `SPRING_RABBITMQ_VIRTUAL_HOST` — Spring AMQP auto-configured `CachingConnectionFactory` was defaulting to `localhost:5672`, causing `RabbitHealthIndicator` to return DOWN and the startup probe to fail with 503; custom `ConnectionManager` connected correctly but Spring's health indicator used the auto-configured factory
- CI: switch `lint` and `build` jobs from `PACKAGES_TOKEN` to `GITHUB_TOKEN` in `.github/maven-settings.xml` and `.github/workflows/ci.yml` — resolves 401 when resolving `rabbitmq-client-java` from GitHub Packages; `GITHUB_TOKEN` is automatic and scoped to `packages: read` via workflow permissions
- Align k8s manifests with data-layer: correct DB_PASSWORD, RABBITMQ_PASSWORD, add SPRING_CLOUD_VAULT_ENABLED=false (Vault unreachable from app cluster), reduce resource requests cpu 200m→100m / memory 512Mi→256Mi for t3.medium headroom

### Changed
- Reduce deployment replicas from 2 to 1 for dev/test environment; delete HPA (`minReplicas: 2` was scaling pods back up on single-node cluster); will reintroduce in v1.1.0 EKS
- Bump `rabbitmq-client` dependency from `1.0.0-SNAPSHOT` to `1.0.1`; remove the
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
