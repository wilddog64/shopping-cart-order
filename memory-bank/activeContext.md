# Active Context: Order Service

## Current Status (2026-03-14)

Branch protection active. `feature/p4-linter` (PR [#2](https://github.com/wilddog64/shopping-cart-order/pull/2)) now runs Checkstyle + OWASP in CI, but the latest build (run `23096292362`) fails because OWASP Dependency Check cannot download NVD data without an API key.

## What's Implemented

- Order lifecycle: PENDING→PAID→PROCESSING→SHIPPED→COMPLETED/CANCELLED
- RabbitMQ event publishing for all 5 order lifecycle events
- JWT auth via Keycloak OAuth2 Resource Server
- Rate limiting (Bucket4j + Caffeine), input sanitization, security headers
- Testcontainers integration test infrastructure
- `com.shoppingcart:rabbitmq-client:1.0.0-SNAPSHOT` resolved from GitHub Packages via PACKAGES_TOKEN

## CI History

- **fix/ci-stabilization PR #1** — merged 2026-03-14. Fixed: GitHub Packages repo + PACKAGES_TOKEN auth for rabbitmq-client dependency.
- **Branch protection** — 1 review + CI required, enforce_admins: false

## Active Task

- **P4 linter** — Checkstyle + OWASP. Spec: `wilddog64/shopping-cart-infra/docs/plans/p4-linter-order.md`. Branch `feature/p4-linter`, PR #2 open; Checkstyle job passes but OWASP dependency-check fails (run `23096292362`) because the repository lacks an `NVD_API_KEY` secret. See `docs/issues/2026-03-14-owasp-nvd-api-key.md` for remediation.

## Agent Rules (Codex must follow)

1. Read the spec at `wilddog64/shopping-cart-infra/docs/plans/p4-linter-order.md` before touching any code.
2. Use CI to verify — do NOT run `mvn` locally (local Java 25 vs pom Java 21 causes timeouts).
3. Do NOT update `memory-bank/activeContext.md` until `gh run list --repo wilddog64/shopping-cart-order` shows `completed success`.
4. Verify commit SHA with `gh api repos/wilddog64/shopping-cart-order/commits/<sha>` before reporting.
5. Open a PR when CI is green; do NOT merge it yourself.

## Key Notes

- `ddl-auto: validate` — no Flyway/Liquibase. Schema must be created manually before first run.
- Makefile hardcodes `JAVA_HOME=/home/linuxbrew/...` — override for macOS/ARM64.
- rabbitmq-client pinned to `1.0.0-SNAPSHOT` — never use in production without stable release.
