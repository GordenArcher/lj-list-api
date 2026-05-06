package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/jackc/pgx/v5"
)

type stubAuthRepo struct {
	existsByPhoneNumber       bool
	existsByStaffNumber       bool
	existsByGhanaCard         bool
	createInput               repositories.CreateUserInput
	createCalled              bool
	createdUser               *models.User
	createErr                 error
	deleteUserID              string
	deleteErr                 error
	findByPhoneNumber         string
	findByPhoneNumberUser     *models.User
	findByPhoneNumberErr      error
	updateActivationOTPUserID string
	updateActivationOTPHash   string
	updateActivationOTPAt     time.Time
	updateActivationOTPErr    error
	activateUserID            string
	activatedUser             *models.User
	activateErr               error
}

func (r *stubAuthRepo) ExistsByPhoneNumber(ctx context.Context, phoneNumber string) (bool, error) {
	return r.existsByPhoneNumber, nil
}

func (r *stubAuthRepo) ExistsByStaffNumber(ctx context.Context, staffNumber string) (bool, error) {
	return r.existsByStaffNumber, nil
}

func (r *stubAuthRepo) ExistsByGhanaCardNumber(ctx context.Context, ghanaCardNumber string) (bool, error) {
	return r.existsByGhanaCard, nil
}

func (r *stubAuthRepo) Create(ctx context.Context, input repositories.CreateUserInput) (*models.User, error) {
	r.createCalled = true
	r.createInput = input
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createdUser != nil {
		return r.createdUser, nil
	}

	return &models.User{
		ID:              "user-1",
		DisplayName:     input.DisplayName,
		PhoneNumber:     input.PhoneNumber,
		StaffNumber:     input.StaffNumber,
		Institution:     input.Institution,
		GhanaCardNumber: input.GhanaCardNumber,
		IsActive:        input.IsActive,
		OTPHash:         cloneStringPtr(input.OTPHash),
		OTPExpiresAt:    cloneTimePtr(input.OTPExpiresAt),
		Role:            input.Role,
		CreatedAt:       time.Unix(1, 0).UTC(),
		UpdatedAt:       time.Unix(2, 0).UTC(),
	}, nil
}

func (r *stubAuthRepo) DeleteByID(ctx context.Context, id string) error {
	r.deleteUserID = id
	return r.deleteErr
}

func (r *stubAuthRepo) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	r.findByPhoneNumber = phoneNumber
	if r.findByPhoneNumberErr != nil {
		return nil, r.findByPhoneNumberErr
	}
	if r.findByPhoneNumberUser != nil {
		return r.findByPhoneNumberUser, nil
	}
	return nil, pgx.ErrNoRows
}

func (r *stubAuthRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	return nil, pgx.ErrNoRows
}

func (r *stubAuthRepo) UpdateActivationOTP(ctx context.Context, userID string, otpHash string, otpExpiresAt time.Time) error {
	r.updateActivationOTPUserID = userID
	r.updateActivationOTPHash = otpHash
	r.updateActivationOTPAt = otpExpiresAt
	return r.updateActivationOTPErr
}

func (r *stubAuthRepo) Activate(ctx context.Context, userID string) (*models.User, error) {
	r.activateUserID = userID
	if r.activateErr != nil {
		return nil, r.activateErr
	}
	if r.activatedUser != nil {
		return r.activatedUser, nil
	}
	return nil, pgx.ErrNoRows
}

type stubAuthSMSSender struct {
	phoneNumber string
	displayName string
	otp         string
	err         error
	calls       int
}

func (s *stubAuthSMSSender) SendVerificationOTP(ctx context.Context, phoneNumber, displayName, otp string) error {
	s.calls++
	s.phoneNumber = phoneNumber
	s.displayName = displayName
	s.otp = otp
	return s.err
}

func TestAuthServiceSignupRejectsDuplicateRegistrationFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repo    *stubAuthRepo
		field   string
		message string
	}{
		{
			name:    "duplicate phone number",
			repo:    &stubAuthRepo{existsByPhoneNumber: true},
			field:   "phone_number",
			message: "Phone number already registered",
		},
		{
			name:    "duplicate staff number",
			repo:    &stubAuthRepo{existsByStaffNumber: true},
			field:   "staff_number",
			message: "Staff number already registered",
		},
		{
			name:    "duplicate ghana card number",
			repo:    &stubAuthRepo{existsByGhanaCard: true},
			field:   "ghana_card_number",
			message: "Ghana Card number already registered",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &AuthService{
				userRepo: tc.repo,
				cfg:      config.Config{JWTSecret: "secret"},
			}

			_, err := service.Signup(context.Background(), SignupInput{
				PhoneNumber:     "+233240000000",
				StaffNumber:     "GES-2024-0018",
				Institution:     "Ghana Education Service",
				GhanaCardNumber: "GHA-123456789-0",
				Password:        "password123",
				DisplayName:     "Kwame Mensah",
			})
			if err == nil {
				t.Fatal("expected conflict error")
			}

			var appErr *apperrors.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected apperrors.Error, got %T", err)
			}
			if appErr.Kind != apperrors.KindConflict {
				t.Fatalf("expected conflict kind, got %s", appErr.Kind)
			}
			if appErr.Message != tc.message {
				t.Fatalf("expected message %q, got %q", tc.message, appErr.Message)
			}
			if _, ok := appErr.Details[tc.field]; !ok {
				t.Fatalf("expected error details for field %q, got %#v", tc.field, appErr.Details)
			}
			if tc.repo.createCalled {
				t.Fatal("expected create to be skipped when duplicate exists")
			}
		})
	}
}

func TestAuthServiceSignupCreatesInactiveUserAndSendsOTP(t *testing.T) {
	t.Parallel()

	repo := &stubAuthRepo{
		createdUser: &models.User{
			ID:              "user-1",
			DisplayName:     "Kwame Mensah",
			PhoneNumber:     "+233240000000",
			StaffNumber:     "GES-2024-0018",
			Institution:     "Ghana Education Service",
			GhanaCardNumber: "GHA-123456789-0",
			IsActive:        false,
			Role:            "customer",
		},
	}
	smsSender := &stubAuthSMSSender{}
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	service := &AuthService{
		userRepo:    repo,
		smsSender:   smsSender,
		cfg:         config.Config{JWTSecret: "secret"},
		now:         func() time.Time { return now },
		generateOTP: func() (string, error) { return "123456", nil },
	}

	user, err := service.Signup(context.Background(), SignupInput{
		PhoneNumber:     "+233 24-000-0000",
		StaffNumber:     " GES-2024-0018 ",
		Institution:     " Ghana Education Service ",
		GhanaCardNumber: " GHA-123456789-0 ",
		Password:        "password123",
		DisplayName:     " Kwame Mensah ",
	})
	if err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}
	if !repo.createCalled {
		t.Fatal("expected create to be called")
	}
	if repo.createInput.PhoneNumber != "+233240000000" {
		t.Fatalf("expected normalized phone number, got %q", repo.createInput.PhoneNumber)
	}
	if repo.createInput.IsActive {
		t.Fatal("expected created user to be inactive pending OTP verification")
	}
	if repo.createInput.OTPHash == nil || !utils.CheckPassword("123456", *repo.createInput.OTPHash) {
		t.Fatal("expected OTP hash to match generated OTP")
	}
	if repo.createInput.OTPExpiresAt == nil || !repo.createInput.OTPExpiresAt.Equal(now.Add(ActivationOTPExpiry)) {
		t.Fatalf("unexpected OTP expiry: %#v", repo.createInput.OTPExpiresAt)
	}
	if smsSender.calls != 1 {
		t.Fatalf("expected one OTP SMS, got %d", smsSender.calls)
	}
	if smsSender.phoneNumber != "+233240000000" || smsSender.otp != "123456" {
		t.Fatalf("unexpected SMS payload: phone=%q otp=%q", smsSender.phoneNumber, smsSender.otp)
	}
	if user == nil || user.ID != "user-1" {
		t.Fatalf("unexpected signup result: %#v", user)
	}
}

func TestAuthServiceVerifyOTPActivatesUser(t *testing.T) {
	t.Parallel()

	otpHash, err := utils.HashPassword("123456")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	repo := &stubAuthRepo{
		findByPhoneNumberUser: &models.User{
			ID:           "user-1",
			PhoneNumber:  "+233240000000",
			DisplayName:  "Kwame Mensah",
			OTPHash:      &otpHash,
			Role:         "customer",
			IsActive:     false,
			OTPExpiresAt: cloneTimePtrValue(time.Date(2026, 4, 29, 12, 10, 0, 0, time.UTC)),
		},
		activatedUser: &models.User{
			ID:          "user-1",
			PhoneNumber: "+233240000000",
			DisplayName: "Kwame Mensah",
			Role:        "customer",
			IsActive:    true,
		},
	}
	service := &AuthService{
		userRepo: repo,
		cfg:      config.Config{JWTSecret: "secret"},
		now:      func() time.Time { return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC) },
	}

	user, tokenPair, err := service.VerifyOTP(context.Background(), "+233 24-000-0000", "123456")
	if err != nil {
		t.Fatalf("VerifyOTP returned error: %v", err)
	}
	if repo.activateUserID != "user-1" {
		t.Fatalf("expected activate to be called for user-1, got %q", repo.activateUserID)
	}
	if user == nil || !user.IsActive {
		t.Fatalf("expected activated user, got %#v", user)
	}
	if tokenPair == nil {
		t.Fatal("expected token pair after OTP verification")
	}
}

func TestAuthServiceSignupDeletesPendingAccountIfOTPSendFails(t *testing.T) {
	t.Parallel()

	repo := &stubAuthRepo{
		createdUser: &models.User{
			ID:              "user-1",
			DisplayName:     "Kwame Mensah",
			PhoneNumber:     "+233240000000",
			StaffNumber:     "GES-2024-0018",
			Institution:     "Ghana Education Service",
			GhanaCardNumber: "GHA-123456789-0",
			IsActive:        false,
			Role:            "customer",
		},
	}
	smsSender := &stubAuthSMSSender{err: errors.New("sms down")}

	service := &AuthService{
		userRepo:    repo,
		smsSender:   smsSender,
		cfg:         config.Config{JWTSecret: "secret"},
		now:         func() time.Time { return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC) },
		generateOTP: func() (string, error) { return "123456", nil },
	}

	_, err := service.Signup(context.Background(), SignupInput{
		PhoneNumber:     "+233240000000",
		StaffNumber:     "GES-2024-0018",
		Institution:     "Ghana Education Service",
		GhanaCardNumber: "GHA-123456789-0",
		Password:        "password123",
		DisplayName:     "Kwame Mensah",
	})
	if err == nil {
		t.Fatal("expected signup error when OTP send fails")
	}
	if repo.deleteUserID != "user-1" {
		t.Fatalf("expected pending account cleanup for user-1, got %q", repo.deleteUserID)
	}
}

func TestAuthServiceResendOTPRefreshesInactiveUserCode(t *testing.T) {
	t.Parallel()

	repo := &stubAuthRepo{
		findByPhoneNumberUser: &models.User{
			ID:          "user-1",
			PhoneNumber: "+233240000000",
			DisplayName: "Kwame Mensah",
			IsActive:    false,
		},
	}
	smsSender := &stubAuthSMSSender{}
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	service := &AuthService{
		userRepo:    repo,
		smsSender:   smsSender,
		cfg:         config.Config{JWTSecret: "secret"},
		now:         func() time.Time { return now },
		generateOTP: func() (string, error) { return "654321", nil },
	}

	if err := service.ResendOTP(context.Background(), "+233 24-000-0000"); err != nil {
		t.Fatalf("ResendOTP returned error: %v", err)
	}
	if repo.updateActivationOTPUserID != "user-1" {
		t.Fatalf("expected OTP refresh for user-1, got %q", repo.updateActivationOTPUserID)
	}
	if !utils.CheckPassword("654321", repo.updateActivationOTPHash) {
		t.Fatal("expected refreshed OTP hash to match generated OTP")
	}
	if !repo.updateActivationOTPAt.Equal(now.Add(ActivationOTPExpiry)) {
		t.Fatalf("unexpected refreshed OTP expiry: %v", repo.updateActivationOTPAt)
	}
	if smsSender.calls != 1 || smsSender.otp != "654321" {
		t.Fatalf("unexpected OTP resend payload: calls=%d otp=%q", smsSender.calls, smsSender.otp)
	}
}

func TestAuthServiceLoginRejectsInactiveAccounts(t *testing.T) {
	t.Parallel()

	hash, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	repo := &stubAuthRepo{
		findByPhoneNumberUser: &models.User{
			ID:           "user-1",
			PasswordHash: hash,
			PhoneNumber:  "+233240000000",
			Role:         "customer",
			IsActive:     false,
		},
	}

	service := &AuthService{
		userRepo: repo,
		cfg:      config.Config{JWTSecret: "secret"},
	}

	_, _, err = service.Login(context.Background(), "+233 24-000-0000", "password123")
	if err == nil {
		t.Fatal("expected inactive account error")
	}

	var appErr *apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperrors.Error, got %T", err)
	}
	if appErr.Kind != apperrors.KindForbidden {
		t.Fatalf("expected forbidden error, got %s", appErr.Kind)
	}
}

func TestAuthServiceLoginAdminRejectsCustomerAccounts(t *testing.T) {
	t.Parallel()

	hash, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	repo := &stubAuthRepo{
		findByPhoneNumberUser: &models.User{
			ID:           "user-1",
			PasswordHash: hash,
			PhoneNumber:  "+233240000000",
			Role:         "customer",
			IsActive:     true,
		},
	}

	service := &AuthService{
		userRepo: repo,
		cfg:      config.Config{JWTSecret: "secret"},
	}

	_, _, err = service.LoginAdmin(context.Background(), "+233 24-000-0000", "password123")
	if err == nil {
		t.Fatal("expected admin access error")
	}

	var appErr *apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperrors.Error, got %T", err)
	}
	if appErr.Kind != apperrors.KindForbidden {
		t.Fatalf("expected forbidden error, got %s", appErr.Kind)
	}
}

func TestAuthServiceLoginAdminAllowsAdminAccounts(t *testing.T) {
	t.Parallel()

	hash, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	repo := &stubAuthRepo{
		findByPhoneNumberUser: &models.User{
			ID:           "admin-1",
			PasswordHash: hash,
			PhoneNumber:  "+233240000000",
			Role:         "admin",
			IsActive:     true,
		},
	}

	service := &AuthService{
		userRepo: repo,
		cfg:      config.Config{JWTSecret: "secret"},
	}

	user, tokenPair, err := service.LoginAdmin(context.Background(), "+233 24-000-0000", "password123")
	if err != nil {
		t.Fatalf("LoginAdmin returned error: %v", err)
	}
	if user == nil || user.Role != "admin" {
		t.Fatalf("expected admin user, got %#v", user)
	}
	if tokenPair == nil || tokenPair.AccessToken == "" || tokenPair.RefreshToken == "" {
		t.Fatalf("expected token pair, got %#v", tokenPair)
	}
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePtrValue(value time.Time) *time.Time {
	cloned := value
	return &cloned
}
