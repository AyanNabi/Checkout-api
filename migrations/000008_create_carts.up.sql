
CREATE TABLE carts (
                       id SERIAL PRIMARY KEY,
                       user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE cart_items (
                            id SERIAL PRIMARY KEY,
                            cart_id INTEGER NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
                            item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
                            quantity INTEGER NOT NULL,
                            UNIQUE(cart_id, item_id)
);

