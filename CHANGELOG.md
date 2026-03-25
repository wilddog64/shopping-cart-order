# Changelog

## [Unreleased]

### Added
- `docs/guides/configuration.md` — full env var reference (DB, server, rate-limit, RabbitMQ, Vault, OAuth2, logging) + how-to guide for enabling Spring Cloud Bus config auto-refresh

### Fixed
- CI: switch `lint` and `build` jobs from `PACKAGES_TOKEN` to `GITHUB_TOKEN` in `.github/maven-settings.xml` and `.github/workflows/ci.yml` — resolves 401 when resolving `rabbitmq-client-java` from GitHub Packages; `GITHUB_TOKEN` is automatic and scoped to `packages: read` via workflow permissions
- Align k8s manifests with data-layer: correct DB_PASSWORD, RABBITMQ_PASSWORD, add SPRING_CLOUD_VAULT_ENABLED=false (Vault unreachable from app cluster), reduce resource requests cpu 200m→100m / memory 512Mi→256Mi for t3.medium headroom

### Changed
- Reduce deployment replicas from 2 to 1 for dev/test environment; delete HPA (`minReplicas: 2` was scaling pods back up on single-node cluster); will reintroduce in v1.1.0 EKS

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
