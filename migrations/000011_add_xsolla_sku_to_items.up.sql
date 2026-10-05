BEGIN;

ALTER TABLE items ADD COLUMN xsolla_sku TEXT;

CREATE INDEX idx_items_xsolla_sku ON items (xsolla_sku) WHERE xsolla_sku IS NOT NULL;

COMMIT;
