package httpx

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	req.TLS = &tls.ConnectionState{}
	ctx.Request = req

	SecurityHeaders()(ctx)

	if got := rec.Header().Get("X-XSS-Protection"); got != "1; mode=block" {
		t.Fatalf("xss header = %q", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatalf("expected HSTS header")
	}
}

func TestCorrelationIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/orders", nil)

	CorrelationID()(ctx)

	if got := rec.Header().Get("X-Correlation-ID"); got == "" {
		t.Fatalf("expected correlation id header")
	}
}

func TestRateLimiterExemptsActuator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(1, 1)
	defer close(limiter.stopCh)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/actuator/health", nil)

	limiter.Middleware()(ctx)

	if rec.Code == http.StatusTooManyRequests {
		t.Fatalf("actuator request should not be rate limited, got status %d", rec.Code)
	}
}

func TestRateLimiterRejectsExcessRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(1, 1)
	defer close(limiter.stopCh)

	first := httptest.NewRecorder()
	firstCtx, _ := gin.CreateTestContext(first)
	firstCtx.Request = httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	firstCtx.Request.RemoteAddr = "127.0.0.1:1234"
	limiter.Middleware()(firstCtx)

	second := httptest.NewRecorder()
	secondCtx, _ := gin.CreateTestContext(second)
	secondCtx.Request = httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	secondCtx.Request.RemoteAddr = "127.0.0.1:1234"
	limiter.Middleware()(secondCtx)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
}

func TestRequestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zap.NewNop()
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/orders?customerId=cust", nil)
	ctx.Request.RemoteAddr = "127.0.0.1:1234"
	ctx.Set(string(correlationIDKey), "corr-1")

	RequestLogger(logger)(ctx)
}
