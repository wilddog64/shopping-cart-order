# Copilot PR #25 Review Findings

**PR:** #25 — fix(ci): fix trivy-action version + docs/next-improvements catch-up
**Date:** 2026-04-11
**Fix commit:** `4a13dba`

---

## Finding 1 — Stale status date in `memory-bank/activeContext.md`

**File:** `memory-bank/activeContext.md:3`
**Flagged:** Header read `## Current Status (2026-03-25)` but the most recent entry (PR #24) was dated 2026-04-11.

**Before:**
```markdown
## Current Status (2026-03-25)
```

**After:**
```markdown
## Current Status (2026-04-11)
```

**Root cause:** Memory-bank header was written when PR #19 merged (2026-03-25) and not updated when PR #24 merged six weeks later.

**Process note:** Update the date in the `## Current Status` header every time a PR is recorded in `activeContext.md`.

---

## Finding 2 — Inaccurate CHANGELOG entry for README Issue Logs

**File:** `CHANGELOG.md:8`
**Flagged:** Entry listed only two of the five Issue Logs entries added to the README: "adds Copilot PR #24 findings, RabbitMQ connection refused" — omitting Rate limiting distributed state, Multi-arch workflow pin, and CI GitHub Packages auth.

**Before:**
```markdown
- README: expand Issue Logs to 5 most recent entries (adds Copilot PR #24 findings, RabbitMQ connection refused)
```

**After:**
```markdown
- README: expand Issue Logs to 5 most recent entries — Copilot PR #24 findings, RabbitMQ connection refused, Rate limiting distributed state, Multi-arch workflow pin, CI GitHub Packages auth
```

**Root cause:** CHANGELOG was written incrementally; the earlier draft only listed the two newest entries and was not updated when the merge brought in three additional entries from main.

**Process note:** After resolving a merge conflict that changes a CHANGELOG entry's scope, verify the entry lists all items that will be in the final merged file.
