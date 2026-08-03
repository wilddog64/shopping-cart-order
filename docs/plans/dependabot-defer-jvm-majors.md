# Defer JVM/maven major bumps in Dependabot

**Repo:** shopping-cart-order
**File:** `.github/dependabot.yml`
**Date:** 2026-08-03

## Problem
The weekly Dependabot run opened several breaking majors that fail CI or need a dedicated
migration:
- #45 `spring-boot-starter-parent 3.2.0 → 4.1.0` — Spring Boot 4 framework migration
- #43 `testcontainers-bom 1.19.3 → 2.0.5` — major
- #44 `owasp:dependency-check-maven 9.0.9 → 12.2.2` — major
- #39 `eclipse-temurin 21 → 25-jre-alpine` — JRE major

## Decision (2026-08-03)
Defer **all** maven majors via `dependency-name: "*"` semver-major (mirroring the frontend
npm policy), and defer the docker `eclipse-temurin` major (stay on the 21 LTS line). Minor/patch
still flow (grouped). GitHub Actions majors are merged separately. Close #45, #43, #44, #39.

## Change — `.github/dependabot.yml`
- maven ecosystem: add `- dependency-name: "*"` / `["version-update:semver-major"]`
- docker ecosystem: add `- dependency-name: "eclipse-temurin"` / `["version-update:semver-major"]`

## Definition of Done
- [ ] Dependabot config check green on the PR
- [ ] After merge: `@dependabot close` on #45, #43, #44, #39
