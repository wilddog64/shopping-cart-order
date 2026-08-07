CREATE TABLE IF NOT EXISTS orders (
    id                   UUID PRIMARY KEY,
    customer_id          VARCHAR(255)  NOT NULL,
    status               VARCHAR(255)  NOT NULL,
    total_amount         NUMERIC(10,2) NOT NULL,
    currency             VARCHAR(3)     NOT NULL,
    tracking_number      VARCHAR(255),
    carrier              VARCHAR(255),
    created_at           TIMESTAMPTZ    NOT NULL,
    updated_at           TIMESTAMPTZ,
    paid_at              TIMESTAMPTZ,
    shipped_at           TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    cancelled_at         TIMESTAMPTZ,
    cancellation_reason  VARCHAR(255),
    shipping_street      VARCHAR(255),
    shipping_city        VARCHAR(255),
    shipping_state       VARCHAR(255),
    shipping_postal_code VARCHAR(255),
    shipping_country     VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS order_items (
    id            UUID PRIMARY KEY,
    order_id      UUID           NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id    VARCHAR(255)   NOT NULL,
    product_name  VARCHAR(255)   NOT NULL,
    quantity      INTEGER        NOT NULL,
    unit_price    NUMERIC(10,2)  NOT NULL,
    total_price   NUMERIC(10,2)  NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders (customer_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);
