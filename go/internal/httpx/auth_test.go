package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

func zapNop() *zap.Logger { return zap.NewNop() }
