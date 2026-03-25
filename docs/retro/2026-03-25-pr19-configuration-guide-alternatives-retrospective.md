# Retrospective — PR #19 (Configuration Guide Alternatives)

**Date:** 2026-03-25
**Milestone:** Extend configuration guide with broker-free config refresh alternatives
**PR:** #19 — merged to main (`aa022a5`)
**Participants:** Claude, Copilot

## What Went Well

- **Copilot caught real bugs** — all 3 findings were accurate: wrong `spring.config.import` type (`file:` vs `configtree:`), wrong ConfigMap key names (`RATE_LIMIT_CAPACITY`/`RATE_LIMIT_REFILL_TOKENS` don't exist in `application.yml`), and missing management-port migration warning for probes/Prometheus
- **`configtree:` is the correct import** — Kubernetes mounts each ConfigMap key as a separate file; `file:` silently resolves nothing; Copilot caught this before it misled anyone
- **Three-option structure is clear** — ConfigMap mount (A), Spring Cloud Kubernetes (B), Kafka (C) each have distinct trade-offs documented inline

## What Went Wrong

- **Merge conflict on every branch** — `docs/next-improvements` had diverged from main because PR #17 (original config guide) merged to main first; the alternatives were added to the branch before main was merged in; resulted in a dirty PR that silently blocked CI
- **Wrong env var names in ConfigMap example** — `RATE_LIMIT_CAPACITY` and `RATE_LIMIT_REFILL_TOKENS` were guessed rather than read from `application.yml`; Copilot caught this; should have read the file first

## Process Rules Added

| Rule | Where |
|---|---|
| Before writing ConfigMap key examples, read `application.yml` to confirm the actual env var names | Config guide template |
| Use `configtree:/path/` (not `file:/path/`) when mounting Kubernetes ConfigMap key files | Config guide template |

## Decisions Made

- **`configtree:` is the canonical import for k8s ConfigMap mounts** — `file:` expects a single `application.yml`/`.properties`; `configtree:` reads one-key-per-file layout; all future how-tos should default to `configtree:`
- **Option B (Spring Cloud Kubernetes) is the recommended no-broker path** — it polls the k8s API directly; no extra infrastructure needed beyond RBAC; cleanest integration in a k8s-native deployment
- **Management port separation requires updating probes** — documented explicitly; if moved, k8s liveness/readiness probes and Prometheus scrape must also be updated or health checks break

## Theme

A clean docs-only PR that extended the configuration guide with three broker-free config refresh alternatives. The main friction was a recurring merge conflict pattern (branch diverges from main between PR merges) that silently kills CI — the fix is always the same: merge main into the branch, resolve CHANGELOG/activeContext conflicts, push. Copilot caught three genuine doc inaccuracies including a wrong Spring Boot import type that would have caused silent failures for anyone following the how-to. All three fixes were straightforward one-liners.
