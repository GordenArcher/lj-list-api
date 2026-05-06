ALTER TABLE products
    ADD COLUMN IF NOT EXISTS legacy_id INT;

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS old_price INT;

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS tag VARCHAR(100) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_legacy_id
    ON products (legacy_id)
    WHERE legacy_id IS NOT NULL;

UPDATE products
SET legacy_id = CASE name
    WHEN 'Royal Aroma 25kg (5*5)' THEN 101
    WHEN 'Millicent Fragrant 25kg (5*5)' THEN 109
    WHEN 'Everest Viet (Yellow) 25kg (5*5)' THEN 112
    WHEN 'Ginny Viet 25kg (5*5)' THEN 113
    WHEN 'Ginny Gold Thai 25kg (5*5)' THEN 114
    WHEN 'Everest Thai Holm 25kg (5*5) Red' THEN 115
    WHEN 'Spaghetti Big Size 400g ×20' THEN 301
    WHEN 'Frytol/Sunflower Oil 1L' THEN 201
    WHEN 'Sunflower Oil 5L' THEN 202
    WHEN 'Tasty Tom Tomato 2.2kg' THEN 401
    WHEN 'Rosa/Hondi Tomato 2.2kg' THEN 402
    WHEN 'Mackerel 420g ×24' THEN 403
    WHEN 'Mackerel 155g ×50' THEN 404
    WHEN 'Sardine ×50' THEN 405
    WHEN 'Frozen Chicken Thigh 10kg (Hard)' THEN 601
    WHEN 'Frozen Chicken Thigh 10kg (Soft)' THEN 602
    ELSE legacy_id
END
WHERE legacy_id IS NULL;

UPDATE products
SET old_price = CASE name
    WHEN 'Royal Aroma 25kg (5*5)' THEN 440
    WHEN 'Millicent Fragrant 25kg (5*5)' THEN 445
    WHEN 'Everest Viet (Yellow) 25kg (5*5)' THEN 440
    WHEN 'Ginny Viet 25kg (5*5)' THEN 370
    WHEN 'Ginny Gold Thai 25kg (5*5)' THEN 650
    WHEN 'Everest Thai Holm 25kg (5*5) Red' THEN 780
    WHEN 'Spaghetti Big Size 400g ×20' THEN 175
    WHEN 'Frytol/Sunflower Oil 1L' THEN 50
    WHEN 'Sunflower Oil 5L' THEN 210
    WHEN 'Tasty Tom Tomato 2.2kg' THEN 265
    WHEN 'Rosa/Hondi Tomato 2.2kg' THEN 55
    WHEN 'Mackerel 420g ×24' THEN 470
    WHEN 'Mackerel 155g ×50' THEN 480
    WHEN 'Sardine ×50' THEN 480
    WHEN 'Frozen Chicken Thigh 10kg (Hard)' THEN 480
    WHEN 'Frozen Chicken Thigh 10kg (Soft)' THEN 415
    ELSE old_price
END
WHERE old_price IS NULL;

UPDATE products
SET tag = CASE name
    WHEN 'Ginny Gold Thai 25kg (5*5)' THEN 'Premium'
    WHEN 'Everest Thai Holm 25kg (5*5) Red' THEN 'Premium'
    WHEN 'Frozen Chicken Thigh 10kg (Hard)' THEN 'Frozen'
    WHEN 'Frozen Chicken Thigh 10kg (Soft)' THEN 'Frozen'
    ELSE 'In Stock'
END
WHERE tag = '';
