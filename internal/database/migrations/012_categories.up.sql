CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sort_order INT NOT NULL DEFAULT 0,
    name VARCHAR(100) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE products
    ADD COLUMN category_id UUID;

UPDATE products p
SET category_id = c.id
FROM categories c
WHERE p.category = c.name;

ALTER TABLE products
    ALTER COLUMN category_id SET NOT NULL;

ALTER TABLE products
    ADD CONSTRAINT fk_products_category
    FOREIGN KEY (category_id) REFERENCES categories(id);

CREATE INDEX idx_categories_active ON categories (active);
CREATE INDEX idx_categories_sort_order ON categories (sort_order, name);
CREATE INDEX idx_products_category_id ON products (category_id);
