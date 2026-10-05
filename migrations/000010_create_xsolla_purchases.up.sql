BEGIN;

CREATE TABLE xsolla_purchases (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id INTEGER REFERENCES orders(id) ON DELETE SET NULL,
    item_id INTEGER NOT NULL REFERENCES items(id),
    sku TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    xsolla_transaction_id TEXT UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'paid', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_xsolla_purchases_user_id ON xsolla_purchases (user_id, created_at DESC);
CREATE INDEX idx_xsolla_purchases_status ON xsolla_purchases (status);
CREATE INDEX idx_xsolla_purchases_order_id ON xsolla_purchases (order_id);

COMMIT;
