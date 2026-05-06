package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PackageRepository struct {
	pool *pgxpool.Pool
}

func NewPackageRepository(pool *pgxpool.Pool) *PackageRepository {
	return &PackageRepository{pool: pool}
}

func (r *PackageRepository) ListFixed(ctx context.Context, includeInactive bool) ([]models.FixedPackage, error) {
	query := `
		SELECT id, sort_order, name, tagline, price, monthly, tag, popular, rice_options, items, active
		FROM fixed_packages
	`
	args := []any{}
	if !includeInactive {
		query += " WHERE active = true"
	}
	query += " ORDER BY sort_order, name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query fixed packages: %w", err)
	}
	defer rows.Close()

	var out []models.FixedPackage
	for rows.Next() {
		var pkg models.FixedPackage
		var itemsBytes []byte
		var active bool
		var sortOrder int
		if err := rows.Scan(
			&pkg.ID, &sortOrder, &pkg.Name, &pkg.Tagline, &pkg.Price, &pkg.Monthly, &pkg.Tag, &pkg.Popular, &pkg.RiceOptions, &itemsBytes, &active,
		); err != nil {
			return nil, fmt.Errorf("scan fixed package: %w", err)
		}
		pkg.SortOrder = sortOrder
		if err := json.Unmarshal(itemsBytes, &pkg.Items); err != nil {
			return nil, fmt.Errorf("unmarshal fixed package items: %w", err)
		}
		out = append(out, pkg)
	}
	if out == nil {
		out = []models.FixedPackage{}
	}
	return out, nil
}

func (r *PackageRepository) FindFixedByID(ctx context.Context, id string, includeInactive bool) (*models.FixedPackage, error) {
	query := `
		SELECT id, sort_order, name, tagline, price, monthly, tag, popular, rice_options, items, active
		FROM fixed_packages
		WHERE id = $1
	`
	args := []any{id}
	if !includeInactive {
		query += " AND active = true"
	}

	row := r.pool.QueryRow(ctx, query, args...)
	var pkg models.FixedPackage
	var itemsBytes []byte
	var active bool
	var sortOrder int
	if err := row.Scan(
		&pkg.ID, &sortOrder, &pkg.Name, &pkg.Tagline, &pkg.Price, &pkg.Monthly, &pkg.Tag, &pkg.Popular, &pkg.RiceOptions, &itemsBytes, &active,
	); err != nil {
		return nil, fmt.Errorf("scan fixed package: %w", err)
	}
	pkg.SortOrder = sortOrder
	if err := json.Unmarshal(itemsBytes, &pkg.Items); err != nil {
		return nil, fmt.Errorf("unmarshal fixed package items: %w", err)
	}
	return &pkg, nil
}

func (r *PackageRepository) FindFixedByName(ctx context.Context, name string, includeInactive bool) (*models.FixedPackage, error) {
	query := `
		SELECT id, sort_order, name, tagline, price, monthly, tag, popular, rice_options, items, active
		FROM fixed_packages
		WHERE LOWER(name) = LOWER($1)
	`
	args := []any{name}
	if !includeInactive {
		query += " AND active = true"
	}

	row := r.pool.QueryRow(ctx, query, args...)
	var pkg models.FixedPackage
	var itemsBytes []byte
	var active bool
	var sortOrder int
	if err := row.Scan(
		&pkg.ID, &sortOrder, &pkg.Name, &pkg.Tagline, &pkg.Price, &pkg.Monthly, &pkg.Tag, &pkg.Popular, &pkg.RiceOptions, &itemsBytes, &active,
	); err != nil {
		return nil, fmt.Errorf("scan fixed package: %w", err)
	}
	pkg.SortOrder = sortOrder
	if err := json.Unmarshal(itemsBytes, &pkg.Items); err != nil {
		return nil, fmt.Errorf("unmarshal fixed package items: %w", err)
	}
	return &pkg, nil
}

func (r *PackageRepository) CreateFixed(ctx context.Context, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error) {
	itemsJSON, err := json.Marshal(pkg.Items)
	if err != nil {
		return nil, fmt.Errorf("marshal fixed package items: %w", err)
	}

	query := `
		INSERT INTO fixed_packages (id, sort_order, name, tagline, price, monthly, tag, popular, rice_options, items, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true)
		RETURNING id, sort_order, name, tagline, price, monthly, tag, popular, rice_options, items, active
	`

	var result models.FixedPackage
	var itemsBytes []byte
	var active bool
	err = r.pool.QueryRow(ctx, query,
		pkg.ID, sortOrder, pkg.Name, pkg.Tagline, pkg.Price, pkg.Monthly, pkg.Tag, pkg.Popular, pkg.RiceOptions, itemsJSON,
	).Scan(
		&result.ID, &sortOrder, &result.Name, &result.Tagline, &result.Price, &result.Monthly, &result.Tag, &result.Popular, &result.RiceOptions, &itemsBytes, &active,
	)
	if err != nil {
		return nil, fmt.Errorf("insert fixed package: %w", err)
	}
	result.SortOrder = sortOrder

	if err := json.Unmarshal(itemsBytes, &result.Items); err != nil {
		return nil, fmt.Errorf("unmarshal fixed package items: %w", err)
	}
	return &result, nil
}

func (r *PackageRepository) UpdateFixed(ctx context.Context, id string, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error) {
	itemsJSON, err := json.Marshal(pkg.Items)
	if err != nil {
		return nil, fmt.Errorf("marshal fixed package items: %w", err)
	}

	query := `
		UPDATE fixed_packages
		SET sort_order = $2,
			name = $3,
			tagline = $4,
			price = $5,
			monthly = $6,
			tag = $7,
			popular = $8,
			rice_options = $9,
			items = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, sort_order, name, tagline, price, monthly, tag, popular, rice_options, items, active
	`

	var result models.FixedPackage
	var itemsBytes []byte
	var active bool
	err = r.pool.QueryRow(ctx, query,
		id, sortOrder, pkg.Name, pkg.Tagline, pkg.Price, pkg.Monthly, pkg.Tag, pkg.Popular, pkg.RiceOptions, itemsJSON,
	).Scan(
		&result.ID, &sortOrder, &result.Name, &result.Tagline, &result.Price, &result.Monthly, &result.Tag, &result.Popular, &result.RiceOptions, &itemsBytes, &active,
	)
	if err != nil {
		return nil, fmt.Errorf("update fixed package: %w", err)
	}
	result.SortOrder = sortOrder

	if err := json.Unmarshal(itemsBytes, &result.Items); err != nil {
		return nil, fmt.Errorf("unmarshal fixed package items: %w", err)
	}
	return &result, nil
}

func (r *PackageRepository) DeleteFixed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE fixed_packages
		SET active = false,
			updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete fixed package: %w", err)
	}
	return nil
}

func (r *PackageRepository) ListDepartment(ctx context.Context, kind string, includeInactive bool) ([]models.SimplePackage, error) {
	query := `
		SELECT id, kind, sort_order, name, price, items, active
		FROM department_packages
		WHERE kind = $1
	`
	args := []any{kind}
	if !includeInactive {
		query += " AND active = true"
	}
	query += " ORDER BY sort_order, name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query department packages: %w", err)
	}
	defer rows.Close()

	var out []models.SimplePackage
	for rows.Next() {
		var pkg models.SimplePackage
		var _kind string
		var sortOrder int
		var active bool
		if err := rows.Scan(&pkg.ID, &_kind, &sortOrder, &pkg.Name, &pkg.Price, &pkg.Items, &active); err != nil {
			return nil, fmt.Errorf("scan department package: %w", err)
		}
		pkg.SortOrder = sortOrder
		out = append(out, pkg)
	}
	if out == nil {
		out = []models.SimplePackage{}
	}
	return out, nil
}

func (r *PackageRepository) FindDepartmentByID(ctx context.Context, kind, id string, includeInactive bool) (*models.SimplePackage, error) {
	query := `
		SELECT id, kind, sort_order, name, price, items, active
		FROM department_packages
		WHERE kind = $1 AND id = $2
	`
	args := []any{kind, id}
	if !includeInactive {
		query += " AND active = true"
	}

	var pkg models.SimplePackage
	var _kind string
	var sortOrder int
	var active bool
	err := r.pool.QueryRow(ctx, query, args...).Scan(&pkg.ID, &_kind, &sortOrder, &pkg.Name, &pkg.Price, &pkg.Items, &active)
	if err != nil {
		return nil, fmt.Errorf("find department package: %w", err)
	}
	pkg.SortOrder = sortOrder
	return &pkg, nil
}

func (r *PackageRepository) CreateDepartment(ctx context.Context, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error) {
	query := `
		INSERT INTO department_packages (id, kind, sort_order, name, price, items, active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id, kind, sort_order, name, price, items, active
	`

	var result models.SimplePackage
	var _kind string
	var active bool
	err := r.pool.QueryRow(ctx, query, pkg.ID, kind, sortOrder, pkg.Name, pkg.Price, pkg.Items).Scan(
		&result.ID, &_kind, &sortOrder, &result.Name, &result.Price, &result.Items, &active,
	)
	if err != nil {
		return nil, fmt.Errorf("insert department package: %w", err)
	}
	result.SortOrder = sortOrder
	return &result, nil
}

func (r *PackageRepository) UpdateDepartment(ctx context.Context, id, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error) {
	query := `
		UPDATE department_packages
		SET kind = $2,
			sort_order = $3,
			name = $4,
			price = $5,
			items = $6,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, kind, sort_order, name, price, items, active
	`

	var result models.SimplePackage
	var _kind string
	var active bool
	err := r.pool.QueryRow(ctx, query, id, kind, sortOrder, pkg.Name, pkg.Price, pkg.Items).Scan(
		&result.ID, &_kind, &sortOrder, &result.Name, &result.Price, &result.Items, &active,
	)
	if err != nil {
		return nil, fmt.Errorf("update department package: %w", err)
	}
	result.SortOrder = sortOrder
	return &result, nil
}

func (r *PackageRepository) DeleteDepartment(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE department_packages
		SET active = false,
			updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete department package: %w", err)
	}
	return nil
}
