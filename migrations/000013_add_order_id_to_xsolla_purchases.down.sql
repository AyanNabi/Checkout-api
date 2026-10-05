DROP INDEX IF EXISTS idx_xsolla_purchases_order_id;
ALTER TABLE xsolla_purchases DROP COLUMN IF EXISTS order_id;