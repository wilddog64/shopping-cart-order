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

type OrderService interface {
	CreateOrder(context.Context, order.CreateOrderRequest, string) (*order.Order, error)
	UpdateOrderStatus(context.Context, uuid.UUID, order.UpdateOrderStatusRequest, string) (*order.Order, error)
}
type Basket interface {
	GetCart(context.Context, string, string) (*Cart, error)
	ClearCart(context.Context, string, string) error
}
type Payment interface {
	ProcessPayment(context.Context, string, PaymentRequest) (PaymentOutcome, error)
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
	cart, err := h.basket.GetCart(ctx, authHeader, customerID)
	if err != nil {
		h.logger.Error("checkout: failed to read cart", zap.String("customerId", customerID), zap.Error(err))
		writeError(c, http.StatusBadGateway, "BASKET_UNAVAILABLE", "could not read cart")
		return
	}
	if len(cart.Items) == 0 {
		writeError(c, http.StatusBadRequest, "EMPTY_CART", "cart is empty")
		return
	}
	createReq := order.CreateOrderRequest{CustomerID: customerID, Currency: cart.Currency, ShippingAddress: req.ShippingAddress, Items: make([]order.CreateOrderItemRequest, 0, len(cart.Items))}
	for _, item := range cart.Items {
		createReq.Items = append(createReq.Items, order.CreateOrderItemRequest{ProductID: item.ProductID, ProductName: item.Name, Quantity: item.Quantity, UnitPrice: decimal.NewFromFloat(item.UnitPrice).Round(2)})
	}
	entity, err := h.orders.CreateOrder(ctx, createReq, correlationID)
	if err != nil {
		h.logger.Error("checkout: failed to create order", zap.String("customerId", customerID), zap.Error(err))
		writeError(c, http.StatusBadRequest, "ORDER_CREATE_FAILED", err.Error())
		return
	}
	outcome, err := h.payment.ProcessPayment(ctx, authHeader, PaymentRequest{OrderID: entity.ID.String(), CustomerID: customerID, Amount: entity.TotalAmount, Currency: entity.Currency, Gateway: h.gateway, PaymentMethodID: req.PaymentMethodID})
	if err != nil || !outcome.Paid {
		reason := "payment failed"
		if err != nil {
			h.logger.Error("checkout: payment call failed", zap.String("orderId", entity.ID.String()), zap.Error(err))
			reason = "payment service unavailable"
		} else if outcome.FailureReason != "" {
			reason = outcome.FailureReason
		}
		c.JSON(http.StatusPaymentRequired, checkoutResponse{OrderID: entity.ID.String(), Amount: entity.TotalAmount.StringFixed(2), Currency: entity.Currency, PaymentStatus: "FAILED", Retryable: true, FailureReason: reason})
		return
	}
	if _, err := h.orders.UpdateOrderStatus(ctx, entity.ID, order.UpdateOrderStatusRequest{Status: order.OrderStatusPaid, PaymentMethod: h.gateway, PaymentID: outcome.PaymentID}, correlationID); err != nil {
		h.logger.Error("checkout: paid but failed to mark order PAID", zap.String("orderId", entity.ID.String()), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "ORDER_UPDATE_FAILED", "payment captured but order update failed; contact support")
		return
	}
	if err := h.basket.ClearCart(ctx, authHeader, customerID); err != nil {
		h.logger.Warn("checkout: order PAID but cart clear failed", zap.String("orderId", entity.ID.String()), zap.Error(err))
	}
	c.JSON(http.StatusOK, checkoutResponse{OrderID: entity.ID.String(), Amount: entity.TotalAmount.StringFixed(2), Currency: entity.Currency, PaymentStatus: "PAID"})
}
