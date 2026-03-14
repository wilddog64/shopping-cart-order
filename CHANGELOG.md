# Changelog

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
