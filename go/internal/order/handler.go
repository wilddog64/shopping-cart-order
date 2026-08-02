package order

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wilddog64/shopping-cart-order/internal/httpx"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type CreateOrderRequest struct {
	CustomerID      string                     `json:"customerId"`
	Items           []CreateOrderItemRequest   `json:"items"`
	ShippingAddress *CreateOrderAddressRequest `json:"shippingAddress"`
	Currency        string                     `json:"currency"`
}

type CreateOrderItemRequest struct {
	ProductID   string          `json:"productId"`
	ProductName string          `json:"productName"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
}

type CreateOrderAddressRequest struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type UpdateOrderStatusRequest struct {
	Status            OrderStatus `json:"status"`
	PaymentID         string      `json:"paymentId"`
	PaymentMethod     string      `json:"paymentMethod"`
	TrackingNumber    string      `json:"trackingNumber"`
	Carrier           string      `json:"carrier"`
	EstimatedDelivery *DateOnly   `json:"estimatedDelivery"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

type OrderResponse struct {
	ID                 uuid.UUID           `json:"id"`
	CustomerID         string              `json:"customerId"`
	Status             OrderStatus         `json:"status"`
	Items              []OrderItemResponse `json:"items"`
	TotalAmount        decimal.Decimal     `json:"totalAmount"`
	Currency           string              `json:"currency"`
	ShippingAddress    *AddressResponse    `json:"shippingAddress"`
	TrackingNumber     *string             `json:"trackingNumber"`
	Carrier            *string             `json:"carrier"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
	PaidAt             *time.Time          `json:"paidAt"`
	ShippedAt          *time.Time          `json:"shippedAt"`
	CompletedAt        *time.Time          `json:"completedAt"`
	CancelledAt        *time.Time          `json:"cancelledAt"`
	CancellationReason *string             `json:"cancellationReason"`
}

type OrderItemResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProductID   string          `json:"productId"`
	ProductName string          `json:"productName"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

type AddressResponse struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

func (r CreateOrderRequest) Validate() error {
	if strings.TrimSpace(r.CustomerID) == "" {
		return errors.New("customerId is required")
	}
	if len(r.Items) == 0 {
		return errors.New("items must not be empty")
	}
	for _, item := range r.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return errors.New("productId is required")
		}
		if strings.TrimSpace(item.ProductName) == "" {
			return errors.New("productName is required")
		}
		if item.Quantity <= 0 {
			return errors.New("quantity must be greater than 0")
		}
		if item.UnitPrice.Sign() <= 0 {
			return errors.New("unitPrice must be greater than 0")
		}
	}
	if r.ShippingAddress != nil {
		if err := r.ShippingAddress.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (a *CreateOrderAddressRequest) Validate() error {
	if a == nil {
		return nil
	}
	if strings.TrimSpace(a.Street) == "" || strings.TrimSpace(a.City) == "" || strings.TrimSpace(a.State) == "" ||
		strings.TrimSpace(a.PostalCode) == "" || strings.TrimSpace(a.Country) == "" {
		return errors.New("shippingAddress fields must not be blank")
	}
	return nil
}

func (r UpdateOrderStatusRequest) Validate() error {
	if !r.Status.IsValid() {
		return errors.New("status is required")
	}
	return nil
}

func (r CancelOrderRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request: "+err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	orderEntity, err := h.service.CreateOrder(c.Request.Context(), req, correlationIDFromContext(c))
	if err != nil {
		h.logger.Error("failed to create order", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create order")
		return
	}

	c.JSON(http.StatusCreated, toOrderResponse(orderEntity))
}

func (h *Handler) GetOrder(c *gin.Context) {
	id, ok := parseOrderID(c, h.logger)
	if !ok {
		return
	}

	orderEntity, err := h.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	customerID := strings.TrimSpace(httpx.GetCustomerID(c))
	if customerID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated customer required")
		return
	}
	if orderEntity.CustomerID != customerID {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(orderEntity))
}

func (h *Handler) ListOrdersByCustomer(c *gin.Context) {
	customerID := strings.TrimSpace(httpx.GetCustomerID(c))
	if customerID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated customer required")
		return
	}

	orders, err := h.service.ListOrdersByCustomer(c.Request.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to list orders", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list orders")
		return
	}

	responses := make([]OrderResponse, 0, len(orders))
	for _, orderEntity := range orders {
		responses = append(responses, toOrderResponse(orderEntity))
	}
	c.JSON(http.StatusOK, responses)
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id, ok := parseOrderID(c, h.logger)
	if !ok {
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request: "+err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	orderEntity, err := h.service.UpdateOrderStatus(c.Request.Context(), id, req, correlationIDFromContext(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(orderEntity))
}

func (h *Handler) CancelOrder(c *gin.Context) {
	id, ok := parseOrderID(c, h.logger)
	if !ok {
		return
	}

	var req CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request: "+err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	cancelledBy := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if cancelledBy == "" {
		cancelledBy = "system"
	}

	orderEntity, err := h.service.CancelOrder(c.Request.Context(), id, req.Reason, cancelledBy, correlationIDFromContext(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(orderEntity))
}

func parseOrderID(c *gin.Context, logger *zap.Logger) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid order ID")
		return uuid.Nil, false
	}
	return id, true
}

func toOrderResponse(orderEntity *Order) OrderResponse {
	response := OrderResponse{
		ID:                 orderEntity.ID,
		CustomerID:         orderEntity.CustomerID,
		Status:             orderEntity.Status,
		Items:              make([]OrderItemResponse, 0, len(orderEntity.Items)),
		TotalAmount:        orderEntity.TotalAmount.Round(2),
		Currency:           orderEntity.Currency,
		TrackingNumber:     orderEntity.TrackingNumber,
		Carrier:            orderEntity.Carrier,
		CreatedAt:          orderEntity.CreatedAt.UTC(),
		UpdatedAt:          orderEntity.UpdatedAt.UTC(),
		PaidAt:             orderEntity.PaidAt,
		ShippedAt:          orderEntity.ShippedAt,
		CompletedAt:        orderEntity.CompletedAt,
		CancelledAt:        orderEntity.CancelledAt,
		CancellationReason: orderEntity.CancellationReason,
	}

	if orderEntity.ShippingAddress != nil {
		response.ShippingAddress = &AddressResponse{
			Street:     orderEntity.ShippingAddress.Street,
			City:       orderEntity.ShippingAddress.City,
			State:      orderEntity.ShippingAddress.State,
			PostalCode: orderEntity.ShippingAddress.PostalCode,
			Country:    orderEntity.ShippingAddress.Country,
		}
	}

	for _, item := range orderEntity.Items {
		response.Items = append(response.Items, OrderItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice.Round(2),
			Subtotal:    item.Subtotal.Round(2),
		})
	}

	return response
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Code: code, Message: message})
}

func handleServiceError(c *gin.Context, err error) {
	var invalidTransition *InvalidTransitionError
	var cancelNotAllowed *CancelNotAllowedError
	switch {
	case errors.Is(err, ErrOrderNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.As(err, &invalidTransition):
		writeError(c, http.StatusBadRequest, "INVALID_STATE", err.Error())
	case errors.As(err, &cancelNotAllowed):
		writeError(c, http.StatusBadRequest, "INVALID_STATE", err.Error())
	default:
		if errors.Is(err, ErrInvalidTransition) {
			writeError(c, http.StatusBadRequest, "INVALID_STATE", err.Error())
			return
		}
		if errors.Is(err, ErrCancelNotAllowed) {
			writeError(c, http.StatusBadRequest, "INVALID_STATE", err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}

func correlationIDFromContext(c *gin.Context) string {
	if value, ok := c.Get("correlationID"); ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	if id := c.GetHeader("X-Correlation-ID"); id != "" {
		return id
	}
	return uuid.NewString()
}
