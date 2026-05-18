package repositories

import (
	"context"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) List(ctx context.Context, includeInactive bool) ([]models.Category, error) {
	query := `
		SELECT id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
		FROM categories
	`
	if !includeInactive {
		query += " WHERE active = true"
	}
	query += " ORDER BY sort_order, name"

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

func (r *CategoryRepository) FindByID(ctx context.Context, id string, includeInactive bool) (*models.Category, error) {
	query := `
		SELECT id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
		FROM categories
		WHERE id = $1
	`
	if !includeInactive {
		query += " AND active = true"
	}

	var cat models.Category
	if err := scanCategoryRow(r.pool.QueryRow(ctx, query, id), &cat); err != nil {
		return nil, fmt.Errorf("find category by id: %w", err)
	}
	return &cat, nil
}

func (r *CategoryRepository) FindByName(ctx context.Context, name string, includeInactive bool) (*models.Category, error) {
	query := `
		SELECT id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
		FROM categories
		WHERE LOWER(name) = LOWER($1)
	`
	if !includeInactive {
		query += " AND active = true"
	}

	var cat models.Category
	if err := scanCategoryRow(r.pool.QueryRow(ctx, query, name), &cat); err != nil {
		return nil, fmt.Errorf("find category by name: %w", err)
	}
	return &cat, nil
}

func (r *CategoryRepository) Create(ctx context.Context, cat *models.Category) (*models.Category, error) {
	query := `
		INSERT INTO categories (sort_order, name, description, instructions, tag, requires_inquiry, orderable, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
	`

	var result models.Category
	if err := scanCategoryRow(r.pool.QueryRow(ctx, query, cat.SortOrder, cat.Name, cat.Description, cat.Instructions, cat.Tag, cat.RequiresInquiry, cat.Orderable, cat.Active), &result); err != nil {
		return nil, fmt.Errorf("insert category: %w", err)
	}
	return &result, nil
}

func (r *CategoryRepository) Update(ctx context.Context, id string, cat *models.Category) (*models.Category, error) {
	query := `
		UPDATE categories
		SET sort_order = $2,
			name = $3,
			description = $4,
			instructions = $5,
			tag = $6,
			requires_inquiry = $7,
			orderable = $8,
			active = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, sort_order, name, COALESCE(description, ''), COALESCE(instructions, ''), COALESCE(tag, ''), requires_inquiry, orderable, active
	`

	var result models.Category
	if err := scanCategoryRow(r.pool.QueryRow(ctx, query, id, cat.SortOrder, cat.Name, cat.Description, cat.Instructions, cat.Tag, cat.RequiresInquiry, cat.Orderable, cat.Active), &result); err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	return &result, nil
}

type categoryScanner interface {
	Scan(dest ...any) error
}

func scanCategoryRow(row categoryScanner, cat *models.Category) error {
	return row.Scan(
		&cat.ID,
		&cat.SortOrder,
		&cat.Name,
		&cat.Description,
		&cat.Instructions,
		&cat.Tag,
		&cat.RequiresInquiry,
		&cat.Orderable,
		&cat.Active,
	)
}

func scanCategory(rows categoryScanner, cat *models.Category) error {
	return scanCategoryRow(rows, cat)
}

func (r *CategoryRepository) Deactivate(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE categories
		SET active = false,
			updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("deactivate category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM categories
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) CountProductsByCategoryID(ctx context.Context, categoryID string) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE category_id = $1`, categoryID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count products by category: %w", err)
	}
	return count, nil
}

func (r *CategoryRepository) UpdateProductCategoryName(ctx context.Context, categoryID, categoryName string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE products
		SET category = $2,
			updated_at = NOW()
		WHERE category_id = $1
	`, categoryID, categoryName)
	if err != nil {
		return fmt.Errorf("update product category names: %w", err)
	}
	return nil
}
