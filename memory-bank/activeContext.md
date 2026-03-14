# Active Context: Order Service

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

- **P4 linter** — Checkstyle + OWASP. Branch `feature/p4-linter`, PR #2 open. CI being fixed — see OWASP note below.

## OWASP NVD Fix (DO NOT REVERT)

`pom.xml` has `<failOnError>false</failOnError>` in the OWASP dependency-check config. **Do not remove this line.**

- The `NVD_API_KEY` secret is not set in this repo. Without `failOnError=false`, the build crashes with a 403 when OWASP tries to update from NVD.
- `failOnError=false` makes the scan use cached NVD data instead of crashing. `failBuildOnCVSS=9` still applies — the build will fail on any CVSS ≥9 CVE found in cached data.
- This is the correct behavior for CI without a registered NVD API key. It was set intentionally by the repo owner.
- To get live CVE data: add `NVD_API_KEY` as a GitHub Actions secret. Until then, leave `failOnError=false` in place.

## Agent Rules (Codex must follow)

1. Do NOT touch the OWASP plugin config in pom.xml — `failOnError=false` must stay.
2. Use CI to verify — do NOT run `mvn` locally (local Java 25 vs pom Java 21 causes timeouts).
3. Do NOT update `memory-bank/activeContext.md` until `gh run list --repo wilddog64/shopping-cart-order` shows `completed success`.
4. Verify commit SHA with `gh api repos/wilddog64/shopping-cart-order/commits/<sha>` before reporting.
5. Do NOT merge the PR yourself.

## Key Notes

- `ddl-auto: validate` — no Flyway/Liquibase. Schema must be created manually before first run.
- Makefile hardcodes `JAVA_HOME=/home/linuxbrew/...` — override for macOS/ARM64.
- rabbitmq-client pinned to `1.0.0-SNAPSHOT` — never use in production without stable release.
