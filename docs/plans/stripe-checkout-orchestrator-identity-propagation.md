# Checkout orchestrator — propagate caller identity (X-User-ID) to basket + payment

**Branch:** `feat/stripe-checkout-orchestrator` (existing — completes Phase C; do NOT branch from main)
**Files:** `go/internal/checkout/client.go`, `go/internal/checkout/handler.go`, `go/internal/checkout/handler_test.go`

---

## Problem

The orchestrator forwards **only** the `Authorization` header to the basket and payment
services (`handler.go`: `authHeader := c.GetHeader("Authorization")`). But those services
resolve the caller's identity from **two** sources depending on auth mode:

- **OAuth2 mode** (`OAUTH2_ENABLED=true`): identity comes from the JWT `subject` carried in
  `Authorization: Bearer …`. Forwarding `Authorization` works — basket reads the same cart. ✅
- **Mock-auth mode** (`OAUTH2_ENABLED=false`, the default for local/dev and the e2e suite):
  identity comes from the **`X-User-ID`** header (`MockAuthMiddleware`). The orchestrator
  resolves its own `customerID` from `X-User-ID`, but then forwards an **empty** `Authorization`
  downstream and drops `X-User-ID`. Basket's `MockAuthMiddleware` then falls back to
  `"dev-user"` and reads the **wrong cart** → checkout fails with `EMPTY_CART`, or worse, could
  read/clear a different user's cart. ❌

Production runs OAuth2-only, so this is not a live incident — but it is a latent footgun
(empty `Authorization` → someone else's cart) and it blocks any mock-mode / dev exercise of the
orchestrator. Fix: forward `X-User-ID = <resolved customerID>` on every downstream call,
**alongside** `Authorization`. This is additive and safe in OAuth2 mode — basket/payment ignore
`X-User-ID` whenever a valid JWT is present.

---

## Fix

### Change 1 — `client.go`: `BasketClient.GetCart` forwards `X-User-ID`

**Exact old block:**

```go
func (bc *BasketClient) GetCart(ctx context.Context, authHeader string) (*Cart, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bc.baseURL+"/api/v1/cart", nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
```

**Exact new block:**

```go
func (bc *BasketClient) GetCart(ctx context.Context, authHeader, customerID string) (*Cart, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bc.baseURL+"/api/v1/cart", nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if customerID != "" {
		req.Header.Set("X-User-ID", customerID)
	}
```

### Change 2 — `client.go`: `BasketClient.ClearCart` forwards `X-User-ID`

**Exact old block:**

```go
func (bc *BasketClient) ClearCart(ctx context.Context, authHeader string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, bc.baseURL+"/api/v1/cart", nil)
	if err != nil {
		return err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
```

**Exact new block:**

```go
func (bc *BasketClient) ClearCart(ctx context.Context, authHeader, customerID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, bc.baseURL+"/api/v1/cart", nil)
	if err != nil {
		return err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if customerID != "" {
		req.Header.Set("X-User-ID", customerID)
	}
```

### Change 3 — `client.go`: `PaymentClient.ProcessPayment` forwards `X-User-ID` from `pr.CustomerID`

**Exact old block:**

```go
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", pr.OrderID)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
```

**Exact new block:**

```go
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", pr.OrderID)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if pr.CustomerID != "" {
		req.Header.Set("X-User-ID", pr.CustomerID)
	}
```

### Change 4 — `handler.go`: update the `Basket` interface signatures

**Exact old block:**

```go
type Basket interface {
	GetCart(context.Context, string) (*Cart, error)
	ClearCart(context.Context, string) error
}
```

**Exact new block:**

```go
type Basket interface {
	GetCart(context.Context, string, string) (*Cart, error)
	ClearCart(context.Context, string, string) error
}
```

### Change 5 — `handler.go`: pass `customerID` at the `GetCart` call site

`customerID` is already in scope (resolved via `httpx.GetCustomerID(c)` earlier in the handler).

**Exact old block:**

```go
	cart, err := h.basket.GetCart(ctx, authHeader)
```

**Exact new block:**

```go
	cart, err := h.basket.GetCart(ctx, authHeader, customerID)
```

### Change 6 — `handler.go`: pass `customerID` at the `ClearCart` call site

**Exact old block:**

```go
	if err := h.basket.ClearCart(ctx, authHeader); err != nil {
```

**Exact new block:**

```go
	if err := h.basket.ClearCart(ctx, authHeader, customerID); err != nil {
```

### Change 7 — `handler_test.go`: update `fakeBasket` to match the new interface

**Exact old block:**

```go
func (f *fakeBasket) GetCart(context.Context, string) (*Cart, error) { return f.cart, nil }
func (f *fakeBasket) ClearCart(context.Context, string) error        { f.cleared = true; return nil }
```

**Exact new block (run `gofmt` — it normalizes the brace alignment):**

```go
func (f *fakeBasket) GetCart(context.Context, string, string) (*Cart, error) { return f.cart, nil }
func (f *fakeBasket) ClearCart(context.Context, string, string) error        { f.cleared = true; return nil }
```

> Do NOT touch `fakePayment` or any assertion in `TestCheckoutHappyPath` — the amount/gateway
> hardening (`88e5be6`) must survive this change unchanged. `Payment.ProcessPayment`'s signature
> is unchanged; only its internal header set is added in Change 3.

---

## Files Changed

| File | Change |
|------|--------|
| `go/internal/checkout/client.go` | `GetCart`/`ClearCart` gain a `customerID` param and set `X-User-ID`; `ProcessPayment` sets `X-User-ID` from `pr.CustomerID` |
| `go/internal/checkout/handler.go` | `Basket` interface signatures updated; both call sites pass `customerID` |
| `go/internal/checkout/handler_test.go` | `fakeBasket` method signatures updated to match |

---

## Rules

- `gofmt -l go/internal/checkout/` — no output
- `go vet ./...` — clean
- `go build ./...` — clean
- `go test -count=1 ./internal/checkout/...` — all pass (happy/declined/empty-cart, hardening intact)
- No other file touched — **no `main.go`, no route, no order/payment domain code**
- No `go.mod` / `go.sum` changes
- OAuth2-mode behavior must be unchanged — `X-User-ID` is only ADDED, `Authorization` is still forwarded exactly as before

---

## Definition of Done

- [ ] Changes 1–7 applied exactly — no other production code touched
- [ ] `gofmt`/`go vet`/`go build`/`go test -count=1 ./internal/checkout/...` all green
- [ ] `fakePayment` + amount/gateway assertions from `88e5be6` unchanged
- [ ] Committed and pushed to `feat/stripe-checkout-orchestrator`
- [ ] memory-bank updated with commit SHA and task status

**Commit message (exact):**
```
fix(checkout): forward X-User-ID to basket and payment for mock-auth parity
```

---

## What NOT to Do

- Do NOT create a PR
- Do NOT skip pre-commit hooks (`--no-verify`)
- Do NOT modify any file other than the three listed targets
- Do NOT change the `Payment` interface signature or any assertion in `handler_test.go`
- Do NOT commit to `main` — work on `feat/stripe-checkout-orchestrator`
- Do NOT branch from main — check out the existing branch (it is stacked on Phase A)
