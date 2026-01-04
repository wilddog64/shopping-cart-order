# Order Service API Reference

## Base URL

```
http://localhost:8080/api
```

In Kubernetes:
```
http://order-service.shopping-cart.svc.cluster.local/api
```

## Authentication

When OAuth2 is enabled, all API endpoints require a valid JWT token:

```http
Authorization: Bearer <jwt-token>
```

## Endpoints

### Orders

#### Create Order

```http
POST /api/orders
Content-Type: application/json
```

**Request Body:**
```json
{
  "customerId": "customer-123",
  "items": [
    {
      "productId": "prod-456",
      "productName": "Widget",
      "quantity": 2,
      "unitPrice": 29.99
    }
  ],
  "shippingAddress": {
    "street": "123 Main St",
    "city": "Seattle",
    "state": "WA",
    "zipCode": "98101",
    "country": "US"
  }
}
```

**Response:** `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customerId": "customer-123",
  "status": "PENDING",
  "items": [...],
  "totalAmount": 59.98,
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

#### Get Order by ID

```http
GET /api/orders/{orderId}
```

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customerId": "customer-123",
  "status": "CONFIRMED",
  "items": [...],
  "totalAmount": 59.98,
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:35:00Z"
}
```

#### Get Orders by Customer

```http
GET /api/orders?customerId={customerId}
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| customerId | string | Yes | Customer identifier |
| status | string | No | Filter by order status |
| page | int | No | Page number (default: 0) |
| size | int | No | Page size (default: 20) |

**Response:** `200 OK`
```json
{
  "content": [...],
  "page": 0,
  "size": 20,
  "totalElements": 45,
  "totalPages": 3
}
```

#### Update Order Status

```http
PATCH /api/orders/{orderId}/status
Content-Type: application/json
```

**Request Body:**
```json
{
  "status": "CONFIRMED"
}
```

**Response:** `200 OK`

#### Cancel Order

```http
POST /api/orders/{orderId}/cancel
```

**Response:** `200 OK`

### Health & Monitoring

#### Health Check

```http
GET /actuator/health
```

**Response:** `200 OK`
```json
{
  "status": "UP",
  "components": {
    "db": { "status": "UP" },
    "rabbit": { "status": "UP" }
  }
}
```

#### Prometheus Metrics

```http
GET /actuator/prometheus
```

## Error Responses

### Error Format

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "status": 400,
  "error": "Bad Request",
  "message": "Validation failed",
  "path": "/api/orders",
  "errors": [
    {
      "field": "customerId",
      "message": "must not be blank"
    }
  ]
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Missing or invalid token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Invalid state transition |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error |

## Rate Limiting

API requests are rate-limited per IP address:

| Limit Type | Value |
|------------|-------|
| Requests per minute | 100 |
| Requests per second | 20 |
| Burst capacity | 50 |

Rate limit headers in response:
```http
X-Rate-Limit-Remaining: 95
X-Rate-Limit-Retry-After: 0
```

## Data Types

### Order Status

| Status | Description |
|--------|-------------|
| PENDING | Order created, awaiting confirmation |
| CONFIRMED | Order confirmed by system |
| PROCESSING | Order being prepared |
| SHIPPED | Order shipped to customer |
| DELIVERED | Order delivered |
| CANCELLED | Order cancelled |

### Valid Status Transitions

| From | To |
|------|-----|
| PENDING | CONFIRMED, CANCELLED |
| CONFIRMED | PROCESSING, CANCELLED |
| PROCESSING | SHIPPED, CANCELLED |
| SHIPPED | DELIVERED |

## Examples

### cURL Examples

**Create Order:**
```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "customerId": "cust-123",
    "items": [{"productId": "p1", "productName": "Item", "quantity": 1, "unitPrice": 10.00}]
  }'
```

**Get Order:**
```bash
curl http://localhost:8080/api/orders/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $TOKEN"
```

## OpenAPI Specification

OpenAPI 3.0 specification available at:
```
GET /v3/api-docs
GET /swagger-ui.html
```
