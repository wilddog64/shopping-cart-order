# Retrospective — PR #17 (Configuration Guide)

**Date:** 2026-03-25
**Milestone:** Add configuration guide + Spring Boot config auto-refresh how-to
**PR:** #17 — merged to main (`c68757d`)
**Participants:** Claude, Copilot

## What Went Well

- **Copilot findings were all substantive** — all 7 findings were real accuracy problems, not style nits. The doc would have actively misled readers without them (wrong defaults, inaccurate security posture, false claim about Bus reusing the existing connection)
- **Spring Cloud Bus connection finding was the most valuable** — the service uses a custom `rabbitmq-client` library with `rabbitmq.*` properties, while Spring Cloud Bus AMQP uses Spring AMQP and `spring.rabbitmq.*`. Copilot correctly flagged that they don't share a connection factory. This would have caused a confusing failure if someone followed the how-to without the fix
- **Doc content sourced directly from code** — env var table was built from `application.yml` line-by-line; no guessing

## What Went Wrong

- **OAUTH2_JWK_SET_URI default wrong in first draft** — wrote `.well-known/openid-connect/certs` instead of `protocol/openid-connect/certs`; easily avoided by copy-pasting the default from `application.yml` rather than abbreviating
- **OAuth2 disabled security posture oversimplified** — "all endpoints are open" is incorrect; SecurityConfig still restricts non-`/api/**` paths. Rule: always cross-check security claims against the actual `SecurityConfig` class, not just `application.yml`
- **Actuator metrics reachability not checked against SecurityConfig** — listed `/actuator/metrics` as exposed without verifying the security layer. Actuator exposure ≠ security reachability
- **Merge conflict on CHANGELOG** — branch was cut from `63cf4ce` but main had `d109004` (a k8s manifest fix PR that landed between branch cut and PR creation); resolved cleanly but added a commit
- **CI always red** — pre-existing GitHub Packages 401 means CI signal is useless for this repo; next branch (`fix/ci-github-packages-auth`) addresses this

## Process Rules Added

| Rule | Where |
|---|---|
| When documenting env var defaults, copy-paste from the source file — never abbreviate or paraphrase | Doc authoring checklist |
| Security posture claims must be cross-checked against `SecurityConfig`, not just `application.yml` | Doc authoring checklist |
| Actuator endpoint reachability ≠ actuator exposure — verify both layers (Actuator `include` + SecurityConfig `permitAll`) | Doc authoring checklist |

## Decisions Made

- **Next branch is `fix/ci-github-packages-auth`** — the always-red CI is the highest-priority follow-up; docs-only PRs can't be verified clean until CI works
- **No version bump for this PR** — docs-only change stays in `[Unreleased]`

## Theme

A docs-only PR that turned out to need 7 accuracy fixes before it was safe to merge. The pattern: writing a reference doc from memory (or `application.yml` in isolation) produces plausible-sounding but wrong details — wrong defaults, oversimplified security descriptions, and a how-to section that would have broken if followed literally. Copilot's value here was specifically in cross-checking the doc against code the author didn't read (`SecurityConfig`, the custom rabbitmq-client library). The lesson for future reference docs: always read the implementation files, not just the config files.
