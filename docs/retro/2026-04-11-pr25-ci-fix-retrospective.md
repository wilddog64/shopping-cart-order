# Retrospective — PR #25: CI Fix + docs/next-improvements Catch-Up

**Date:** 2026-04-11
**Milestone:** CI unblock + docs catch-up
**PR:** #25 — merged to main (`49ff6b87`)
**Participants:** Claude, Copilot

## What Went Well

- Trivy action version fix was a clean 1-line diff once the root cause was identified
- Merge conflict resolution (CHANGELOG, README, memory-bank/activeContext.md) was handled correctly — all three sides merged without data loss
- `mergeable_state: "dirty"` was caught before waiting for CI, preventing the silent "no CI run" trap
- Copilot caught 2 real issues (stale date, inaccurate CHANGELOG entry) — both were genuine bugs in the PR content
- Thread resolution via GraphQL worked smoothly on first attempt

## What Went Wrong

- **trivy-action@0.30.0 was silently broken for weeks** — `Build, Scan & Push` failed on every main push but there was no alerting mechanism; discovered only when tracing the order-service CrashLoopBackOff root cause chain
- **Branch protection had a stale `"CI"` status check context** — this caused "CI Expected — Waiting for status to be reported" after merge even though all Actions checks passed; required a branch protection edit to replace with actual check run names (`Build & Test`, `Checkstyle`)
- **docs/next-improvements diverged 6 commits both ways** — should have been rebased onto main sooner; left it long enough that it needed 3-way conflict resolution

## Process Rules Added

| Rule | Source |
|------|--------|
| After resolving a merge conflict that changes a CHANGELOG entry's scope, verify the entry lists all items in the final merged file | Copilot finding #2 |
| Update the `## Current Status` date in memory-bank/activeContext.md every time a PR entry is recorded | Copilot finding #1 |

## Decisions Made

- **Branch protection required status checks updated**: replaced legacy `"CI"` commit status context with `Build & Test` + `Checkstyle` check run names — these now accurately reflect what the Java CI workflow actually reports
- **Next branch named `docs/next-improvements-2`** since `docs/next-improvements` was the merged branch

## Theme

This PR unblocked the entire `order-service` deployment chain. The root cause of the CrashLoopBackOff was three layers deep: expired GHCR pull secret → ReplicaSet missing imagePullSecrets → `rabbitmq-client 1.0.0-SNAPSHOT` NPE. The 1.0.1 fix was in main (PR #24) but the Docker image was never built because `Build, Scan & Push` had been silently failing since the trivy-action version `0.30.0` was pinned — a version that doesn't exist. One stale SHA in a reusable workflow pin was the single point of failure for the entire CI publish chain. The fix was a one-line SHA bump, the diagnosis took longer than the fix.
