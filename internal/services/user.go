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

type userRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error)
	CountAll(ctx context.Context, role string) (int, error)
	Update(ctx context.Context, id, displayName string, phone *string, role string) (*models.User, error)
}

type UserService struct {
	userRepo userRepository
	cfg      config.Config
}

type UpdateProfileInput struct {
	DisplayName *string
	Phone       *string
}

type AdminUpdateUserInput struct {
	DisplayName *string
	Phone       *string
	Role        *string
}

func NewUserService(userRepo *repositories.UserRepository, cfg config.Config) *UserService {
	return &UserService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve profile", err)
	}
	return user, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*models.User, error) {
	current, err := s.userRepo.FindByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve profile", err)
	}

	displayName, phone, changed, err := normalizeUserEditInput(current.DisplayName, current.Phone, input.DisplayName, input.Phone, nil)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, noUserFieldsProvidedError()
	}

	user, err := s.userRepo.Update(ctx, current.ID, displayName, phone, current.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to update profile", err)
	}

	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context, role string, offset, limit int) ([]models.User, error) {
	normalizedRole, err := normalizeRoleFilter(role)
	if err != nil {
		return nil, err
	}

	users, err := s.userRepo.FindAll(ctx, normalizedRole, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve users", err)
	}
	return users, nil
}

func (s *UserService) CountUsers(ctx context.Context, role string) (int, error) {
	normalizedRole, err := normalizeRoleFilter(role)
	if err != nil {
		return 0, err
	}

	count, err := s.userRepo.CountAll(ctx, normalizedRole)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve user count", err)
	}
	return count, nil
}

func (s *UserService) AdminUpdateUser(ctx context.Context, userID string, input AdminUpdateUserInput) (*models.User, error) {
	current, err := s.userRepo.FindByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve user", err)
	}

	displayName, phone, role, changed, err := normalizeAdminUserEditInput(current, input, s.cfg.AdminEmail)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, noUserFieldsProvidedError()
	}

	user, err := s.userRepo.Update(ctx, current.ID, displayName, phone, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to update user", err)
	}

	return user, nil
}

func normalizeRoleFilter(role string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(role))
	if trimmed == "" {
		return "", nil
	}
	if trimmed != "customer" && trimmed != "admin" {
		return "", apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"role": {"must be customer or admin"},
		})
	}
	return trimmed, nil
}

func normalizeAdminUserEditInput(current *models.User, input AdminUpdateUserInput, configuredAdminEmail string) (string, *string, string, bool, error) {
	displayName, phone, changed, err := normalizeUserEditInput(current.DisplayName, current.Phone, input.DisplayName, input.Phone, input.Role)
	if err != nil {
		return "", nil, "", false, err
	}

	role := current.Role
	if input.Role != nil {
		role = strings.ToLower(strings.TrimSpace(*input.Role))
		changed = true
	}

	if strings.EqualFold(strings.TrimSpace(current.Email), strings.TrimSpace(configuredAdminEmail)) && role != "admin" {
		return "", nil, "", false, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"role": {"configured admin account must keep admin role"},
		})
	}

	return displayName, phone, role, changed, nil
}

func normalizeUserEditInput(currentDisplayName string, currentPhone *string, displayNameInput, phoneInput, roleInput *string) (string, *string, bool, error) {
	displayName := currentDisplayName
	phone := cloneStringPtr(currentPhone)
	changed := false
	errs := make(map[string][]string)

	if displayNameInput != nil {
		trimmed := strings.TrimSpace(*displayNameInput)
		if !utils.ValidateDisplayName(trimmed) {
			errs["display_name"] = []string{"must be between 2 and 100 characters"}
		} else {
			displayName = trimmed
			changed = true
		}
	}

	if phoneInput != nil {
		normalizedPhone := utils.NormalizePhone(*phoneInput)
		if normalizedPhone == "" {
			phone = nil
			changed = true
		} else if !utils.ValidatePhone(normalizedPhone) {
			errs["phone"] = []string{"must be a valid phone number"}
		} else {
			phone = &normalizedPhone
			changed = true
		}
	}

	if roleInput != nil {
		role := strings.ToLower(strings.TrimSpace(*roleInput))
		if role != "customer" && role != "admin" {
			errs["role"] = []string{"must be customer or admin"}
		}
	}

	if len(errs) > 0 {
		return "", nil, false, apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	return displayName, phone, changed, nil
}

func noUserFieldsProvidedError() error {
	return apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
		"user": {"at least one field must be provided"},
	})
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
