package order

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wilddog64/shopping-cart-order/internal/httpx"
	"go.uber.org/zap"
)

func TestGetOrderRejectsCrossCustomer(t *testing.T) {
	store := newMemoryStore()
	entity := NewOrder("owner-user", "USD")
	entity.ID = uuid.New()
	if err := store.Create(t.Context(), entity); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(store, nil, zap.NewNop()), zap.NewNop())
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/orders/"+entity.ID.String(), nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "orderId", Value: entity.ID.String()}}
	httpx.SetCustomerID(c, "other-user")

	h.GetOrder(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.String() == "" || rec.Body.String() == entity.ID.String() {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestListOrdersByCustomerIgnoresQueryParameter(t *testing.T) {
	store := newMemoryStore()
	owner := NewOrder("owner-user", "USD")
	owner.ID = uuid.New()
	other := NewOrder("someone-else", "USD")
	other.ID = uuid.New()
	if err := store.Create(t.Context(), owner); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(t.Context(), other); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(store, nil, zap.NewNop()), zap.NewNop())
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/orders?customerId=someone-else", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	httpx.SetCustomerID(c, "owner-user")

	h.ListOrdersByCustomer(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got == "" || !containsOrderID(got, owner.ID.String()) || containsOrderID(got, other.ID.String()) {
		t.Fatalf("response did not contain only authenticated customer's order: %s", got)
	}
}

func containsOrderID(body, id string) bool {
	return len(body) >= len(id) && stringContains(body, id)
}

func stringContains(body, value string) bool {
	for i := 0; i+len(value) <= len(body); i++ {
		if body[i:i+len(value)] == value {
			return true
		}
	}
	return false
}
