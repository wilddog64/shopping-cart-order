# Retrospective — PR #18 (CI GitHub Packages Auth Fix)

**Date:** 2026-03-25
**Milestone:** Fix GitHub Packages 401 in CI — switch to GITHUB_TOKEN
**PR:** #18 — merged to main (`e6739d7`)
**Participants:** Claude, Copilot

## What Went Well

- **Root cause found quickly** — CI error message changed from 401 to "Could not find artifact" after switching to `GITHUB_TOKEN`, which immediately revealed the real issue: private package, not expired token
- **Permanent fix** — making `rabbitmq-client-java` repo + package public eliminates PAT rotation as a recurring concern; `GITHUB_TOKEN` never expires and is auto-provisioned
- **Copilot findings were minor** — only 2 doc wording issues; the code change itself was clean
- **rabbitmq-client-java branch protection confirmed** — already had required CI + 1 PR review; `enforce_admins` added to match standard

## What Went Wrong

- **Wrong diagnosis on first attempt** — assumed 401 was caused by `GITHUB_TOKEN` being insufficient; switched back to `PACKAGES_TOKEN` before realizing the real issue was package visibility
- **Two round-trips to get the root cause** — would have been faster to check package visibility before changing any code
- **Empty commit needed to retrigger CI** — the revert + re-apply left the branch in the same state as the last push; had to push an empty commit to get CI to run again

## Process Rules Added

| Rule | Where |
|---|---|
| Before changing CI auth, check package visibility first: `gh api "user/packages/maven/<name>" --jq '{visibility}'` | CI debugging checklist |
| When a PAT-backed secret fails with 401, check both token expiry AND package visibility before writing code | CI debugging checklist |

## Decisions Made

- **`rabbitmq-client-java` made public** — utility library, no secrets; public visibility means any workflow's `GITHUB_TOKEN` can resolve it without PAT management
- **`enforce_admins` enabled on `rabbitmq-client-java`** — matches standard branch protection across all repos in the project
- **`PACKAGES_TOKEN` retained in `publish` job** — the reusable `build-push-deploy.yml` workflow requires it for Docker push; only the `lint` and `build` jobs switched to `GITHUB_TOKEN`

## Theme

A straightforward CI auth fix that took one unnecessary detour. The 401 error pointed at an expired PAT; the real issue was a private GitHub Package that `GITHUB_TOKEN` can't read cross-repo. The permanent fix — making the library repo public — eliminated an entire category of recurring work (PAT rotation across all consuming repos). Two Copilot findings caught minor doc inaccuracies in the CHANGELOG path and a premature strike-through in the README issue log.
