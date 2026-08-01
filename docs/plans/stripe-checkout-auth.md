# Phase A — Order service Keycloak JWT auth

**Repo:** `shopping-cart-order`  **Branch:** `feat/stripe-checkout-auth`
**Module:** `github.com/wilddog64/shopping-cart-order` (code under `go/`)
**Design:** `shopping-cart-frontend/docs/plans/stripe-checkout-orchestration-design.md`

---

## Objective

Add Keycloak JWKS JWT validation to the order service and protect `/api/orders/**`. This is the "PR2 will add JWT middleware" note in `go/cmd/server/main.go:57`. Mirror the working validator from the basket service — do not invent a new one. When `OAUTH2_ENABLED=false` (default, local/CI), fall back to a mock middleware so existing tests/e2e keep working.

**Behavior after this phase:**
- `OAUTH2_ENABLED=true` + no/invalid `Authorization: Bearer` → **401**.
- `OAUTH2_ENABLED=true` + valid Keycloak token → request proceeds, `customerID = token sub` in gin context.
- `OAUTH2_ENABLED=false` → mock middleware sets `customerID` from `X-User-ID` (default `dev-user`); no token required.

---

## Before You Start

- `git checkout -b feat/stripe-checkout-auth origin/main`
- Read `go/cmd/server/main.go`, `go/internal/config/config.go`, `go/internal/httpx/middleware.go`.
- Reference (read-only, different repo/module): `shopping-cart-basket/internal/auth/jwt.go` and `shopping-cart-basket/internal/handler/middleware.go` — the patterns you are mirroring.

---

## Change 1 — new file `go/internal/auth/jwt.go` (verbatim copy)

`shopping-cart-basket/internal/auth/jwt.go` has **no basket-internal imports** (only `golang-jwt/jwt/v5` and `zap`), so it copies verbatim — package name stays `auth`, change nothing:

```bash
cp ../shopping-cart-basket/internal/auth/jwt.go go/internal/auth/jwt.go
```

Confirm the copied file begins with `package auth` and imports only stdlib + `github.com/golang-jwt/jwt/v5` + `go.uber.org/zap`. Do not edit it.

---

## Change 2 — new file `go/internal/httpx/auth.go`

`contextKey` is already declared in `middleware.go` (same package) — reuse it.

```go
package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wilddog64/shopping-cart-order/internal/auth"
	"go.uber.org/zap"
)

const customerIDKey contextKey = "customerID"

// SetCustomerID stores the authenticated customer id on the gin context.
func SetCustomerID(c *gin.Context, id string) { c.Set(string(customerIDKey), id) }

// GetCustomerID returns the authenticated customer id, or "" if unset.
func GetCustomerID(c *gin.Context) string { return c.GetString(string(customerIDKey)) }

// AuthMiddleware validates a Keycloak JWT and sets the customer id (token sub).
func AuthMiddleware(validator *auth.JWTValidator, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "Authorization header required",
			})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "Invalid authorization header format",
			})
			return
		}
		claims, err := validator.ValidateToken(c.Request.Context(), parts[1])
		if err != nil {
			logger.Warn("token validation failed", zap.Error(err), zap.String("ip", clientIP(c.Request)))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "Invalid or expired token",
			})
			return
		}
		SetCustomerID(c, claims.Subject)
		c.Set("claims", claims)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}

// MockAuthMiddleware is used when OAUTH2_ENABLED=false (local/CI). It trusts the
// X-User-ID header (default "dev-user") and performs no cryptographic checks.
func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.GetHeader("X-User-ID")
		if customerID == "" {
			customerID = "dev-user"
		}
		SetCustomerID(c, customerID)
		c.Set("roles", []string{"order-user"})
		c.Next()
	}
}
```

---

## Change 3 — `go/internal/config/config.go`: add `OAuth2ClientID`

**Old (struct fields):**
```go
	OAuth2Enabled   bool
	OAuth2IssuerURI string
	OAuth2JWKSetURI string
```
**New:**
```go
	OAuth2Enabled   bool
	OAuth2IssuerURI string
	OAuth2JWKSetURI string
	OAuth2ClientID  string
```

**Old (Load()):**
```go
		OAuth2Enabled:   getEnvAsBool("OAUTH2_ENABLED", false),
		OAuth2IssuerURI: getEnv("OAUTH2_ISSUER_URI", ""),
		OAuth2JWKSetURI: getEnv("OAUTH2_JWK_SET_URI", ""),
```
**New:**
```go
		OAuth2Enabled:   getEnvAsBool("OAUTH2_ENABLED", false),
		OAuth2IssuerURI: getEnv("OAUTH2_ISSUER_URI", ""),
		OAuth2JWKSetURI: getEnv("OAUTH2_JWK_SET_URI", ""),
		OAuth2ClientID:  getEnv("OAUTH2_CLIENT_ID", "order-service"),
```

---

## Change 4 — `go/cmd/server/main.go`: wire the middleware

**Old (import block — add the auth import):**
```go
	"github.com/wilddog64/shopping-cart-order/internal/config"
	"github.com/wilddog64/shopping-cart-order/internal/events"
```
**New:**
```go
	"github.com/wilddog64/shopping-cart-order/internal/auth"
	"github.com/wilddog64/shopping-cart-order/internal/config"
	"github.com/wilddog64/shopping-cart-order/internal/events"
```

**Old (the api group):**
```go
	// PR1 intentionally leaves /api/** open. PR2 will add JWT middleware here.
	api := router.Group("/api/orders")
	api.Use(rateLimiter.Middleware())
	{
```
**New:**
```go
	var authMiddleware gin.HandlerFunc
	if cfg.OAuth2Enabled {
		validator := auth.NewJWTValidator(cfg.OAuth2IssuerURI, cfg.OAuth2ClientID, logger)
		authMiddleware = httpx.AuthMiddleware(validator, logger)
	} else {
		authMiddleware = httpx.MockAuthMiddleware()
	}

	api := router.Group("/api/orders")
	api.Use(rateLimiter.Middleware())
	api.Use(authMiddleware)
	{
```

---

## Change 5 — dependency

```bash
cd go && go get github.com/golang-jwt/jwt/v5@v5.2.1 && go mod tidy
```

---

## Change 6 — new file `go/internal/httpx/auth_test.go`

```go
package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware(nil, zapNop()))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", w.Code)
	}
}

func TestMockAuthMiddlewareSetsCustomerID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MockAuthMiddleware())
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, GetCustomerID(c)) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-User-ID", "cust-123")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "cust-123" {
		t.Fatalf("expected 200/cust-123, got %d/%q", w.Code, w.Body.String())
	}
}
```

Add a tiny nop-logger helper at the bottom of the same file (the missing-token path never dereferences the validator, so `nil` is safe, but `AuthMiddleware` needs a non-nil `*zap.Logger`):

```go
func zapNop() *zap.Logger { return zap.NewNop() }
```

Add `"go.uber.org/zap"` to this test file's imports.

---

## Files Changed

| File | Change |
|------|--------|
| `go/internal/auth/jwt.go` | NEW — verbatim copy of basket validator |
| `go/internal/httpx/auth.go` | NEW — Auth + Mock middleware, customerID helpers |
| `go/internal/httpx/auth_test.go` | NEW — 401 + mock unit tests |
| `go/internal/config/config.go` | add `OAuth2ClientID` (`OAUTH2_CLIENT_ID`, default `order-service`) |
| `go/cmd/server/main.go` | build validator/mock, apply auth middleware to `/api/orders` |
| `go/go.mod`, `go/go.sum` | add `github.com/golang-jwt/jwt/v5` |

---

## Rules

- `cd go && gofmt -l .` → no output (all files formatted)
- `cd go && go vet ./...` → clean
- `cd go && go build ./...` → compiles
- `cd go && go test ./...` → all pass (including the two new tests)
- No files touched outside the table above.

---

## Definition of Done

- [ ] `/api/orders` returns 401 with `OAUTH2_ENABLED=true` and no token (covered by unit test)
- [ ] Mock path sets customerID with `OAUTH2_ENABLED=false`
- [ ] `go build ./...` and `go test ./...` pass under `go/`
- [ ] Committed and pushed to `feat/stripe-checkout-auth`
- [ ] memory-bank updated with commit SHA and task status

**Commit message (exact):**
```
feat(auth): add Keycloak JWT validation and middleware to order service
```

---

## What NOT to Do

- Do NOT create a PR.
- Do NOT skip pre-commit hooks (`--no-verify`).
- Do NOT modify order-creation logic or handlers — that is Phase C.
- Do NOT edit files outside the listed targets.
- Do NOT commit to `main` — work on `feat/stripe-checkout-auth`.
