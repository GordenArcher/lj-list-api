package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

type CreateUserInput struct {
	PasswordHash    string
	DisplayName     string
	PhoneNumber     string
	StaffNumber     string
	Institution     string
	GhanaCardNumber string
	IsActive        bool
	OTPHash         *string
	OTPExpiresAt    *time.Time
	Role            string
}

type UpdateUserInput struct {
	DisplayName     string
	PhoneNumber     string
	StaffNumber     string
	Institution     string
	GhanaCardNumber string
	PasswordHash    *string
	Role            string
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user and returns the full record including the
// server-generated UUID and timestamps. Role defaults to "customer" at the
// database level; we override it here if the phone number matches the
// configured admin phone so the promotion happens atomically with creation.
func (r *UserRepository) Create(ctx context.Context, input CreateUserInput) (*models.User, error) {
	query := `
		INSERT INTO users (
			password_hash,
			display_name,
			phone_number,
			staff_number,
			institution,
			ghana_card_number,
			is_active,
			otp_hash,
			otp_expires_at,
			role
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING
			id,
			password_hash,
			display_name,
			COALESCE(phone_number, ''),
			COALESCE(staff_number, ''),
			COALESCE(institution, ''),
			COALESCE(ghana_card_number, ''),
			is_active,
			otp_hash,
			otp_expires_at,
			role,
			created_at,
			updated_at
	`

	var user models.User
	err := r.pool.QueryRow(
		ctx,
		query,
		input.PasswordHash,
		input.DisplayName,
		input.PhoneNumber,
		input.StaffNumber,
		input.Institution,
		input.GhanaCardNumber,
		input.IsActive,
		input.OTPHash,
		input.OTPExpiresAt,
		input.Role,
	).Scan(
		&user.ID,
		&user.PasswordHash,
		&user.DisplayName,
		&user.PhoneNumber,
		&user.StaffNumber,
		&user.Institution,
		&user.GhanaCardNumber,
		&user.IsActive,
		&user.OTPHash,
		&user.OTPExpiresAt,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

// FindByPhoneNumber returns a user by phone_number, or pgx.ErrNoRows if not
// found. Used during login, OTP verification, and admin lookup.
func (r *UserRepository) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	query := `
		SELECT
			id,
			password_hash,
			display_name,
			COALESCE(phone_number, ''),
			COALESCE(staff_number, ''),
			COALESCE(institution, ''),
			COALESCE(ghana_card_number, ''),
			is_active,
			otp_hash,
			otp_expires_at,
			role,
			created_at,
			updated_at
		FROM users
		WHERE phone_number = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, phoneNumber).Scan(
		&user.ID,
		&user.PasswordHash,
		&user.DisplayName,
		&user.PhoneNumber,
		&user.StaffNumber,
		&user.Institution,
		&user.GhanaCardNumber,
		&user.IsActive,
		&user.OTPHash,
		&user.OTPExpiresAt,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find user by phone number: %w", err)
	}

	return &user, nil
}

// FindByID returns a user by their UUID. Used to fetch the "other user"
// profile when building conversation details.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT
			id,
			password_hash,
			display_name,
			COALESCE(phone_number, ''),
			COALESCE(staff_number, ''),
			COALESCE(institution, ''),
			COALESCE(ghana_card_number, ''),
			is_active,
			otp_hash,
			otp_expires_at,
			role,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.PasswordHash,
		&user.DisplayName,
		&user.PhoneNumber,
		&user.StaffNumber,
		&user.Institution,
		&user.GhanaCardNumber,
		&user.IsActive,
		&user.OTPHash,
		&user.OTPExpiresAt,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) UpdateActivationOTP(ctx context.Context, userID string, otpHash string, otpExpiresAt time.Time) error {
	query := `
		UPDATE users
		SET otp_hash = $2,
			otp_expires_at = $3,
			is_active = FALSE,
			updated_at = NOW()
		WHERE id = $1
	`

	if _, err := r.pool.Exec(ctx, query, userID, otpHash, otpExpiresAt); err != nil {
		return fmt.Errorf("update activation otp: %w", err)
	}

	return nil
}

func (r *UserRepository) Activate(ctx context.Context, userID string) (*models.User, error) {
	query := `
		UPDATE users
		SET is_active = TRUE,
			otp_hash = NULL,
			otp_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			password_hash,
			display_name,
			COALESCE(phone_number, ''),
			COALESCE(staff_number, ''),
			COALESCE(institution, ''),
			COALESCE(ghana_card_number, ''),
			is_active,
			otp_hash,
			otp_expires_at,
			role,
			created_at,
			updated_at
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.PasswordHash,
		&user.DisplayName,
		&user.PhoneNumber,
		&user.StaffNumber,
		&user.Institution,
		&user.GhanaCardNumber,
		&user.IsActive,
		&user.OTPHash,
		&user.OTPExpiresAt,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("activate user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) DeleteByID(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	if _, err := r.pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("delete user by id: %w", err)
	}

	return nil
}

func (r *UserRepository) ExistsByPhoneNumber(ctx context.Context, phoneNumber string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE phone_number = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, phoneNumber).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists by phone number: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByPhoneNumberExcludingID(ctx context.Context, phoneNumber, userID string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE phone_number = $1 AND id <> $2)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, phoneNumber, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists by phone number excluding id: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByStaffNumber(ctx context.Context, staffNumber string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE staff_number = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, staffNumber).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists by staff number: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByStaffNumberExcludingID(ctx context.Context, staffNumber, userID string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE staff_number = $1 AND id <> $2)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, staffNumber, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists by staff number excluding id: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByGhanaCardNumber(ctx context.Context, ghanaCardNumber string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE ghana_card_number = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, ghanaCardNumber).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists by ghana card number: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByGhanaCardNumberExcludingID(ctx context.Context, ghanaCardNumber, userID string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE ghana_card_number = $1 AND id <> $2)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, ghanaCardNumber, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists by ghana card number excluding id: %w", err)
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
			SELECT id, display_name, COALESCE(phone_number, ''), role, created_at, updated_at
			FROM users
			ORDER BY created_at DESC
			OFFSET $1 LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, offset, limit)
	} else {
		query := `
			SELECT id, display_name, COALESCE(phone_number, ''), role, created_at, updated_at
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
			&user.DisplayName,
			&user.PhoneNumber,
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
// Activation metadata remains server-managed and is intentionally excluded.
func (r *UserRepository) Update(ctx context.Context, id string, input UpdateUserInput) (*models.User, error) {
	query := `
		UPDATE users
		SET display_name = $2,
			phone_number = $3,
			staff_number = $4,
			institution = $5,
			ghana_card_number = $6,
			password_hash = COALESCE($7, password_hash),
			role = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, display_name, COALESCE(phone_number, ''), COALESCE(staff_number, ''), COALESCE(institution, ''), COALESCE(ghana_card_number, ''), role, created_at, updated_at
	`

	var user models.User
	err := r.pool.QueryRow(
		ctx,
		query,
		id,
		input.DisplayName,
		input.PhoneNumber,
		input.StaffNumber,
		input.Institution,
		input.GhanaCardNumber,
		input.PasswordHash,
		input.Role,
	).Scan(
		&user.ID,
		&user.DisplayName,
		&user.PhoneNumber,
		&user.StaffNumber,
		&user.Institution,
		&user.GhanaCardNumber,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &user, nil
}
