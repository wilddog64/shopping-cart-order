package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var ErrOrderNotFound = errors.New("order not found")

type Store interface {
	Create(ctx context.Context, order *Order) error
	Get(ctx context.Context, id uuid.UUID) (*Order, error)
	ListByCustomer(ctx context.Context, customerID string) ([]*Order, error)
	Update(ctx context.Context, order *Order) error
	Close()
	Ping(ctx context.Context) error
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *PostgresStore) Create(ctx context.Context, order *Order) (err error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	order.UpdateTimestampsOnPersist()
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}

	if _, err = tx.Exec(ctx, insertOrderSQL,
		order.ID,
		order.CustomerID,
		string(order.Status),
		order.TotalAmount.StringFixed(2),
		order.Currency,
		stringOrNil(order.TrackingNumber),
		stringOrNil(order.Carrier),
		order.CreatedAt,
		order.UpdatedAt,
		timeOrNil(order.PaidAt),
		timeOrNil(order.ShippedAt),
		timeOrNil(order.CompletedAt),
		timeOrNil(order.CancelledAt),
		stringOrNil(order.CancellationReason),
		addressOrNil(order.ShippingAddress, "street"),
		addressOrNil(order.ShippingAddress, "city"),
		addressOrNil(order.ShippingAddress, "state"),
		addressOrNil(order.ShippingAddress, "postalCode"),
		addressOrNil(order.ShippingAddress, "country"),
	); err != nil {
		return err
	}

	for i := range order.Items {
		item := &order.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		item.OrderID = order.ID
		item.RecalculateSubtotal()
		if _, err = tx.Exec(ctx, insertOrderItemSQL,
			item.ID,
			order.ID,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPrice.StringFixed(2),
		); err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*Order, error) {
	order, err := s.getOrderRow(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.listItems(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Items = items
	order.RecalculateTotals()
	return order, nil
}

func (s *PostgresStore) ListByCustomer(ctx context.Context, customerID string) ([]*Order, error) {
	rows, err := s.pool.Query(ctx, listOrdersByCustomerSQL, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*Order, 0)
	orderIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		order, err := scanOrderRow(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
		orderIDs = append(orderIDs, order.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	itemsByOrder, err := s.listItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		order.Items = itemsByOrder[order.ID]
		order.RecalculateTotals()
	}
	return orders, nil
}

func (s *PostgresStore) listItemsByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]OrderItem, error) {
	itemsByOrder := make(map[uuid.UUID][]OrderItem)
	if len(orderIDs) == 0 {
		return itemsByOrder, nil
	}
	ids := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		ids[i] = id.String()
	}
	rows, err := s.pool.Query(ctx, listItemsByOrderIDsSQL, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item OrderItem
		var unitPrice string
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.Quantity, &unitPrice); err != nil {
			return nil, err
		}
		item.UnitPrice, err = decimal.NewFromString(unitPrice)
		if err != nil {
			return nil, err
		}
		item.RecalculateSubtotal()
		itemsByOrder[item.OrderID] = append(itemsByOrder[item.OrderID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return itemsByOrder, nil
}

func (s *PostgresStore) Update(ctx context.Context, order *Order) (err error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	order.UpdateTimestampsOnPersist()
	tag, err := tx.Exec(ctx, updateOrderSQL,
		order.CustomerID,
		string(order.Status),
		order.TotalAmount.StringFixed(2),
		order.Currency,
		stringOrNil(order.TrackingNumber),
		stringOrNil(order.Carrier),
		order.CreatedAt,
		order.UpdatedAt,
		timeOrNil(order.PaidAt),
		timeOrNil(order.ShippedAt),
		timeOrNil(order.CompletedAt),
		timeOrNil(order.CancelledAt),
		stringOrNil(order.CancellationReason),
		addressOrNil(order.ShippingAddress, "street"),
		addressOrNil(order.ShippingAddress, "city"),
		addressOrNil(order.ShippingAddress, "state"),
		addressOrNil(order.ShippingAddress, "postalCode"),
		addressOrNil(order.ShippingAddress, "country"),
		order.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		err = ErrOrderNotFound
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) getOrderRow(ctx context.Context, id uuid.UUID) (*Order, error) {
	row := s.pool.QueryRow(ctx, getOrderSQL, id)
	order, err := scanOrderRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

func (s *PostgresStore) listItems(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	rows, err := s.pool.Query(ctx, listItemsSQL, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		var unitPrice string
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.Quantity, &unitPrice); err != nil {
			return nil, err
		}
		item.UnitPrice, err = decimal.NewFromString(unitPrice)
		if err != nil {
			return nil, err
		}
		item.RecalculateSubtotal()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanOrderRow(row pgx.Row) (*Order, error) {
	var order Order
	var totalAmount string
	var trackingNumber sql.NullString
	var carrier sql.NullString
	var cancellationReason sql.NullString
	var paidAt sql.NullTime
	var shippedAt sql.NullTime
	var completedAt sql.NullTime
	var cancelledAt sql.NullTime
	var shippingStreet sql.NullString
	var shippingCity sql.NullString
	var shippingState sql.NullString
	var shippingPostalCode sql.NullString
	var shippingCountry sql.NullString

	err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.Status,
		&totalAmount,
		&order.Currency,
		&trackingNumber,
		&carrier,
		&order.CreatedAt,
		&order.UpdatedAt,
		&paidAt,
		&shippedAt,
		&completedAt,
		&cancelledAt,
		&cancellationReason,
		&shippingStreet,
		&shippingCity,
		&shippingState,
		&shippingPostalCode,
		&shippingCountry,
	)
	if err != nil {
		return nil, err
	}

	order.TotalAmount, err = decimal.NewFromString(totalAmount)
	if err != nil {
		return nil, err
	}

	if trackingNumber.Valid {
		order.TrackingNumber = &trackingNumber.String
	}
	if carrier.Valid {
		order.Carrier = &carrier.String
	}
	if cancellationReason.Valid {
		order.CancellationReason = &cancellationReason.String
	}
	if paidAt.Valid {
		t := paidAt.Time.UTC()
		order.PaidAt = &t
	}
	if shippedAt.Valid {
		t := shippedAt.Time.UTC()
		order.ShippedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time.UTC()
		order.CompletedAt = &t
	}
	if cancelledAt.Valid {
		t := cancelledAt.Time.UTC()
		order.CancelledAt = &t
	}

	if shippingStreet.Valid || shippingCity.Valid || shippingState.Valid || shippingPostalCode.Valid || shippingCountry.Valid {
		order.ShippingAddress = &ShippingAddress{
			Street:     shippingStreet.String,
			City:       shippingCity.String,
			State:      shippingState.String,
			PostalCode: shippingPostalCode.String,
			Country:    shippingCountry.String,
		}
	}

	order.Items = []OrderItem{}
	return &order, nil
}

func stringOrNil(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func timeOrNil(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func addressOrNil(address *ShippingAddress, field string) any {
	if address == nil {
		return nil
	}
	switch field {
	case "street":
		return address.Street
	case "city":
		return address.City
	case "state":
		return address.State
	case "postalCode":
		return address.PostalCode
	case "country":
		return address.Country
	default:
		return nil
	}
}

const getOrderSQL = `
SELECT id, customer_id, status, total_amount, currency, tracking_number, carrier,
       created_at, updated_at, paid_at, shipped_at, completed_at, cancelled_at,
       cancellation_reason, shipping_street, shipping_city, shipping_state,
       shipping_postal_code, shipping_country
FROM orders
WHERE id = $1
`

const listOrdersByCustomerSQL = `
SELECT id, customer_id, status, total_amount, currency, tracking_number, carrier,
       created_at, updated_at, paid_at, shipped_at, completed_at, cancelled_at,
       cancellation_reason, shipping_street, shipping_city, shipping_state,
       shipping_postal_code, shipping_country
FROM orders
WHERE customer_id = $1
ORDER BY created_at ASC, id ASC
`

const listItemsSQL = `
SELECT id, order_id, product_id, product_name, quantity, unit_price
FROM order_items
WHERE order_id = $1
ORDER BY id ASC
`

const listItemsByOrderIDsSQL = `
SELECT id, order_id, product_id, product_name, quantity, unit_price
FROM order_items
WHERE order_id = ANY($1::uuid[])
ORDER BY id ASC
`

const insertOrderSQL = `
INSERT INTO orders (
    id, customer_id, status, total_amount, currency, tracking_number, carrier,
    created_at, updated_at, paid_at, shipped_at, completed_at, cancelled_at,
    cancellation_reason, shipping_street, shipping_city, shipping_state,
    shipping_postal_code, shipping_country
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19
)
`

const updateOrderSQL = `
UPDATE orders SET
    customer_id = $1,
    status = $2,
    total_amount = $3,
    currency = $4,
    tracking_number = $5,
    carrier = $6,
    created_at = $7,
    updated_at = $8,
    paid_at = $9,
    shipped_at = $10,
    completed_at = $11,
    cancelled_at = $12,
    cancellation_reason = $13,
    shipping_street = $14,
    shipping_city = $15,
    shipping_state = $16,
    shipping_postal_code = $17,
    shipping_country = $18
WHERE id = $19
`

const insertOrderItemSQL = `
INSERT INTO order_items (
    id, order_id, product_id, product_name, quantity, unit_price
) VALUES (
    $1, $2, $3, $4, $5, $6
)
`
