DROP INDEX IF EXISTS idx_products_category_id;

ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_products_category;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;

DROP INDEX IF EXISTS idx_categories_sort_order;
DROP INDEX IF EXISTS idx_categories_active;

DROP TABLE IF EXISTS categories;
