package checkout

import (
	"context"
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
	created *order.Order
	updated int
}

func (f *fakeOrders) CreateOrder(_ context.Context, req order.CreateOrderRequest, _ string) (*order.Order, error) {
	o := order.NewOrder(req.CustomerID, req.Currency)
	for _, i := range req.Items {
		o.AddItem(order.OrderItem{ProductID: i.ProductID, ProductName: i.ProductName, Quantity: i.Quantity, UnitPrice: i.UnitPrice})
	}
	f.created = o
	return o, nil
}
func (f *fakeOrders) UpdateOrderStatus(_ context.Context, id uuid.UUID, req order.UpdateOrderStatusRequest, _ string) (*order.Order, error) {
	f.updated++
	f.created.Status = req.Status
	return f.created, nil
}

type fakeBasket struct {
	cart    *Cart
	cleared bool
}

func (f *fakeBasket) GetCart(context.Context, string) (*Cart, error) { return f.cart, nil }
func (f *fakeBasket) ClearCart(context.Context, string) error        { f.cleared = true; return nil }

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
func testRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/orders/checkout", func(c *gin.Context) { c.Set("customerID", "cust-1"); h.Checkout(c) })
	return r
}
func TestCheckoutHappyPath(t *testing.T) {
	fo := &fakeOrders{}
	fb := &fakeBasket{cart: &Cart{Items: []CartItem{{ProductID: "p1", Name: "Widget", Quantity: 2, UnitPrice: 10.5}}, Currency: "USD"}}
	fp := &fakePayment{paid: true}
	w := httptest.NewRecorder()
	r := testRouter(NewHandler(fo, fb, fp, "stripe", zap.NewNop()))
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/orders/checkout", strings.NewReader(`{"paymentMethodId":"pm_card_visa"}`)))
	if w.Code != 200 || !fb.cleared || fo.updated != 1 {
		t.Fatalf("unexpected happy path: %d cleared=%v updates=%d", w.Code, fb.cleared, fo.updated)
	}
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
func TestCheckoutPaymentDeclined(t *testing.T) {
	fo := &fakeOrders{}
	fb := &fakeBasket{cart: &Cart{Items: []CartItem{{ProductID: "p1", Name: "Widget", Quantity: 1, UnitPrice: 5}}, Currency: "USD"}}
	r := testRouter(NewHandler(fo, fb, &fakePayment{}, "stripe", zap.NewNop()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/orders/checkout", strings.NewReader(`{"paymentMethodId":"pm_declined"}`)))
	if w.Code != http.StatusPaymentRequired || fb.cleared || fo.updated != 0 {
		t.Fatalf("unexpected declined path: %d cleared=%v updates=%d", w.Code, fb.cleared, fo.updated)
	}
}
func TestCheckoutEmptyCart(t *testing.T) {
	fo := &fakeOrders{}
	fb := &fakeBasket{cart: &Cart{Items: []CartItem{}, Currency: "USD"}}
	r := testRouter(NewHandler(fo, fb, &fakePayment{paid: true}, "stripe", zap.NewNop()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/orders/checkout", strings.NewReader(`{"paymentMethodId":"pm_card_visa"}`)))
	if w.Code != http.StatusBadRequest || fo.created != nil {
		t.Fatalf("unexpected empty path: %d", w.Code)
	}
}
