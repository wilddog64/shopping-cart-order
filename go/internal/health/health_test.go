package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestHealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(stubPinger{}, "0.1.0")

	t.Run("health", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
		handler.Health(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("readiness down", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/actuator/health/readiness", nil)
		handler = NewHandler(stubPinger{err: errors.New("boom")}, "0.1.0")
		handler.Readiness(ctx)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("info", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/actuator/info", nil)
		handler.Info(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}
