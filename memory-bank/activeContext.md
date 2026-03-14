## CI Status (as of 2026-03-13)

**Branch:** `fix/ci-stabilization` — PR #1 open

| Job | Status |
|---|---|
| Build & Test | ❌ fail |

**Failure:** `Could not find artifact com.shoppingcart:rabbitmq-client:jar:1.0.0-SNAPSHOT in github-rabbitmq-client`

maven-settings.xml and `-s` flag are already in place. Root cause: the build job is
missing `packages: read` permission — `GITHUB_TOKEN` cannot read packages from another
repo (`wilddog64/rabbitmq-client-java`) without this explicit permission declaration.

**Round 3 fix required (spec: `wilddog64/shopping-cart-infra` → `docs/plans/ci-stabilization-round3.md` @ c5797539):**
- `.github/workflows/ci.yml`: add `packages: read` to build job permissions block

---# Active Context: Order Service

## Current Status (2026-03-14)

CI green. PR #1 merged to main. Branch protection active.

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

- **P4 linter** — Checkstyle + OWASP. Spec: `wilddog64/shopping-cart-infra/docs/plans/p4-linter-order.md`. Branch: `feature/p4-linter`. Not started.

## Key Notes

- `ddl-auto: validate` — no Flyway/Liquibase. Schema must be created manually before first run.
- Makefile hardcodes `JAVA_HOME=/home/linuxbrew/...` — override for macOS/ARM64.
- rabbitmq-client pinned to `1.0.0-SNAPSHOT` — never use in production without stable release.
