package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) FindAll(ctx context.Context, categoryID string, offset, limit int) ([]models.Product, error) {
	return r.findAll(ctx, categoryID, offset, limit, true)
}

func (r *ProductRepository) FindAllAdmin(ctx context.Context, categoryID string, offset, limit int) ([]models.Product, error) {
	return r.findAll(ctx, categoryID, offset, limit, false)
}

func (r *ProductRepository) findAll(ctx context.Context, categoryID string, offset, limit int, activeOnly bool) ([]models.Product, error) {
	var rows pgx.Rows
	var err error

	if categoryID == "" {
		where := "WHERE "
		if activeOnly {
			where += "p.active = true AND c.active = true"
		} else {
			where += "1 = 1"
		}
		query := `
			SELECT p.id, p.name, p.legacy_id, COALESCE(p.category_id::text, ''), p.category, p.price, p.old_price, p.tag, COALESCE(NULLIF(p.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, p.image_url, p.unit, p.active
			FROM products p
			JOIN categories c ON c.id = p.category_id
		` + where + `
			ORDER BY p.category, p.name
			OFFSET $1 LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, offset, limit)
	} else {
		where := "WHERE p.category_id = $1"
		if activeOnly {
			where += " AND p.active = true AND c.active = true"
		}
		query := `
			SELECT p.id, p.name, p.legacy_id, COALESCE(p.category_id::text, ''), p.category, p.price, p.old_price, p.tag, COALESCE(NULLIF(p.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, p.image_url, p.unit, p.active
			FROM products p
			JOIN categories c ON c.id = p.category_id
		` + where + `
			ORDER BY p.category, p.name
			OFFSET $2 LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, categoryID, offset, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		var legacyID sql.NullInt64
		var oldPrice sql.NullInt64
		if err := scanProduct(rows, &p, &legacyID, &oldPrice); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		applyNullableProductFields(&p, legacyID, oldPrice)
		products = append(products, p)
	}

	if products == nil {
		products = []models.Product{}
	}

	return products, nil
}

type productScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row productScanner, p *models.Product, legacyID, oldPrice *sql.NullInt64) error {
	return row.Scan(
		&p.ID,
		&p.Name,
		legacyID,
		&p.CategoryID,
		&p.Category,
		&p.Price,
		oldPrice,
		&p.Tag,
		&p.DisplayTag,
		&p.Description,
		&p.Instructions,
		&p.RequiresInquiry,
		&p.Orderable,
		&p.ImageURL,
		&p.Unit,
		&p.Active,
	)
}

func applyNullableProductFields(p *models.Product, legacyID, oldPrice sql.NullInt64) {
	if legacyID.Valid {
		id := int(legacyID.Int64)
		p.LegacyID = &id
	}
	if oldPrice.Valid {
		price := int(oldPrice.Int64)
		p.OldPrice = &price
	}
}

func (r *ProductRepository) CountAll(ctx context.Context, categoryID string) (int, error) {
	return r.countAll(ctx, categoryID, true)
}

func (r *ProductRepository) CountAllAdmin(ctx context.Context, categoryID string) (int, error) {
	return r.countAll(ctx, categoryID, false)
}

func (r *ProductRepository) countAll(ctx context.Context, categoryID string, activeOnly bool) (int, error) {
	var count int

	if categoryID == "" {
		query := `SELECT COUNT(*) FROM products`
		if activeOnly {
			query = `
				SELECT COUNT(*)
				FROM products p
				JOIN categories c ON c.id = p.category_id
				WHERE p.active = true AND c.active = true
			`
		}
		err := r.pool.QueryRow(ctx, query).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count products: %w", err)
		}
	} else {
		query := `SELECT COUNT(*) FROM products WHERE category_id = $1`
		if activeOnly {
			query = `
				SELECT COUNT(*)
				FROM products p
				JOIN categories c ON c.id = p.category_id
				WHERE p.category_id = $1 AND p.active = true AND c.active = true
			`
		}
		err := r.pool.QueryRow(ctx, query, categoryID).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count products: %w", err)
		}
	}

	return count, nil
}

// FindAllCategories returns the active storefront categories.
func (r *ProductRepository) FindAllCategories(ctx context.Context) ([]models.Category, error) {
	query := `
		SELECT id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
		FROM categories
		WHERE active = true
		ORDER BY sort_order, name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		if err := scanCategory(rows, &cat); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	if categories == nil {
		categories = []models.Category{}
	}

	return categories, nil
}

// FindCategoryByID returns an active storefront category by UUID.
func (r *ProductRepository) FindCategoryByID(ctx context.Context, id string) (*models.Category, error) {
	query := `
		SELECT id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
		FROM categories
		WHERE id = $1 AND active = true
	`

	var cat models.Category
	if err := scanCategoryRow(r.pool.QueryRow(ctx, query, id), &cat); err != nil {
		return nil, fmt.Errorf("find category by id: %w", err)
	}
	return &cat, nil
}

// FindCategoryByName returns an active storefront category by its display name.
func (r *ProductRepository) FindCategoryByName(ctx context.Context, name string) (*models.Category, error) {
	query := `
		SELECT id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
		FROM categories
		WHERE LOWER(name) = LOWER($1) AND active = true
	`

	var cat models.Category
	if err := scanCategoryRow(r.pool.QueryRow(ctx, query, name), &cat); err != nil {
		return nil, fmt.Errorf("find category by name: %w", err)
	}
	return &cat, nil
}

// FindByID returns a single product by UUID. Returns pgx.ErrNoRows if
// the product doesn't exist or is inactive. Used when building cart item
// snapshots during application submission — we fetch current price and
// name to freeze them into the application.
func (r *ProductRepository) FindByID(ctx context.Context, id string) (*models.Product, error) {
	query := `
		SELECT p.id, p.name, p.legacy_id, COALESCE(p.category_id::text, ''), p.category, p.price, p.old_price, p.tag, COALESCE(NULLIF(p.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, p.image_url, p.unit, p.active
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE p.id = $1 AND p.active = true AND c.active = true
	`

	var p models.Product
	var legacyID sql.NullInt64
	var oldPrice sql.NullInt64
	err := scanProduct(r.pool.QueryRow(ctx, query, id), &p, &legacyID, &oldPrice)
	if err != nil {
		return nil, fmt.Errorf("find product by id: %w", err)
	}
	applyNullableProductFields(&p, legacyID, oldPrice)

	return &p, nil
}

// FindByIDForAdmin returns a product regardless of its active status. Admin
// product management needs this so inactive products can still be edited or
// reactivated later.
func (r *ProductRepository) FindByIDForAdmin(ctx context.Context, id string) (*models.Product, error) {
	query := `
		SELECT p.id, p.name, p.legacy_id, COALESCE(p.category_id::text, ''), p.category, p.price, p.old_price, p.tag, COALESCE(NULLIF(p.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, p.image_url, p.unit, p.active
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE p.id = $1
	`

	var p models.Product
	var legacyID sql.NullInt64
	var oldPrice sql.NullInt64
	err := scanProduct(r.pool.QueryRow(ctx, query, id), &p, &legacyID, &oldPrice)
	if err != nil {
		return nil, fmt.Errorf("find product by id for admin: %w", err)
	}
	applyNullableProductFields(&p, legacyID, oldPrice)

	return &p, nil
}

// Create inserts a new product. Product images now live in product_images,
// so image_url is initialized as empty and later synced to the first gallery
// image for backward compatibility with clients that still read one URL.
func (r *ProductRepository) Create(ctx context.Context, name, categoryID, categoryName, unit string, price int, oldPriceValue *int, tag string, active bool) (*models.Product, error) {
	query := `
		WITH inserted AS (
			INSERT INTO products (name, category_id, category, price, legacy_id, old_price, tag, image_url, unit, active)
			VALUES ($1, $2, $3, $4, NULL, $5, $6, '', $7, $8)
			RETURNING id, name, legacy_id, category_id, category, price, old_price, tag, image_url, unit, active
		)
		SELECT i.id, i.name, i.legacy_id, COALESCE(i.category_id::text, ''), i.category, i.price, i.old_price, i.tag, COALESCE(NULLIF(i.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, i.image_url, i.unit, i.active
		FROM inserted i
		JOIN categories c ON c.id = i.category_id
	`

	var p models.Product
	var legacyID sql.NullInt64
	var dbOldPrice sql.NullInt64
	err := scanProduct(r.pool.QueryRow(ctx, query, name, categoryID, categoryName, price, oldPriceValue, tag, unit, active), &p, &legacyID, &dbOldPrice)
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}
	applyNullableProductFields(&p, legacyID, dbOldPrice)

	return &p, nil
}

// Update overwrites the editable catalog fields for a product and returns the
// fresh row. Product image management is separate; this method intentionally
// leaves image_url alone so gallery endpoints control the primary image sync.
func (r *ProductRepository) Update(ctx context.Context, id, name, categoryID, categoryName, unit string, price int, oldPriceValue *int, tag string, active bool) (*models.Product, error) {
	query := `
		WITH updated AS (
			UPDATE products
			SET name = $2,
				category_id = $3,
				category = $4,
				price = $5,
				old_price = $6,
				tag = $7,
				unit = $8,
				active = $9,
				updated_at = NOW()
			WHERE id = $1
			RETURNING id, name, legacy_id, category_id, category, price, old_price, tag, image_url, unit, active
		)
		SELECT u.id, u.name, u.legacy_id, COALESCE(u.category_id::text, ''), u.category, u.price, u.old_price, u.tag, COALESCE(NULLIF(u.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, u.image_url, u.unit, u.active
		FROM updated u
		JOIN categories c ON c.id = u.category_id
	`

	var p models.Product
	var legacyID sql.NullInt64
	var dbOldPrice sql.NullInt64
	err := scanProduct(r.pool.QueryRow(ctx, query, id, name, categoryID, categoryName, price, oldPriceValue, tag, unit, active), &p, &legacyID, &dbOldPrice)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	applyNullableProductFields(&p, legacyID, dbOldPrice)

	return &p, nil
}

// Delete removes a product row permanently. Callers should only use this
// when the product has no dependent application history.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete product: no rows affected")
	}

	return nil
}

// FindByLegacyID returns a single product by the frontend numeric ID.
func (r *ProductRepository) FindByLegacyID(ctx context.Context, legacyID int) (*models.Product, error) {
	query := `
		SELECT p.id, p.name, p.legacy_id, COALESCE(p.category_id::text, ''), p.category, p.price, p.old_price, p.tag, COALESCE(NULLIF(p.tag, ''), c.tag, ''), COALESCE(c.description, ''), COALESCE(c.instructions, ''), c.requires_inquiry, c.orderable, p.image_url, p.unit, p.active
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE p.legacy_id = $1 AND p.active = true AND c.active = true
	`

	var p models.Product
	var dbLegacyID sql.NullInt64
	var oldPrice sql.NullInt64
	err := scanProduct(r.pool.QueryRow(ctx, query, legacyID), &p, &dbLegacyID, &oldPrice)
	if err != nil {
		return nil, fmt.Errorf("find product by legacy id: %w", err)
	}
	applyNullableProductFields(&p, dbLegacyID, oldPrice)

	return &p, nil
}

// CountApplicationsByProductID returns how many applications reference the
// given product UUID in their cart snapshot.
func (r *ProductRepository) CountApplicationsByProductID(ctx context.Context, productID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM applications
		WHERE cart_items @> $1::jsonb
	`

	payload, err := json.Marshal([]map[string]string{{"product_id": productID}})
	if err != nil {
		return 0, fmt.Errorf("marshal product reference payload: %w", err)
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, string(payload)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count applications by product id: %w", err)
	}

	return count, nil
}

// SetPrimaryImageURL updates the denormalized products.image_url field so
// existing consumers still get a representative image while the full gallery
// lives in product_images.
func (r *ProductRepository) SetPrimaryImageURL(ctx context.Context, productID, imageURL string) error {
	query := `
		UPDATE products
		SET image_url = $2,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, productID, imageURL)
	if err != nil {
		return fmt.Errorf("set primary product image url: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("set primary product image url: no rows affected")
	}

	return nil
}
