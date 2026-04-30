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
	Update(ctx context.Context, id string, input repositories.UpdateUserInput) (*models.User, error)
	ExistsByPhoneNumberExcludingID(ctx context.Context, phoneNumber, userID string) (bool, error)
	ExistsByStaffNumberExcludingID(ctx context.Context, staffNumber, userID string) (bool, error)
	ExistsByGhanaCardNumberExcludingID(ctx context.Context, ghanaCardNumber, userID string) (bool, error)
}

type UserService struct {
	userRepo userRepository
	cfg      config.Config
}

type UpdateProfileInput struct {
	DisplayName     *string
	PhoneNumber     *string
	StaffNumber     *string
	Institution     *string
	GhanaCardNumber *string
	Password        *string
}

type AdminUpdateUserInput struct {
	DisplayName *string
	PhoneNumber *string
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

	updateInput, changed, err := s.normalizeProfileUpdateInput(ctx, current, input)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, noUserFieldsProvidedError()
	}

	user, err := s.userRepo.Update(ctx, current.ID, updateInput)
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

	updateInput, changed, err := s.normalizeAdminUpdateInput(ctx, current, input)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, noUserFieldsProvidedError()
	}

	user, err := s.userRepo.Update(ctx, current.ID, updateInput)
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

func (s *UserService) normalizeProfileUpdateInput(ctx context.Context, current *models.User, input UpdateProfileInput) (repositories.UpdateUserInput, bool, error) {
	updateInput := repositories.UpdateUserInput{
		DisplayName:     current.DisplayName,
		PhoneNumber:     current.PhoneNumber,
		StaffNumber:     current.StaffNumber,
		Institution:     current.Institution,
		GhanaCardNumber: current.GhanaCardNumber,
		Role:            current.Role,
	}

	errs := make(map[string][]string)
	changed := false
	phoneNumberProvided := false
	staffNumberProvided := false
	ghanaCardNumberProvided := false

	if input.DisplayName != nil {
		trimmed := strings.TrimSpace(*input.DisplayName)
		if !utils.ValidateDisplayName(trimmed) {
			errs["display_name"] = []string{"must be between 2 and 100 characters"}
		} else {
			updateInput.DisplayName = trimmed
			changed = true
		}
	}

	if input.PhoneNumber != nil {
		phoneNumberProvided = true
		normalizedPhoneNumber := utils.NormalizePhone(*input.PhoneNumber)
		if !utils.ValidatePhone(normalizedPhoneNumber) {
			errs["phone_number"] = []string{"must be a valid phone number"}
		} else {
			updateInput.PhoneNumber = normalizedPhoneNumber
			changed = true
		}
	}

	if input.StaffNumber != nil {
		staffNumberProvided = true
		trimmed := strings.TrimSpace(*input.StaffNumber)
		if !utils.ValidateRequired(trimmed) {
			errs["staff_number"] = []string{"required"}
		} else {
			updateInput.StaffNumber = trimmed
			changed = true
		}
	}

	if input.Institution != nil {
		trimmed := strings.TrimSpace(*input.Institution)
		if !utils.ValidateRequired(trimmed) {
			errs["institution"] = []string{"required"}
		} else {
			updateInput.Institution = trimmed
			changed = true
		}
	}

	if input.GhanaCardNumber != nil {
		ghanaCardNumberProvided = true
		trimmed := strings.TrimSpace(*input.GhanaCardNumber)
		if !utils.ValidateRequired(trimmed) {
			errs["ghana_card_number"] = []string{"required"}
		} else {
			updateInput.GhanaCardNumber = trimmed
			changed = true
		}
	}

	if input.Password != nil {
		if !utils.ValidatePassword(*input.Password) {
			errs["password"] = []string{"must be at least 8 characters"}
		} else {
			passwordHash, err := utils.HashPassword(*input.Password)
			if err != nil {
				return repositories.UpdateUserInput{}, false, apperrors.Wrap(apperrors.KindInternal, "Failed to process password", err)
			}
			updateInput.PasswordHash = &passwordHash
			changed = true
		}
	}

	if len(errs) > 0 {
		return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	if !changed {
		return updateInput, false, nil
	}

	if err := s.validateEditableFieldConflicts(ctx, current, updateInput, phoneNumberProvided, staffNumberProvided, ghanaCardNumberProvided); err != nil {
		return repositories.UpdateUserInput{}, false, err
	}

	if phoneNumberProvided &&
		isReservedAdminPhone(updateInput.PhoneNumber, s.cfg.AdminPhoneNumber) &&
		!isConfiguredAdminAccount(current.PhoneNumber, s.cfg.AdminPhoneNumber) {
		return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"phone_number": {"configured admin phone number is reserved for the admin account"},
		})
	}

	if isConfiguredAdminAccount(current.PhoneNumber, s.cfg.AdminPhoneNumber) &&
		utils.NormalizePhone(updateInput.PhoneNumber) != utils.NormalizePhone(s.cfg.AdminPhoneNumber) {
		return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"phone_number": {"configured admin account must keep the configured admin phone number"},
		})
	}

	return updateInput, true, nil
}

func (s *UserService) normalizeAdminUpdateInput(ctx context.Context, current *models.User, input AdminUpdateUserInput) (repositories.UpdateUserInput, bool, error) {
	updateInput := repositories.UpdateUserInput{
		DisplayName:     current.DisplayName,
		PhoneNumber:     current.PhoneNumber,
		StaffNumber:     current.StaffNumber,
		Institution:     current.Institution,
		GhanaCardNumber: current.GhanaCardNumber,
		Role:            current.Role,
	}

	errs := make(map[string][]string)
	changed := false

	if input.DisplayName != nil {
		trimmed := strings.TrimSpace(*input.DisplayName)
		if !utils.ValidateDisplayName(trimmed) {
			errs["display_name"] = []string{"must be between 2 and 100 characters"}
		} else {
			updateInput.DisplayName = trimmed
			changed = true
		}
	}

	if input.PhoneNumber != nil {
		normalizedPhoneNumber := utils.NormalizePhone(*input.PhoneNumber)
		if !utils.ValidatePhone(normalizedPhoneNumber) {
			errs["phone_number"] = []string{"must be a valid phone number"}
		} else {
			updateInput.PhoneNumber = normalizedPhoneNumber
			changed = true
		}
	}

	if input.Role != nil {
		role := strings.ToLower(strings.TrimSpace(*input.Role))
		if role != "customer" && role != "admin" {
			errs["role"] = []string{"must be customer or admin"}
		} else {
			updateInput.Role = role
			changed = true
		}
	}

	if len(errs) > 0 {
		return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	if !changed {
		return updateInput, false, nil
	}

	if input.PhoneNumber != nil {
		exists, err := s.userRepo.ExistsByPhoneNumberExcludingID(ctx, updateInput.PhoneNumber, current.ID)
		if err != nil {
			return repositories.UpdateUserInput{}, false, apperrors.Wrap(apperrors.KindInternal, "Failed to check phone number availability", err)
		}
		if exists {
			return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindConflict, "Phone number already registered", map[string][]string{
				"phone_number": {"this phone number is already taken"},
			})
		}
	}

	if input.PhoneNumber != nil &&
		isReservedAdminPhone(updateInput.PhoneNumber, s.cfg.AdminPhoneNumber) &&
		!isConfiguredAdminAccount(current.PhoneNumber, s.cfg.AdminPhoneNumber) {
		return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"phone_number": {"configured admin phone number is reserved for the admin account"},
		})
	}

	if isConfiguredAdminAccount(current.PhoneNumber, s.cfg.AdminPhoneNumber) {
		if updateInput.Role != "admin" {
			return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
				"role": {"configured admin account must keep admin role"},
			})
		}
		if utils.NormalizePhone(updateInput.PhoneNumber) != utils.NormalizePhone(s.cfg.AdminPhoneNumber) {
			return repositories.UpdateUserInput{}, false, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
				"phone_number": {"configured admin account must keep the configured admin phone number"},
			})
		}
	}

	return updateInput, true, nil
}

func (s *UserService) validateEditableFieldConflicts(ctx context.Context, current *models.User, updateInput repositories.UpdateUserInput, checkPhoneNumber, checkStaffNumber, checkGhanaCardNumber bool) error {
	if checkPhoneNumber {
		exists, err := s.userRepo.ExistsByPhoneNumberExcludingID(ctx, updateInput.PhoneNumber, current.ID)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "Failed to check phone number availability", err)
		}
		if exists {
			return apperrors.New(apperrors.KindConflict, "Phone number already registered", map[string][]string{
				"phone_number": {"this phone number is already taken"},
			})
		}
	}

	if checkStaffNumber {
		exists, err := s.userRepo.ExistsByStaffNumberExcludingID(ctx, updateInput.StaffNumber, current.ID)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "Failed to check staff number availability", err)
		}
		if exists {
			return apperrors.New(apperrors.KindConflict, "Staff number already registered", map[string][]string{
				"staff_number": {"this staff number is already taken"},
			})
		}
	}

	if checkGhanaCardNumber {
		exists, err := s.userRepo.ExistsByGhanaCardNumberExcludingID(ctx, updateInput.GhanaCardNumber, current.ID)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "Failed to check Ghana Card number availability", err)
		}
		if exists {
			return apperrors.New(apperrors.KindConflict, "Ghana Card number already registered", map[string][]string{
				"ghana_card_number": {"this Ghana Card number is already taken"},
			})
		}
	}

	return nil
}

func isConfiguredAdminAccount(phoneNumber, configuredAdminPhoneNumber string) bool {
	return utils.NormalizePhone(phoneNumber) != "" &&
		utils.NormalizePhone(phoneNumber) == utils.NormalizePhone(configuredAdminPhoneNumber)
}

func isReservedAdminPhone(phoneNumber, configuredAdminPhoneNumber string) bool {
	return utils.NormalizePhone(configuredAdminPhoneNumber) != "" &&
		utils.NormalizePhone(phoneNumber) == utils.NormalizePhone(configuredAdminPhoneNumber)
}

func noUserFieldsProvidedError() error {
	return apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
		"user": {"at least one field must be provided"},
	})
}
