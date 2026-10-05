-- items 31-40 were seeded with placeholder SKUs that don't exist in the Xsolla project catalog
UPDATE items SET xsolla_sku = NULL WHERE id BETWEEN 31 AND 40;
