BEGIN;

DROP INDEX IF EXISTS idx_items_xsolla_sku;
ALTER TABLE items DROP COLUMN IF EXISTS xsolla_sku;

COMMIT;
