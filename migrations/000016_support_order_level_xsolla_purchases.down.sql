BEGIN;

ALTER TABLE xsolla_purchases DROP CONSTRAINT xsolla_purchases_order_or_item_check;

ALTER TABLE xsolla_purchases DROP CONSTRAINT xsolla_purchases_quantity_check;
ALTER TABLE xsolla_purchases ADD CONSTRAINT xsolla_purchases_quantity_check CHECK (quantity > 0);

ALTER TABLE xsolla_purchases ALTER COLUMN item_id SET NOT NULL;
ALTER TABLE xsolla_purchases ALTER COLUMN sku SET NOT NULL;
ALTER TABLE xsolla_purchases ALTER COLUMN quantity SET NOT NULL;

COMMIT;
