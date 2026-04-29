package repositories

import (
	"context"
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

// FindAll returns active products, optionally filtered by category.
// An empty category string returns everything. Results are paginated with
// offset and limit applied. Ordered by category then name for consistent
// display in the shop grid. Use with CountAll to calculate total pages.
func (r *ProductRepository) FindAll(ctx context.Context, category string, offset, limit int) ([]models.Product, error) {
	var rows pgx.Rows
	var err error

	if category == "" {
		query := `
			SELECT id, name, category, price, image_url, unit, active
			FROM products
			WHERE active = true
			ORDER BY category, name
			OFFSET $1 LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, offset, limit)
	} else {
		query := `
			SELECT id, name, category, price, image_url, unit, active
			FROM products
			WHERE active = true AND category = $1
			ORDER BY category, name
			OFFSET $2 LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, category, offset, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.ImageURL, &p.Unit, &p.Active); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}

	if products == nil {
		products = []models.Product{}
	}

	return products, nil
}

// CountAll returns the total number of active products, optionally filtered
// by category. Used to calculate pagination metadata (total pages, etc).
func (r *ProductRepository) CountAll(ctx context.Context, category string) (int, error) {
	var count int

	if category == "" {
		query := `SELECT COUNT(*) FROM products WHERE active = true`
		err := r.pool.QueryRow(ctx, query).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count products: %w", err)
		}
	} else {
		query := `SELECT COUNT(*) FROM products WHERE active = true AND category = $1`
		err := r.pool.QueryRow(ctx, query, category).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count products: %w", err)
		}
	}

	return count, nil
}

// FindAllCategories returns a distinct, sorted list of product categories.
// Only categories with at least one active product are included.
func (r *ProductRepository) FindAllCategories(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT category
		FROM products
		WHERE active = true
		ORDER BY category
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	if categories == nil {
		categories = []string{}
	}

	return categories, nil
}

// FindByID returns a single product by UUID. Returns pgx.ErrNoRows if
// the product doesn't exist or is inactive. Used when building cart item
// snapshots during application submission — we fetch current price and
// name to freeze them into the application.
func (r *ProductRepository) FindByID(ctx context.Context, id string) (*models.Product, error) {
	query := `
		SELECT id, name, category, price, image_url, unit, active
		FROM products
		WHERE id = $1 AND active = true
	`

	var p models.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Category, &p.Price, &p.ImageURL, &p.Unit, &p.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("find product by id: %w", err)
	}

	return &p, nil
}

// FindByIDForAdmin returns a product regardless of its active status. Admin
// product management needs this so inactive products can still be edited or
// reactivated later.
func (r *ProductRepository) FindByIDForAdmin(ctx context.Context, id string) (*models.Product, error) {
	query := `
		SELECT id, name, category, price, image_url, unit, active
		FROM products
		WHERE id = $1
	`

	var p models.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Category, &p.Price, &p.ImageURL, &p.Unit, &p.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("find product by id for admin: %w", err)
	}

	return &p, nil
}

// Create inserts a new product. Product images now live in product_images,
// so image_url is initialized as empty and later synced to the first gallery
// image for backward compatibility with clients that still read one URL.
func (r *ProductRepository) Create(ctx context.Context, name, category, unit string, price int, active bool) (*models.Product, error) {
	query := `
		INSERT INTO products (name, category, price, image_url, unit, active)
		VALUES ($1, $2, $3, '', $4, $5)
		RETURNING id, name, category, price, image_url, unit, active
	`

	var p models.Product
	err := r.pool.QueryRow(ctx, query, name, category, price, unit, active).Scan(
		&p.ID, &p.Name, &p.Category, &p.Price, &p.ImageURL, &p.Unit, &p.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}

	return &p, nil
}

// Update overwrites the editable catalog fields for a product and returns the
// fresh row. Product image management is separate; this method intentionally
// leaves image_url alone so gallery endpoints control the primary image sync.
func (r *ProductRepository) Update(ctx context.Context, id, name, category, unit string, price int, active bool) (*models.Product, error) {
	query := `
		UPDATE products
		SET name = $2,
			category = $3,
			price = $4,
			unit = $5,
			active = $6,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, category, price, image_url, unit, active
	`

	var p models.Product
	err := r.pool.QueryRow(ctx, query, id, name, category, price, unit, active).Scan(
		&p.ID, &p.Name, &p.Category, &p.Price, &p.ImageURL, &p.Unit, &p.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	return &p, nil
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
