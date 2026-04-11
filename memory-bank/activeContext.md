# Active Context: Order Service

## Current Status (2026-04-11)

**PR #25 MERGED** — `49ff6b87` 2026-04-11 — fix trivy-action version (`999f8d70`→`39c3072`); docs/next-improvements catch-up. 2 Copilot findings fixed. Branch protection updated: replaced stale `"CI"` status check context with `Build & Test` + `Checkstyle`. `enforce_admins` restored.
**PR #24 MERGED** — `7f0ea87e` 2026-04-11 — bump rabbitmq-client to 1.0.1; delete RabbitHealthConfig workaround; 3 Copilot findings fixed. `enforce_admins` restored.
**PR #19 MERGED** — `aa022a5` 2026-03-25 — configuration guide alternatives. Copilot 3 findings fixed. `enforce_admins` restored.
**Active branch:** `docs/next-improvements-2`
**rabbitmq-client-java:** v1.0.1 released; JAR on GitHub Packages.

---

## Current Status (2026-03-14)

CI green. All PRs merged to main. Branch protection active.

## What's Implemented

- Order lifecycle: PENDING→PAID→PROCESSING→SHIPPED→COMPLETED/CANCELLED
- RabbitMQ event publishing for all 5 order lifecycle events
- JWT auth via Keycloak OAuth2 Resource Server
- Rate limiting (Bucket4j + Caffeine), input sanitization, security headers
- Testcontainers integration test infrastructure
- GitHub Actions CI: Checkstyle + OWASP dependency-check gate + build/test + ghcr.io push

## CI History

- **fix/ci-stabilization PR #1** — merged 2026-03-14. Fixed: GitHub Packages repo + PACKAGES_TOKEN auth.
- **feature/p4-linter PR #2** — merged 2026-03-14. Added Checkstyle + OWASP (`failOnError=false` — NVD_API_KEY not set).
- **Branch protection** — 1 review + CI required, enforce_admins: false

## Active Task

- **Multi-arch workflow pin** — branch `fix/multiarch-workflow-pin` updates `.github/workflows/ci.yml` to use infra SHA `999f8d7` so CI pushes amd64+arm64 images.
- **CI follow-up — GitHub Packages auth** — Dockerfile now mounts `GH_TOKEN` secret and copies `checkstyle.xml` (commit `cb663a2`). Latest CI run 23175038080 succeeded; release work remains.

## OWASP Note (DO NOT REVERT)

`pom.xml` has `<failOnError>false</failOnError>` in the OWASP plugin. Do not remove — `NVD_API_KEY` secret is not set; without this flag the build crashes with a 403. Add `NVD_API_KEY` to repo secrets to get live CVE data.

## Agent Instructions

Rules that apply to ALL agents working in this repo:

1. **CI only** — do NOT run `mvn` locally (local Java 25 vs pom Java 21 causes timeouts).
2. **Memory-bank discipline** — do NOT update `memory-bank/activeContext.md` until CI shows `completed success`.
3. **SHA verification** — verify commit SHA before reporting.
4. **Do NOT merge PRs** — open the PR and stop.
5. **OWASP `failOnError=false` — DO NOT REMOVE.**

## Key Notes

- `ddl-auto: validate` — schema must be created manually before first run
- `rabbitmq-client` pinned to `1.0.0-SNAPSHOT` — never use in production without stable release
