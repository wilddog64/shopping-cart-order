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

type CartItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}
type Cart struct {
	Items       []CartItem `json:"items"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
}
type basketEnvelope struct {
	Success bool `json:"success"`
	Data    Cart `json:"data"`
}
type BasketClient struct {
	baseURL string
	http    *http.Client
}

func NewBasketClient(baseURL string) *BasketClient {
	return &BasketClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}
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
	resp, err := bc.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("basket GET /api/v1/cart: unexpected status %d", resp.StatusCode)
	}
	var env basketEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode cart: %w", err)
	}
	return &env.Data, nil
}
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
	resp, err := bc.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("basket DELETE /api/v1/cart: unexpected status %d", resp.StatusCode)
	}
	return nil
}

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
type PaymentOutcome struct {
	Paid          bool
	Status        string
	FailureReason string
}
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
	if pr.CustomerID != "" {
		req.Header.Set("X-User-ID", pr.CustomerID)
	}
	resp, err := pc.http.Do(req)
	if err != nil {
		return PaymentOutcome{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var pres paymentResponse
	_ = json.Unmarshal(raw, &pres)
	out := PaymentOutcome{Status: pres.Status}
	if pres.FailureReason != nil {
		out.FailureReason = *pres.FailureReason
	}
	out.Paid = resp.StatusCode >= 200 && resp.StatusCode < 300 && pres.Status == "COMPLETED"
	if !out.Paid && out.FailureReason == "" {
		out.FailureReason = fmt.Sprintf("payment not completed (http %d, status %q)", resp.StatusCode, pres.Status)
	}
	return out, nil
}
