package repositories

import (
	"context"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user and returns the full record including the
// server-generated UUID and timestamps. The caller provides email,
// password_hash, and display_name. Role defaults to "customer" at the
// database level; we override it here if the email matches the admin
// email so the promotion happens atomically with user creation.
func (r *UserRepository) Create(ctx context.Context, email, passwordHash, displayName, role string) (*models.User, error) {
	query := `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, display_name, phone, role, created_at, updated_at
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, email, passwordHash, displayName, role).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.Phone, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

// FindByEmail returns a user by email, or pgx.ErrNoRows if not found.
// Used during login to retrieve the password hash and during signup to
// check for duplicates.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, phone, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.Phone, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &user, nil
}

// FindByID returns a user by their UUID. Used to fetch the "other user"
// profile when building conversation details.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, phone, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.Phone, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}

// ExistsByEmail returns true if a user with the given email already exists.
// This is a lighter query than FindByEmail, it only checks existence and
// doesn't scan a full row. Used during signup validation.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return exists, nil
}

// FindAll returns paginated users, optionally filtered by role. Ordered by
// newest account first so admin dashboards surface recent signups naturally.
func (r *UserRepository) FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error) {
	var rows pgx.Rows
	var err error

	if role == "" {
		query := `
			SELECT id, email, display_name, phone, role, created_at, updated_at
			FROM users
			ORDER BY created_at DESC
			OFFSET $1 LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, offset, limit)
	} else {
		query := `
			SELECT id, email, display_name, phone, role, created_at, updated_at
			FROM users
			WHERE role = $1
			ORDER BY created_at DESC
			OFFSET $2 LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, role, offset, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.DisplayName,
			&user.Phone,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}

	if users == nil {
		users = []models.User{}
	}

	return users, nil
}

// CountAll returns the total number of users, optionally filtered by role.
func (r *UserRepository) CountAll(ctx context.Context, role string) (int, error) {
	var count int

	if role == "" {
		query := `SELECT COUNT(*) FROM users`
		if err := r.pool.QueryRow(ctx, query).Scan(&count); err != nil {
			return 0, fmt.Errorf("count users: %w", err)
		}
	} else {
		query := `SELECT COUNT(*) FROM users WHERE role = $1`
		if err := r.pool.QueryRow(ctx, query, role).Scan(&count); err != nil {
			return 0, fmt.Errorf("count users by role: %w", err)
		}
	}

	return count, nil
}

// Update overwrites the editable user fields and returns the fresh public row.
// Passwords are intentionally excluded from this path; auth credentials are
// managed separately from profile/admin edits.
func (r *UserRepository) Update(ctx context.Context, id, displayName string, phone *string, role string) (*models.User, error) {
	query := `
		UPDATE users
		SET display_name = $2,
			phone = $3,
			role = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, display_name, phone, role, created_at, updated_at
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id, displayName, phone, role).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.Phone,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &user, nil
}
