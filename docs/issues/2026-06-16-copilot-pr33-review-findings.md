# Copilot Review Findings — PR #33 (Go rewrite PR1)

**PR:** https://github.com/wilddog64/shopping-cart-order/pull/33
**Review:** Copilot (submitted 2026-06-16) — 5 inline findings; GitGuardian also flagged a
hardcoded `DB_PASSWORD`.
**Resolution commits:** `66f7e0e` (initial) → `c5a5d95` (config/publisher/not-found hardening)
→ `ac85ad5` (CI integration gate + schema). All threads resolved.

## Fixed

| File:line | Finding | Fix |
|-----------|---------|-----|
| config.go:45 | GitGuardian: hardcoded `DB_PASSWORD` default "postgres" | default → `""` (explicit env required; `DB_USERNAME` left as-is — not a secret) |
| config.go:73 (a) | `sslmode=disable` hardcoded | env-configurable `DBSSLMode` (`DB_SSLMODE`) |
| publisher.go:312 (b) | hand-rolled `net.DialTimeout` AMQP dialer bypasses library TLS | switch to canonical `amqp.DefaultDial(5 * time.Second)` |
| store.go:160 (c) | `Update` ignores command tag — updating a missing order returns `nil` | return `ErrOrderNotFound` when 0 rows affected |

## Deferred to PR2 (authorization)

PR1 runs `OAUTH2_ENABLED=false`; customer identity from the authenticated JWT principal is
PR2 scope. `handler.go` intentionally unchanged:

- **handler.go:98 (d)** — `customerId` taken from request body.
- **handler.go:193 (e)** — list-by-`customerId` query param.

Both replaced by JWT-principal-derived identity in the PR2 (Keycloak JWT) work.

## Process note

Work-repo issue log + README update belongs in the same handoff as the code fix, not a
follow-up backfill.
