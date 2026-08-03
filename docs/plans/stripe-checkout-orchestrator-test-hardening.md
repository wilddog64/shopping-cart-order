# Test hardening — checkout orchestrator: prove server-side amount + gateway

**Branch:** `feat/stripe-checkout-orchestrator` (existing — stacked on Phase A; do NOT branch from main)
**File:** `go/internal/checkout/handler_test.go` (only)

---

## Problem

The checkout orchestrator (`3710b92`) is correct — the handler recomputes the order total
from the basket cart and never reads a client-supplied amount. But the unit tests do **not
prove** it. `fakePayment` ignores the `PaymentRequest` it receives, and `TestCheckoutHappyPath`
asserts only `"paymentStatus":"PAID"`. So the three assertions that actually guard the
money path are absent:

1. the response `amount` is the server-computed `"21.00"` (2 × 10.50),
2. the amount that reached the payment service is `"21.00"`,
3. the `gateway` that reached the payment service is `"stripe"`.

A regression that trusted a client amount or dropped the gateway would still pass all tests.
This change restores those assertions. **Test-only — no handler/production code changes.**

---

## Fix

### Change 1 — capture the request in `fakePayment`

**Exact old block:**

```go
type fakePayment struct{ paid bool }

func (f *fakePayment) ProcessPayment(context.Context, string, PaymentRequest) (PaymentOutcome, error) {
	if f.paid {
		return PaymentOutcome{Paid: true, Status: "COMPLETED"}, nil
	}
	return PaymentOutcome{Status: "FAILED", FailureReason: "card_declined"}, nil
}
```

**Exact new block:**

```go
type fakePayment struct {
	paid bool
	got  PaymentRequest
}

func (f *fakePayment) ProcessPayment(_ context.Context, _ string, req PaymentRequest) (PaymentOutcome, error) {
	f.got = req
	if f.paid {
		return PaymentOutcome{Paid: true, Status: "COMPLETED"}, nil
	}
	return PaymentOutcome{Status: "FAILED", FailureReason: "card_declined"}, nil
}
```

### Change 2 — assert the money path in `TestCheckoutHappyPath`

**Exact old block:**

```go
	if !strings.Contains(w.Body.String(), `"paymentStatus":"PAID"`) {
		t.Fatal(w.Body.String())
	}
}
```

**Exact new block:**

```go
	if !strings.Contains(w.Body.String(), `"paymentStatus":"PAID"`) {
		t.Fatal(w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"amount":"21.00"`) {
		t.Fatalf("response amount not server-computed 21.00: %s", w.Body.String())
	}
	if fp.got.Amount.StringFixed(2) != "21.00" {
		t.Fatalf("payment amount not 21.00, got %q", fp.got.Amount.StringFixed(2))
	}
	if fp.got.Gateway != "stripe" {
		t.Fatalf("payment gateway not stripe, got %q", fp.got.Gateway)
	}
	if fp.got.PaymentMethodID != "pm_card_visa" {
		t.Fatalf("payment method not pm_card_visa, got %q", fp.got.PaymentMethodID)
	}
}
```

> `fp` is already the `*fakePayment` constructed in `TestCheckoutHappyPath`
> (`fp := &fakePayment{paid: true}`). No new import — `.StringFixed(2)` is a method on the
> `decimal.Decimal` value already carried by `PaymentRequest`.

---

## Files Changed

| File | Change |
|------|--------|
| `go/internal/checkout/handler_test.go` | Capture `PaymentRequest` in `fakePayment`; assert response amount, payment amount, gateway, payment method in the happy path |

---

## Rules

- `gofmt -l go/internal/checkout/handler_test.go` — no output
- `go vet ./...` — clean
- `go build ./...` — clean
- `go test -count=1 ./internal/checkout/...` — all pass (the three checkout tests)
- No other file touched — **no handler.go / client.go / config.go / main.go changes**
- No `go.mod` / `go.sum` changes

---

## Definition of Done

- [ ] `handler_test.go` updated exactly as above — no production code touched
- [ ] `gofmt`/`go vet`/`go build`/`go test -count=1 ./internal/checkout/...` all green
- [ ] Committed and pushed to `feat/stripe-checkout-orchestrator`
- [ ] memory-bank updated with commit SHA and task status

**Commit message (exact):**
```
test(checkout): assert server-side amount and gateway reach payment request
```

---

## What NOT to Do

- Do NOT create a PR
- Do NOT skip pre-commit hooks (`--no-verify`)
- Do NOT modify any file other than `go/internal/checkout/handler_test.go`
- Do NOT touch the handler, clients, config, or route — this is a test-only change
- Do NOT commit to `main` — work on `feat/stripe-checkout-orchestrator`
- Do NOT branch from main — check out the existing branch (it is stacked on Phase A)
