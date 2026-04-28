package services

import (
	"context"
	"errors"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/jackc/pgx/v5"
)

type AuthService struct {
	userRepo *repositories.UserRepository
	cfg      config.Config
}

func NewAuthService(userRepo *repositories.UserRepository, cfg config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

// Signup creates a new user account. If the email matches the configured
// ADMIN_EMAIL, the account gets the "admin" role. Otherwise, "customer".
// The password is hashed with bcrypt before storage, the raw password is
// never persisted. Returns the full user record and a signed JWT.
func (s *AuthService) Signup(ctx context.Context, email, password, displayName string) (*models.User, *utils.TokenPair, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to check email availability", err)
	}
	if exists {
		return nil, nil, apperrors.New(apperrors.KindConflict, "Email already registered", map[string][]string{
			"email": {"this email is already taken"},
		})
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to process password", err)
	}

	role := "customer"
	if strings.EqualFold(email, s.cfg.AdminEmail) {
		role = "admin"
	}

	user, err := s.userRepo.Create(ctx, email, hash, displayName, role)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to create account", err)
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to generate auth tokens", err)
	}

	return user, tokenPair, nil
}

// Login verifies credentials and returns a signed JWT. If the email doesn't
// exist or the password doesn't match, the error message is intentionally
// vague, "invalid email or password", to avoid leaking which part failed.
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, *utils.TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, apperrors.New(apperrors.KindUnauthorized, "Invalid email or password", map[string][]string{
				"auth": {"email or password is incorrect"},
			})
		}
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to authenticate user", err)
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, nil, apperrors.New(apperrors.KindUnauthorized, "Invalid email or password", map[string][]string{
			"auth": {"email or password is incorrect"},
		})
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to generate auth tokens", err)
	}

	return user, tokenPair, nil
}

// RefreshTokens validates a refresh token and generates a new token pair.
// It fetches the current user to ensure the role is up-to-date (e.g., user
// might have been promoted to admin). If the refresh token is invalid or the
// user no longer exists, an error is returned.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	// Parse the refresh token to extract the user ID
	userID, err := utils.ParseRefreshToken(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, apperrors.New(apperrors.KindUnauthorized, "Invalid or expired refresh token", map[string][]string{
			"auth": {"invalid refresh token"},
		})
	}

	// Fetch the user to get their current role
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindUnauthorized, "Invalid or expired refresh token", map[string][]string{
				"auth": {"refresh token user no longer exists"},
			})
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to refresh tokens", err)
	}

	// Generate a new token pair with the current user information
	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to generate auth tokens", err)
	}

	return tokenPair, nil
}
