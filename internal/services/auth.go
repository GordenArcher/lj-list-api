package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/jackc/pgx/v5"
)

const ActivationOTPExpiry = 10 * time.Minute

type authUserRepository interface {
	ExistsByPhoneNumber(ctx context.Context, phoneNumber string) (bool, error)
	ExistsByStaffNumber(ctx context.Context, staffNumber string) (bool, error)
	ExistsByGhanaCardNumber(ctx context.Context, ghanaCardNumber string) (bool, error)
	Create(ctx context.Context, input repositories.CreateUserInput) (*models.User, error)
	DeleteByID(ctx context.Context, id string) error
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateActivationOTP(ctx context.Context, userID string, otpHash string, otpExpiresAt time.Time) error
	Activate(ctx context.Context, userID string) (*models.User, error)
}

type authSMSSender interface {
	SendVerificationOTP(ctx context.Context, phoneNumber, displayName, otp string) error
}

type AuthService struct {
	userRepo    authUserRepository
	smsSender   authSMSSender
	cfg         config.Config
	now         func() time.Time
	generateOTP func() (string, error)
}

func NewAuthService(userRepo *repositories.UserRepository, smsSender authSMSSender, cfg config.Config) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		smsSender:   smsSender,
		cfg:         cfg,
		now:         time.Now,
		generateOTP: utils.GenerateOTP,
	}
}

type SignupInput struct {
	Password        string
	DisplayName     string
	PhoneNumber     string
	StaffNumber     string
	Institution     string
	GhanaCardNumber string
}

// Signup creates an inactive account, stores an activation OTP, and sends it
// to the registered phone number. The account becomes active only after OTP
// verification.
func (s *AuthService) Signup(ctx context.Context, input SignupInput) (*models.User, error) {
	normalized := normalizeSignupInput(input)

	exists, err := s.userRepo.ExistsByPhoneNumber(ctx, normalized.PhoneNumber)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to check phone number availability", err)
	}
	if exists {
		return nil, apperrors.New(apperrors.KindConflict, "Phone number already registered", map[string][]string{
			"phone_number": {"this phone number is already taken"},
		})
	}

	exists, err = s.userRepo.ExistsByStaffNumber(ctx, normalized.StaffNumber)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to check staff number availability", err)
	}
	if exists {
		return nil, apperrors.New(apperrors.KindConflict, "Staff number already registered", map[string][]string{
			"staff_number": {"this staff number is already taken"},
		})
	}

	exists, err = s.userRepo.ExistsByGhanaCardNumber(ctx, normalized.GhanaCardNumber)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to check Ghana Card number availability", err)
	}
	if exists {
		return nil, apperrors.New(apperrors.KindConflict, "Ghana Card number already registered", map[string][]string{
			"ghana_card_number": {"this Ghana Card number is already taken"},
		})
	}

	passwordHash, err := utils.HashPassword(normalized.Password)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to process password", err)
	}

	otp, otpHash, otpExpiresAt, err := s.generateActivationOTP()
	if err != nil {
		return nil, err
	}

	role := "customer"
	if utils.NormalizePhone(normalized.PhoneNumber) == utils.NormalizePhone(s.cfg.AdminPhoneNumber) {
		role = "admin"
	}

	user, err := s.userRepo.Create(ctx, repositories.CreateUserInput{
		PasswordHash:    passwordHash,
		DisplayName:     normalized.DisplayName,
		PhoneNumber:     normalized.PhoneNumber,
		StaffNumber:     normalized.StaffNumber,
		Institution:     normalized.Institution,
		GhanaCardNumber: normalized.GhanaCardNumber,
		IsActive:        false,
		OTPHash:         &otpHash,
		OTPExpiresAt:    &otpExpiresAt,
		Role:            role,
	})
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to create account", err)
	}

	if s.smsSender != nil {
		if err := s.smsSender.SendVerificationOTP(ctx, normalized.PhoneNumber, normalized.DisplayName, otp); err != nil {
			if cleanupErr := s.userRepo.DeleteByID(ctx, user.ID); cleanupErr != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to send activation OTP and cleanup pending account", cleanupErr)
			}
			return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to send activation OTP", err)
		}
	}

	return user, nil
}

// VerifyOTP activates the account and returns a signed JWT pair.
func (s *AuthService) VerifyOTP(ctx context.Context, phoneNumber, otp string) (*models.User, *utils.TokenPair, error) {
	user, err := s.userRepo.FindByPhoneNumber(ctx, utils.NormalizePhone(phoneNumber))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, invalidOTPError()
		}
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to verify activation OTP", err)
	}

	if user.IsActive {
		return nil, nil, apperrors.New(apperrors.KindConflict, "Account already activated", map[string][]string{
			"auth": {"account is already active"},
		})
	}

	if !s.isOTPValid(user, otp) {
		return nil, nil, invalidOTPError()
	}

	activatedUser, err := s.userRepo.Activate(ctx, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to activate account", err)
	}

	tokenPair, err := utils.GenerateTokenPair(activatedUser.ID, activatedUser.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to generate auth tokens", err)
	}

	return activatedUser, tokenPair, nil
}

// ResendOTP refreshes the activation code for an inactive account and sends it
// again to the registered phone number.
func (s *AuthService) ResendOTP(ctx context.Context, phoneNumber string) error {
	normalizedPhone := utils.NormalizePhone(phoneNumber)

	user, err := s.userRepo.FindByPhoneNumber(ctx, normalizedPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return apperrors.Wrap(apperrors.KindInternal, "Failed to resend activation OTP", err)
	}

	if user.IsActive {
		return apperrors.New(apperrors.KindConflict, "Account already activated", map[string][]string{
			"auth": {"account is already active"},
		})
	}

	otp, otpHash, otpExpiresAt, err := s.generateActivationOTP()
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdateActivationOTP(ctx, user.ID, otpHash, otpExpiresAt); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to refresh activation OTP", err)
	}

	if s.smsSender != nil {
		if err := s.smsSender.SendVerificationOTP(ctx, normalizedPhone, user.DisplayName, otp); err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "Failed to send activation OTP", err)
		}
	}

	return nil
}

// Login verifies credentials and returns a signed JWT. The account must be
// active; inactive users must complete OTP verification first.
func (s *AuthService) Login(ctx context.Context, phoneNumber, password string) (*models.User, *utils.TokenPair, error) {
	normalizedPhone := utils.NormalizePhone(phoneNumber)

	user, err := s.userRepo.FindByPhoneNumber(ctx, normalizedPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, invalidCredentialsError()
		}
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to authenticate user", err)
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, nil, invalidCredentialsError()
	}

	if !user.IsActive {
		return nil, nil, apperrors.New(apperrors.KindForbidden, "Account not activated", map[string][]string{
			"auth": {"verify the activation OTP sent to your phone number"},
		})
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "Failed to generate auth tokens", err)
	}

	return user, tokenPair, nil
}

// RefreshTokens validates a refresh token and generates a new token pair.
// It fetches the current user to ensure the role is up-to-date.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	userID, err := utils.ParseRefreshToken(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, apperrors.New(apperrors.KindUnauthorized, "Invalid or expired refresh token", map[string][]string{
			"auth": {"invalid refresh token"},
		})
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindUnauthorized, "Invalid or expired refresh token", map[string][]string{
				"auth": {"refresh token user no longer exists"},
			})
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to refresh tokens", err)
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to generate auth tokens", err)
	}

	return tokenPair, nil
}

func normalizeSignupInput(input SignupInput) SignupInput {
	return SignupInput{
		Password:        input.Password,
		DisplayName:     strings.TrimSpace(input.DisplayName),
		PhoneNumber:     utils.NormalizePhone(input.PhoneNumber),
		StaffNumber:     strings.TrimSpace(input.StaffNumber),
		Institution:     strings.TrimSpace(input.Institution),
		GhanaCardNumber: strings.TrimSpace(input.GhanaCardNumber),
	}
}

func (s *AuthService) generateActivationOTP() (string, string, time.Time, error) {
	otp, err := s.generateOTP()
	if err != nil {
		return "", "", time.Time{}, apperrors.Wrap(apperrors.KindInternal, "Failed to generate activation OTP", err)
	}

	otpHash, err := utils.HashPassword(otp)
	if err != nil {
		return "", "", time.Time{}, apperrors.Wrap(apperrors.KindInternal, "Failed to secure activation OTP", err)
	}

	expiresAt := s.now().UTC().Add(ActivationOTPExpiry)
	return otp, otpHash, expiresAt, nil
}

func (s *AuthService) isOTPValid(user *models.User, otp string) bool {
	if user == nil || user.OTPHash == nil || user.OTPExpiresAt == nil {
		return false
	}

	if s.now().UTC().After(user.OTPExpiresAt.UTC()) {
		return false
	}

	return utils.CheckPassword(strings.TrimSpace(otp), *user.OTPHash)
}

func invalidCredentialsError() error {
	return apperrors.New(apperrors.KindUnauthorized, "Invalid phone number or password", map[string][]string{
		"auth": {"phone number or password is incorrect"},
	})
}

func invalidOTPError() error {
	return apperrors.New(apperrors.KindValidation, "Invalid or expired OTP", map[string][]string{
		"otp": {"otp is invalid or expired"},
	})
}
