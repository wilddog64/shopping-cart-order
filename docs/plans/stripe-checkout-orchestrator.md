# Phase C — Order service checkout orchestrator

**Repo:** `shopping-cart-order`  **Branch:** `feat/stripe-checkout-orchestrator`
**Module:** `github.com/wilddog64/shopping-cart-order` (code under `go/`)
**Design:** `shopping-cart-frontend/docs/plans/stripe-checkout-orchestration-design.md`
**Depends on:** Phase A (`feat/stripe-checkout-auth`) — this branch is **based on** it and needs
`httpx.GetCustomerID` and the auth middleware. Do NOT branch off `main`.

---

## Objective

Add `POST /api/orders/checkout` — the synchronous, payment-aware checkout the design calls for.
It reads the authoritative cart from the basket, creates a `PENDING` order with a **server-side**
total, charges the payment service (`gateway:"stripe"`, idempotency key = order id), marks the
order `PAID`, and clears the cart. Every code path here has been compiled and unit-tested against
the real order/basket/payment contracts.

**Invariant (enforce exactly):** the cart is cleared **only** after the order reaches `PAID`.
On any payment failure the order stays `PENDING` and the cart is left intact so the user can retry.
Amount and currency are recomputed server-side from the basket cart — never trusted from the client.

**Behavior:**
- No authenticated customer (`GetCustomerID` empty) → **401**.
- Missing `paymentMethodId` → **400**.
- Basket unreachable → **502**.
- Empty cart → **400**.
- Payment succeeds (`COMPLETED`) → order `PAID`, cart cleared → **200** `{orderId, amount, currency, paymentStatus:"PAID"}`.
- Payment declined / not completed → order stays `PENDING`, cart intact → **402** `{... paymentStatus:"FAILED", retryable:true, failureReason}`.

---

## Before You Start

- `git checkout feat/stripe-checkout-orchestrator && git pull origin feat/stripe-checkout-orchestrator`
  (it is stacked on the Phase A branch — the pull brings Phase A's auth code with it).
- Read `go/cmd/server/main.go`, `go/internal/config/config.go`, `go/internal/httpx/middleware.go`,
  `go/internal/httpx/auth.go` (Phase A), `go/internal/order/handler.go` (`CreateOrderRequest`,
  `CreateOrderItemRequest`, `CreateOrderAddressRequest`), `go/internal/order/service.go`
  (`CreateOrder`, `UpdateOrderStatus` signatures), `go/internal/order/model.go` (`OrderStatusPaid`).
- **No dependency changes.** Everything used (`decimal`, `uuid`, `gin`, `zap`) is already in `go.mod`.
- **Verified fact:** gin v1.10.0 allows `POST /api/orders/checkout` (static) to coexist with the
  existing `:orderId` param routes — no route-tree panic. Register it on the same `/api/orders` group.

---

## Contracts this code was written against (do not re-derive)

- Basket `GET /api/v1/cart` returns `{"success":true,"data":{"items":[{"productId","name","quantity","unitPrice"(float64)}],"totalAmount"(float64),"currency"}}`. Basket `DELETE /api/v1/cart` clears it. Propagate the caller's `Authorization` header on both.
- Payment `POST /api/v1/payments` accepts `{orderId,customerId,amount(decimal),currency,gateway,paymentMethodId}` + `X-Idempotency-Key` header; returns `PaymentResponse` with `status` (`COMPLETED`/`FAILED`/…) and `failureReason`. HTTP 201 when `COMPLETED`, 202 otherwise. **Treat "paid" as: HTTP 2xx AND `status=="COMPLETED"`** — robust regardless of the exact status code.
- Order `CreateOrder(ctx, order.CreateOrderRequest, correlationID)` builds a `PENDING` order and recomputes totals server-side. `UpdateOrderStatus(ctx, id, order.UpdateOrderStatusRequest{Status: order.OrderStatusPaid}, correlationID)` advances `PENDING → PAID` (a valid transition).

---

## Change 1 — `go/internal/config/config.go`: service URLs + gateway

**Old (struct — last two lines shown for anchor):**
```go
	VaultEnabled bool
}
```
**New:**
```go
	VaultEnabled bool

	BasketServiceURL  string
	PaymentServiceURL string
	PaymentGateway    string
}
```

**Old (end of `Load()`):**
```go
		VaultEnabled: getEnvAsBool("VAULT_ENABLED", false),
	}
}
```
**New:**
```go
		VaultEnabled: getEnvAsBool("VAULT_ENABLED", false),

		BasketServiceURL:  getEnv("BASKET_URL", "http://localhost:8083"),
		PaymentServiceURL: getEnv("PAYMENT_URL", "http://localhost:8084"),
		PaymentGateway:    getEnv("PAYMENT_GATEWAY", "stripe"),
	}
}
```
> `BASKET_URL`/`PAYMENT_URL` match the e2e-tests convention (basket :8083, payment :8084).

---

## Change 2 — `go/internal/httpx/middleware.go`: export correlation-id getter

Add immediately after the `CorrelationID()` function (which ends with `}` after `c.Next()` / `}`):

**Old:**
```go
		c.Set(string(correlationIDKey), correlationID)
		c.Header("X-Correlation-ID", correlationID)
		c.Next()
	}
}
```
**New:**
```go
		c.Set(string(correlationIDKey), correlationID)
		c.Header("X-Correlation-ID", correlationID)
		c.Next()
	}
}

// GetCorrelationID returns the correlation id set by the CorrelationID middleware.
func GetCorrelationID(c *gin.Context) string { return c.GetString(string(correlationIDKey)) }
```

---

## Change 3 — new file `go/internal/checkout/client.go`

Thin HTTP clients for basket + payment, propagating the caller's JWT. Verbatim:

```go
package checkout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// CartItem is the subset of the basket cart item the orchestrator needs.
type CartItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

// Cart is the subset of the basket cart response the orchestrator needs.
type Cart struct {
	Items       []CartItem `json:"items"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
}

type basketEnvelope struct {
	Success bool `json:"success"`
	Data    Cart `json:"data"`
}

// BasketClient calls the basket service, propagating the caller's JWT.
type BasketClient struct {
	baseURL string
	http    *http.Client
}

func NewBasketClient(baseURL string) *BasketClient {
	return &BasketClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

func (bc *BasketClient) GetCart(ctx context.Context, authHeader string) (*Cart, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bc.baseURL+"/api/v1/cart", nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := bc.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("basket GET /api/v1/cart: unexpected status %d", resp.StatusCode)
	}
	var env basketEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode cart: %w", err)
	}
	return &env.Data, nil
}

func (bc *BasketClient) ClearCart(ctx context.Context, authHeader string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, bc.baseURL+"/api/v1/cart", nil)
	if err != nil {
		return err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := bc.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("basket DELETE /api/v1/cart: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// PaymentRequest is the order->payment charge request. Raw card fields are
// intentionally absent — only the PaymentMethod token is sent.
type PaymentRequest struct {
	OrderID         string          `json:"orderId"`
	CustomerID      string          `json:"customerId"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	Gateway         string          `json:"gateway"`
	PaymentMethodID string          `json:"paymentMethodId"`
}

type paymentResponse struct {
	Status        string  `json:"status"`
	FailureReason *string `json:"failureReason"`
}

// PaymentOutcome is the normalized result the orchestrator acts on.
type PaymentOutcome struct {
	Paid          bool
	Status        string
	FailureReason string
}

// PaymentClient calls the payment service, propagating the caller's JWT.
type PaymentClient struct {
	baseURL string
	http    *http.Client
}

func NewPaymentClient(baseURL string) *PaymentClient {
	return &PaymentClient{baseURL: baseURL, http: &http.Client{Timeout: 30 * time.Second}}
}

func (pc *PaymentClient) ProcessPayment(ctx context.Context, authHeader string, pr PaymentRequest) (PaymentOutcome, error) {
	body, err := json.Marshal(pr)
	if err != nil {
		return PaymentOutcome{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pc.baseURL+"/api/v1/payments", bytes.NewReader(body))
	if err != nil {
		return PaymentOutcome{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", pr.OrderID)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := pc.http.Do(req)
	if err != nil {
		return PaymentOutcome{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var pres paymentResponse
	_ = json.Unmarshal(raw, &pres)
	outcome := PaymentOutcome{Status: pres.Status}
	if pres.FailureReason != nil {
		outcome.FailureReason = *pres.FailureReason
	}
	outcome.Paid = resp.StatusCode >= 200 && resp.StatusCode < 300 && pres.Status == "COMPLETED"
	if !outcome.Paid && outcome.FailureReason == "" {
		outcome.FailureReason = fmt.Sprintf("payment not completed (http %d, status %q)", resp.StatusCode, pres.Status)
	}
	return outcome, nil
}
```

---

## Change 4 — new file `go/internal/checkout/handler.go`

The orchestrator. `OrderService`/`Basket`/`Payment` are consumer-defined interfaces so the handler
is testable without a database. Verbatim:

```go
package checkout

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wilddog64/shopping-cart-order/internal/httpx"
	"github.com/wilddog64/shopping-cart-order/internal/order"
	"go.uber.org/zap"
)

// OrderService is the subset of *order.Service the orchestrator depends on
// (consumer-defined so the handler is testable without a database).
type OrderService interface {
	CreateOrder(ctx context.Context, req order.CreateOrderRequest, correlationID string) (*order.Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, req order.UpdateOrderStatusRequest, correlationID string) (*order.Order, error)
}

// Basket is the subset of *BasketClient the orchestrator depends on.
type Basket interface {
	GetCart(ctx context.Context, authHeader string) (*Cart, error)
	ClearCart(ctx context.Context, authHeader string) error
}

// Payment is the subset of *PaymentClient the orchestrator depends on.
type Payment interface {
	ProcessPayment(ctx context.Context, authHeader string, pr PaymentRequest) (PaymentOutcome, error)
}

type Handler struct {
	orders  OrderService
	basket  Basket
	payment Payment
	gateway string
	logger  *zap.Logger
}

func NewHandler(orders OrderService, basket Basket, payment Payment, gateway string, logger *zap.Logger) *Handler {
	return &Handler{orders: orders, basket: basket, payment: payment, gateway: gateway, logger: logger}
}

type checkoutRequest struct {
	ShippingAddress *order.CreateOrderAddressRequest `json:"shippingAddress"`
	PaymentMethodID string                           `json:"paymentMethodId"`
}

type checkoutResponse struct {
	OrderID       string `json:"orderId"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	PaymentStatus string `json:"paymentStatus"`
	Retryable     bool   `json:"retryable,omitempty"`
	FailureReason string `json:"failureReason,omitempty"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"code": code, "message": message})
}

// Checkout orchestrates the synchronous, payment-aware checkout:
// read authoritative cart -> create PENDING order (server-side total) ->
// charge -> mark PAID and clear cart. The cart is cleared ONLY after PAID.
func (h *Handler) Checkout(c *gin.Context) {
	customerID := httpx.GetCustomerID(c)
	if customerID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated customer required")
		return
	}

	var req checkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request: "+err.Error())
		return
	}
	if req.PaymentMethodID == "" {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "paymentMethodId is required")
		return
	}

	ctx := c.Request.Context()
	authHeader := c.GetHeader("Authorization")
	correlationID := httpx.GetCorrelationID(c)

	cart, err := h.basket.GetCart(ctx, authHeader)
	if err != nil {
		h.logger.Error("checkout: failed to read cart", zap.String("customerId", customerID), zap.Error(err))
		writeError(c, http.StatusBadGateway, "BASKET_UNAVAILABLE", "could not read cart")
		return
	}
	if len(cart.Items) == 0 {
		writeError(c, http.StatusBadRequest, "EMPTY_CART", "cart is empty")
		return
	}

	createReq := order.CreateOrderRequest{
		CustomerID:      customerID,
		Currency:        cart.Currency,
		ShippingAddress: req.ShippingAddress,
		Items:           make([]order.CreateOrderItemRequest, 0, len(cart.Items)),
	}
	for _, item := range cart.Items {
		createReq.Items = append(createReq.Items, order.CreateOrderItemRequest{
			ProductID:   item.ProductID,
			ProductName: item.Name,
			Quantity:    item.Quantity,
			UnitPrice:   decimal.NewFromFloat(item.UnitPrice).Round(2),
		})
	}

	orderEntity, err := h.orders.CreateOrder(ctx, createReq, correlationID)
	if err != nil {
		h.logger.Error("checkout: failed to create order", zap.String("customerId", customerID), zap.Error(err))
		writeError(c, http.StatusBadRequest, "ORDER_CREATE_FAILED", err.Error())
		return
	}

	outcome, err := h.payment.ProcessPayment(ctx, authHeader, PaymentRequest{
		OrderID:         orderEntity.ID.String(),
		CustomerID:      customerID,
		Amount:          orderEntity.TotalAmount,
		Currency:        orderEntity.Currency,
		Gateway:         h.gateway,
		PaymentMethodID: req.PaymentMethodID,
	})
	if err != nil || !outcome.Paid {
		reason := "payment failed"
		if err != nil {
			h.logger.Error("checkout: payment call failed", zap.String("orderId", orderEntity.ID.String()), zap.Error(err))
			reason = "payment service unavailable"
		} else if outcome.FailureReason != "" {
			reason = outcome.FailureReason
		}
		// Order stays PENDING; cart is NOT cleared. Client may retry the same
		// order id (payment de-dupes on the idempotency key) or cancel.
		c.JSON(http.StatusPaymentRequired, checkoutResponse{
			OrderID:       orderEntity.ID.String(),
			Amount:        orderEntity.TotalAmount.StringFixed(2),
			Currency:      orderEntity.Currency,
			PaymentStatus: "FAILED",
			Retryable:     true,
			FailureReason: reason,
		})
		return
	}

	if _, err := h.orders.UpdateOrderStatus(ctx, orderEntity.ID, order.UpdateOrderStatusRequest{
		Status:        order.OrderStatusPaid,
		PaymentMethod: h.gateway,
	}, correlationID); err != nil {
		// Payment captured but we could not advance the order. Do NOT clear the cart.
		h.logger.Error("checkout: paid but failed to mark order PAID", zap.String("orderId", orderEntity.ID.String()), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "ORDER_UPDATE_FAILED", "payment captured but order update failed; contact support")
		return
	}

	if err := h.basket.ClearCart(ctx, authHeader); err != nil {
		// Non-fatal: the order is PAID. A stale cart is recoverable; do not fail the checkout.
		h.logger.Warn("checkout: order PAID but cart clear failed", zap.String("orderId", orderEntity.ID.String()), zap.Error(err))
	}

	c.JSON(http.StatusOK, checkoutResponse{
		OrderID:       orderEntity.ID.String(),
		Amount:        orderEntity.TotalAmount.StringFixed(2),
		Currency:      orderEntity.Currency,
		PaymentStatus: "PAID",
	})
}
```

---

## Change 5 — new file `go/internal/checkout/handler_test.go`

Hermetic tests: `httptest` basket + payment servers exercise the real clients; a fake `OrderService`
avoids the database. Verbatim:

```go
package checkout

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wilddog64/shopping-cart-order/internal/order"
	"go.uber.org/zap"
)

type fakeOrders struct {
	created     *order.Order
	updatedTo   order.OrderStatus
	updateCalls int
}

func (f *fakeOrders) CreateOrder(ctx context.Context, req order.CreateOrderRequest, correlationID string) (*order.Order, error) {
	o := order.NewOrder(req.CustomerID, req.Currency)
	for _, it := range req.Items {
		o.AddItem(order.OrderItem{ProductID: it.ProductID, ProductName: it.ProductName, Quantity: it.Quantity, UnitPrice: it.UnitPrice})
	}
	f.created = o
	return o, nil
}

func (f *fakeOrders) UpdateOrderStatus(ctx context.Context, id uuid.UUID, req order.UpdateOrderStatusRequest, correlationID string) (*order.Order, error) {
	f.updateCalls++
	f.updatedTo = req.Status
	if f.created != nil {
		f.created.Status = req.Status
	}
	return f.created, nil
}

func newTestRouter(h *Handler, customerID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/orders/checkout", func(c *gin.Context) {
		if customerID != "" {
			c.Set("customerID", customerID)
		}
		h.Checkout(c)
	})
	return r
}

func TestCheckoutHappyPath(t *testing.T) {
	basket := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/cart":
			_ = json.NewEncoder(w).Encode(basketEnvelope{Success: true, Data: Cart{
				Items:    []CartItem{{ProductID: "p1", Name: "Widget", Quantity: 2, UnitPrice: 10.50}},
				Currency: "USD",
			}})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/cart":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer basket.Close()

	var gotPayReq PaymentRequest
	payment := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotPayReq)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
	}))
	defer payment.Close()

	fo := &fakeOrders{}
	h := NewHandler(fo, NewBasketClient(basket.URL), NewPaymentClient(payment.URL), "stripe", zap.NewNop())
	r := newTestRouter(h, "cust-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/orders/checkout", strings.NewReader(`{"paymentMethodId":"pm_card_visa"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp checkoutResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.PaymentStatus != "PAID" {
		t.Fatalf("expected PAID, got %q", resp.PaymentStatus)
	}
	if resp.Amount != "21.00" {
		t.Fatalf("expected amount 21.00, got %q", resp.Amount)
	}
	if fo.updatedTo != order.OrderStatusPaid {
		t.Fatalf("order not marked PAID, got %q", fo.updatedTo)
	}
	if gotPayReq.Amount.StringFixed(2) != "21.00" || gotPayReq.Gateway != "stripe" || gotPayReq.PaymentMethodID != "pm_card_visa" {
		t.Fatalf("payment request wrong: %+v", gotPayReq)
	}
}

func TestCheckoutPaymentDeclined(t *testing.T) {
	basket := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			t.Errorf("cart must NOT be cleared on payment failure")
		}
		_ = json.NewEncoder(w).Encode(basketEnvelope{Success: true, Data: Cart{
			Items:    []CartItem{{ProductID: "p1", Name: "Widget", Quantity: 1, UnitPrice: 5}},
			Currency: "USD",
		}})
	}))
	defer basket.Close()

	payment := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		reason := "card_declined"
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "FAILED", "failureReason": reason})
	}))
	defer payment.Close()

	fo := &fakeOrders{}
	h := NewHandler(fo, NewBasketClient(basket.URL), NewPaymentClient(payment.URL), "stripe", zap.NewNop())
	r := newTestRouter(h, "cust-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/orders/checkout", strings.NewReader(`{"paymentMethodId":"pm_card_chargeDeclined"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d: %s", w.Code, w.Body.String())
	}
	var resp checkoutResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.PaymentStatus != "FAILED" || !resp.Retryable {
		t.Fatalf("expected FAILED+retryable, got %+v", resp)
	}
	if fo.updateCalls != 0 {
		t.Fatalf("order must stay PENDING; UpdateOrderStatus called %d times", fo.updateCalls)
	}
}

func TestCheckoutEmptyCart(t *testing.T) {
	basket := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(basketEnvelope{Success: true, Data: Cart{Items: []CartItem{}, Currency: "USD"}})
	}))
	defer basket.Close()

	payment := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("payment must not be called for an empty cart")
	}))
	defer payment.Close()

	fo := &fakeOrders{}
	h := NewHandler(fo, NewBasketClient(basket.URL), NewPaymentClient(payment.URL), "stripe", zap.NewNop())
	r := newTestRouter(h, "cust-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/orders/checkout", strings.NewReader(`{"paymentMethodId":"pm_card_visa"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty cart, got %d", w.Code)
	}
	if fo.created != nil {
		t.Fatalf("no order should be created for an empty cart")
	}
}
```

---

## Change 6 — `go/cmd/server/main.go`: construct + register

**Old (import block — add the checkout import):**
```go
	"github.com/wilddog64/shopping-cart-order/internal/auth"
	"github.com/wilddog64/shopping-cart-order/internal/config"
```
**New:**
```go
	"github.com/wilddog64/shopping-cart-order/internal/auth"
	"github.com/wilddog64/shopping-cart-order/internal/checkout"
	"github.com/wilddog64/shopping-cart-order/internal/config"
```

**Old (handler construction):**
```go
	orderService := order.NewService(store, publisher, logger)
	orderHandler := order.NewHandler(orderService, logger)
```
**New:**
```go
	orderService := order.NewService(store, publisher, logger)
	orderHandler := order.NewHandler(orderService, logger)
	checkoutHandler := checkout.NewHandler(
		orderService,
		checkout.NewBasketClient(cfg.BasketServiceURL),
		checkout.NewPaymentClient(cfg.PaymentServiceURL),
		cfg.PaymentGateway,
		logger,
	)
```

**Old (route registration — inside the `api` group block):**
```go
		api.POST("", orderHandler.CreateOrder)
		api.GET("", orderHandler.ListOrdersByCustomer)
```
**New:**
```go
		api.POST("", orderHandler.CreateOrder)
		api.POST("/checkout", checkoutHandler.Checkout)
		api.GET("", orderHandler.ListOrdersByCustomer)
```
> The route lands inside the existing group, so it inherits the rate limiter **and** the auth
> middleware — `GetCustomerID` resolves the JWT `sub` (or the `X-User-ID` mock in local/CI).

---

## Files Changed

| File | Change |
|------|--------|
| `go/internal/config/config.go` | add `BasketServiceURL`/`PaymentServiceURL`/`PaymentGateway` (`BASKET_URL`/`PAYMENT_URL`/`PAYMENT_GATEWAY`) |
| `go/internal/httpx/middleware.go` | export `GetCorrelationID` |
| `go/internal/checkout/client.go` | NEW — basket + payment HTTP clients, DTOs |
| `go/internal/checkout/handler.go` | NEW — orchestrator handler + consumer interfaces |
| `go/internal/checkout/handler_test.go` | NEW — hermetic happy/declined/empty-cart tests |
| `go/cmd/server/main.go` | construct orchestrator, register `POST /api/orders/checkout` |

No `go.mod`/`go.sum` changes.

---

## Rules

- `cd go && gofmt -l .` → no output
- `cd go && go vet ./...` → clean
- `cd go && go build ./...` → compiles
- `cd go && go test ./...` → all pass (the three new checkout tests included)
- No files touched outside the table above. Do NOT modify `internal/order` logic (that stays as-is).

---

## Definition of Done

- [ ] Happy path → order `PAID`, cart cleared, 200 `paymentStatus:"PAID"` (unit test)
- [ ] Payment declined → order stays `PENDING`, cart NOT cleared, 402 `FAILED`+`retryable` (unit test)
- [ ] Empty cart → 400, no order created (unit test)
- [ ] `go build ./...` and `go test ./...` pass under `go/`
- [ ] Committed and pushed to `feat/stripe-checkout-orchestrator`
- [ ] memory-bank updated with commit SHA and task status

**Commit message (exact):**
```
feat(checkout): add payment-aware checkout orchestrator to order service
```

---

## What NOT to Do

- Do NOT create a PR.
- Do NOT skip pre-commit hooks (`--no-verify`).
- Do NOT clear the cart before the order is `PAID`, and do NOT clear it on any failure path.
- Do NOT trust client-supplied amount/currency/customerId — recompute from the basket cart and the JWT.
- Do NOT populate the payment raw-card fields (`cardNumber`/`cardCvc`/…) — send `paymentMethodId` only.
- Do NOT modify order-creation/status logic in `internal/order`.
- Do NOT branch off `main` — this is stacked on `feat/stripe-checkout-auth` (Phase A).
- Do NOT commit to `main`.
