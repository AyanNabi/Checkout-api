BEGIN;

ALTER TABLE xsolla_purchases ALTER COLUMN item_id DROP NOT NULL;
ALTER TABLE xsolla_purchases ALTER COLUMN sku DROP NOT NULL;
ALTER TABLE xsolla_purchases ALTER COLUMN quantity DROP NOT NULL;

ALTER TABLE xsolla_purchases DROP CONSTRAINT xsolla_purchases_quantity_check;
ALTER TABLE xsolla_purchases ADD CONSTRAINT xsolla_purchases_quantity_check CHECK (quantity IS NULL OR quantity > 0);

-- a purchase is either a whole-order (cart) checkout or a single-item checkout
ALTER TABLE xsolla_purchases ADD CONSTRAINT xsolla_purchases_order_or_item_check
    CHECK (order_id IS NOT NULL OR (item_id IS NOT NULL AND sku IS NOT NULL AND quantity IS NOT NULL));

COMMIT;
