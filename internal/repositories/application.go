package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplicationRepository struct {
	pool *pgxpool.Pool
}

func NewApplicationRepository(pool *pgxpool.Pool) *ApplicationRepository {
	return &ApplicationRepository{pool: pool}
}

// Create inserts a new application. CartItems is serialized to JSONB.
// The caller is responsible for validating that the total meets the
// minimum order threshold and that all product IDs are valid.
func (r *ApplicationRepository) Create(ctx context.Context, app *models.Application) (*models.Application, error) {
	cartJSON, err := json.Marshal(app.CartItems)
	if err != nil {
		return nil, fmt.Errorf("marshal cart items: %w", err)
	}

	query := `
		INSERT INTO applications (user_id, package_type, package_name, cart_items, total_amount, monthly_amount, staff_number, mandate_number, institution, ghana_card_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, user_id, package_type, package_name, cart_items, total_amount, monthly_amount, status, staff_number, mandate_number, institution, ghana_card_number, created_at, updated_at
	`

	var result models.Application
	var cartBytes []byte

	err = r.pool.QueryRow(ctx, query,
		app.UserID, app.PackageType, app.PackageName, cartJSON,
		app.TotalAmount, app.MonthlyAmount, app.StaffNumber,
		app.MandateNumber, app.Institution, app.GhanaCardNumber,
	).Scan(
		&result.ID, &result.UserID, &result.PackageType, &result.PackageName,
		&cartBytes, &result.TotalAmount, &result.MonthlyAmount, &result.Status,
		&result.StaffNumber, &result.MandateNumber, &result.Institution,
		&result.GhanaCardNumber, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert application: %w", err)
	}

	if err := json.Unmarshal(cartBytes, &result.CartItems); err != nil {
		return nil, fmt.Errorf("unmarshal cart items: %w", err)
	}

	return &result, nil
}

// FindByUserID returns paginated applications for a given user, newest first.
// Use with CountByUserID to calculate pagination metadata.
func (r *ApplicationRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]models.Application, error) {
	query := `
		SELECT id, user_id, package_type, package_name, cart_items, total_amount, monthly_amount, status, staff_number, mandate_number, institution, ghana_card_number, created_at, updated_at
		FROM applications
		WHERE user_id = $1
		ORDER BY created_at DESC
		OFFSET $2 LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query applications by user: %w", err)
	}
	defer rows.Close()

	return scanApplications(rows)
}

// CountByUserID returns the total number of applications for a user.
func (r *ApplicationRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM applications WHERE user_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count applications by user: %w", err)
	}

	return count, nil
}

// FindAll returns paginated applications, optionally filtered by status.
// Used by the admin dashboard. Use with CountAll to calculate pagination metadata.
func (r *ApplicationRepository) FindAll(ctx context.Context, status string, offset, limit int) ([]models.Application, error) {
	var rows pgx.Rows
	var err error

	if status == "" {
		query := `
			SELECT
				a.id,
				a.user_id,
				u.id,
				u.display_name,
				COALESCE(u.phone_number, ''),
				u.role,
				a.package_type,
				a.package_name,
				a.cart_items,
				a.total_amount,
				a.monthly_amount,
				a.status,
				a.staff_number,
				a.mandate_number,
				a.institution,
				a.ghana_card_number,
				a.created_at,
				a.updated_at
			FROM applications a
			JOIN users u ON u.id = a.user_id
			ORDER BY a.created_at DESC
			OFFSET $1 LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, offset, limit)
	} else {
		query := `
			SELECT
				a.id,
				a.user_id,
				u.id,
				u.display_name,
				COALESCE(u.phone_number, ''),
				u.role,
				a.package_type,
				a.package_name,
				a.cart_items,
				a.total_amount,
				a.monthly_amount,
				a.status,
				a.staff_number,
				a.mandate_number,
				a.institution,
				a.ghana_card_number,
				a.created_at,
				a.updated_at
			FROM applications a
			JOIN users u ON u.id = a.user_id
			WHERE a.status = $1
			ORDER BY a.created_at DESC
			OFFSET $2 LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, status, offset, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query all applications: %w", err)
	}
	defer rows.Close()

	return scanApplicationsWithCustomer(rows)
}

// CountAll returns the total number of applications, optionally filtered by status.
func (r *ApplicationRepository) CountAll(ctx context.Context, status string) (int, error) {
	var count int

	if status == "" {
		query := `SELECT COUNT(*) FROM applications`
		err := r.pool.QueryRow(ctx, query).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count all applications: %w", err)
		}
	} else {
		query := `SELECT COUNT(*) FROM applications WHERE status = $1`
		err := r.pool.QueryRow(ctx, query, status).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count applications by status: %w", err)
		}
	}

	return count, nil
}

// FindByID returns a single application by UUID. Returns pgx.ErrNoRows if
// not found.
func (r *ApplicationRepository) FindByID(ctx context.Context, id string) (*models.Application, error) {
	query := `
		SELECT id, user_id, package_type, package_name, cart_items, total_amount, monthly_amount, status, staff_number, mandate_number, institution, ghana_card_number, created_at, updated_at
		FROM applications
		WHERE id = $1
	`

	var app models.Application
	var cartBytes []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&app.ID, &app.UserID, &app.PackageType, &app.PackageName,
		&cartBytes, &app.TotalAmount, &app.MonthlyAmount, &app.Status,
		&app.StaffNumber, &app.MandateNumber, &app.Institution,
		&app.GhanaCardNumber, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find application by id: %w", err)
	}

	if err := json.Unmarshal(cartBytes, &app.CartItems); err != nil {
		return nil, fmt.Errorf("unmarshal cart items: %w", err)
	}

	return &app, nil
}

// UpdateStatus changes the status of an application. Only the admin can
// call this. The updated_at timestamp is refreshed automatically.
func (r *ApplicationRepository) UpdateStatus(ctx context.Context, id, status string) (*models.Application, error) {
	query := `
		UPDATE applications
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, package_type, package_name, cart_items, total_amount, monthly_amount, status, staff_number, mandate_number, institution, ghana_card_number, created_at, updated_at
	`

	var app models.Application
	var cartBytes []byte

	err := r.pool.QueryRow(ctx, query, id, status).Scan(
		&app.ID, &app.UserID, &app.PackageType, &app.PackageName,
		&cartBytes, &app.TotalAmount, &app.MonthlyAmount, &app.Status,
		&app.StaffNumber, &app.MandateNumber, &app.Institution,
		&app.GhanaCardNumber, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update application status: %w", err)
	}

	if err := json.Unmarshal(cartBytes, &app.CartItems); err != nil {
		return nil, fmt.Errorf("unmarshal cart items: %w", err)
	}

	return &app, nil
}

// scanApplications is a shared row scanner. Extracted because FindByUserID
// and FindAll have identical scan logic. The JSONB cart_items column is
// decoded back into []CartItem for every row.
func scanApplications(rows pgx.Rows) ([]models.Application, error) {
	var apps []models.Application
	for rows.Next() {
		var app models.Application
		var cartBytes []byte
		if err := rows.Scan(
			&app.ID, &app.UserID, &app.PackageType, &app.PackageName,
			&cartBytes, &app.TotalAmount, &app.MonthlyAmount, &app.Status,
			&app.StaffNumber, &app.MandateNumber, &app.Institution,
			&app.GhanaCardNumber, &app.CreatedAt, &app.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		if err := json.Unmarshal(cartBytes, &app.CartItems); err != nil {
			return nil, fmt.Errorf("unmarshal cart items: %w", err)
		}
		apps = append(apps, app)
	}

	if apps == nil {
		apps = []models.Application{}
	}

	return apps, nil
}

func scanApplicationsWithCustomer(rows pgx.Rows) ([]models.Application, error) {
	var apps []models.Application
	for rows.Next() {
		var app models.Application
		var cartBytes []byte
		app.Customer = &models.ApplicationCustomer{}

		if err := rows.Scan(
			&app.ID,
			&app.UserID,
			&app.Customer.ID,
			&app.Customer.DisplayName,
			&app.Customer.PhoneNumber,
			&app.Customer.Role,
			&app.PackageType,
			&app.PackageName,
			&cartBytes,
			&app.TotalAmount,
			&app.MonthlyAmount,
			&app.Status,
			&app.StaffNumber,
			&app.MandateNumber,
			&app.Institution,
			&app.GhanaCardNumber,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan admin application: %w", err)
		}

		if err := json.Unmarshal(cartBytes, &app.CartItems); err != nil {
			return nil, fmt.Errorf("unmarshal cart items: %w", err)
		}
		apps = append(apps, app)
	}

	if apps == nil {
		apps = []models.Application{}
	}

	return apps, nil
}
