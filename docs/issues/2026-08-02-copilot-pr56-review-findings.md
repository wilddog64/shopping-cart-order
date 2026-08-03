# Copilot PR #56 review findings

**Date:** 2026-08-02
**PR:** #56 — `fix(order): scope order reads to authenticated customer + enforce token audience`
**Reviewer:** copilot-pull-request-reviewer[bot] (state: COMMENTED)

PR #56 lands the order access-control hardening (Copilot findings #2 aud/azp and #4 IDOR
from the deferred-hardening register). Copilot's review of the PR itself raised three
comments.

---

## Finding 1 — `go/internal/order/handler.go:182` — trim parity (fixed)

**Flagged:** `GetOrder` compared `orderEntity.CustomerID` against an **untrimmed**
`httpx.GetCustomerID(c)`, while `ListOrdersByCustomer` trims the customer ID. A subject
with leading/trailing whitespace could produce a false `404`.

**Fix applied** (commit `09b0abf`): `GetOrder` now trims the authenticated customer ID and
returns `401 UNAUTHORIZED` on an empty subject, matching `ListOrdersByCustomer` exactly.

**Before:**
```go
if orderEntity.CustomerID != httpx.GetCustomerID(c) {
	writeError(c, http.StatusNotFound, "NOT_FOUND", "order not found")
	return
}
```

**After:**
```go
customerID := strings.TrimSpace(httpx.GetCustomerID(c))
if customerID == "" {
	writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated customer required")
	return
}
if orderEntity.CustomerID != customerID {
	writeError(c, http.StatusNotFound, "NOT_FOUND", "order not found")
	return
}
```

**Root cause:** the two handlers were written in the same commit but only one normalized
the subject. The spec's exact blocks trimmed in `ListOrdersByCustomer` (carried over from
the original query-param path) but not in the newly added `GetOrder` ownership check.

**Process note:** when a spec adds the same derived value (`httpx.GetCustomerID(c)`) in
more than one handler, normalize it identically in every site — or extract a single helper.
Future access-control specs should show the trimmed form in *every* block that consumes the
subject.

---

## Findings 2 & 3 — `memory-bank/activeContext.md:5`, `memory-bank/progress.md:3` — "COMPLETE" before CI (resolved by state)

**Flagged:** both entries marked the work `COMPLETE` and asserted broad test success while
the PR's CI test-plan boxes were still unchecked, risking the memory-bank becoming a source
of truth that contradicts CI.

**Resolution:** no wording change needed — CI is now **green** on the PR head (Go CI +
Java CI, `push` and `pull_request`), so the "tests pass" claim holds. The entries described
the local uncached gate results (which did pass); the gap was purely temporal. Threads
resolved with a note that CI is green.

**Process note:** memory-bank "COMPLETE" for a change that still has an open PR should read
"implemented + local gates green; PR CI pending" until the PR's CI is confirmed green, then
be upgraded. Minor — the claim was accurate, just early.

---

## Test plan status

- [x] CI green on PR head (`6efe8e1` → `09b0abf`): Go CI + Java CI, both `push` and `pull_request`
- [x] `gofmt -l`, `go build`, `go vet`, `go test -count=1 ./...` — green locally, uncached
- [x] Finding 1 fixed and re-verified
