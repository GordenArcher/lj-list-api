DROP INDEX IF EXISTS idx_products_legacy_id;

ALTER TABLE products
    DROP COLUMN IF EXISTS legacy_id;

ALTER TABLE products
    DROP COLUMN IF EXISTS old_price;

ALTER TABLE products
    DROP COLUMN IF EXISTS tag;
