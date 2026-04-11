# Copilot PR #24 Review Findings

**Date:** 2026-04-11
**PR:** #24 — fix(deps): bump rabbitmq-client to 1.0.1; remove RabbitHealthConfig workaround
**Reviewer:** Copilot

## Finding 1 — Stale image tag in kustomization.yaml

**File:** `k8s/base/kustomization.yaml` line 30
**Flagged:** `newTag: 2026-04-11-rabbit-health` — a manually-pushed tag containing the NPE
bug. Merging with this tag would cause ArgoCD to deploy the broken image in the window before
CI publishes the fixed `sha-<merge-sha>` tag.
**Fix:** Reverted to `newTag: latest`; the post-merge `publish` CI job is the sole mutator
of this field.
**Root cause:** Workaround commit `c813340` changed the tag manually and was included in the
branch; the spec omitted `k8s/base/kustomization.yaml` from the "What NOT to Do" list.
**Process note:** Specs that target repos with CI-managed kustomization tags must explicitly
list `k8s/base/kustomization.yaml` under "What NOT to Do."

## Finding 2 — Stale `### Fixed` CHANGELOG entry

**File:** `CHANGELOG.md` line 10
**Flagged:** Bullet describing the `RabbitHealthConfig` guarded health indicator — code
deleted by this PR. The entry was inaccurate after the workaround was removed.
**Fix:** Removed the stale bullet; the remaining `### Fixed` entries are still accurate.
**Root cause:** Entry written when the workaround was added; not cleaned up when the
workaround was deleted.

## Finding 3 — Dangling word in `### Changed` bullet

**File:** `CHANGELOG.md` line 17
**Flagged:** "remove" left as the last word of a line with no grammatical object, making the
sentence awkward to read.
**Fix:** Changed "remove" → "remove the" so the phrase reads as a complete sentence across
the line break.
**Root cause:** Line-wrap introduced during spec writing.
