package services

import (
	"context"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
)

type stubUserServiceRepo struct {
	currentUser                  *models.User
	updatedUser                  *models.User
	updateInput                  repositories.UpdateUserInput
	updateRole                   string
	existsByPhoneNumber          bool
	existsByStaffNumber          bool
	existsByGhanaCardNumber      bool
	existsByPhoneNumberValue     string
	existsByStaffNumberValue     string
	existsByGhanaCardNumberValue string
}

func (r *stubUserServiceRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	if r.currentUser == nil {
		return nil, nil
	}
	userCopy := *r.currentUser
	return &userCopy, nil
}

func (r *stubUserServiceRepo) FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error) {
	return nil, nil
}

func (r *stubUserServiceRepo) CountAll(ctx context.Context, role string) (int, error) {
	return 0, nil
}

func (r *stubUserServiceRepo) Update(ctx context.Context, id string, input repositories.UpdateUserInput) (*models.User, error) {
	r.updateInput = input
	r.updateRole = input.Role

	if r.updatedUser != nil {
		userCopy := *r.updatedUser
		return &userCopy, nil
	}

	return &models.User{
		ID:              id,
		DisplayName:     input.DisplayName,
		PhoneNumber:     input.PhoneNumber,
		StaffNumber:     input.StaffNumber,
		Institution:     input.Institution,
		GhanaCardNumber: input.GhanaCardNumber,
		Role:            input.Role,
		CreatedAt:       time.Unix(1, 0).UTC(),
		UpdatedAt:       time.Unix(2, 0).UTC(),
	}, nil
}

func (r *stubUserServiceRepo) ExistsByPhoneNumberExcludingID(ctx context.Context, phoneNumber, userID string) (bool, error) {
	r.existsByPhoneNumberValue = phoneNumber
	return r.existsByPhoneNumber, nil
}

func (r *stubUserServiceRepo) ExistsByStaffNumberExcludingID(ctx context.Context, staffNumber, userID string) (bool, error) {
	r.existsByStaffNumberValue = staffNumber
	return r.existsByStaffNumber, nil
}

func (r *stubUserServiceRepo) ExistsByGhanaCardNumberExcludingID(ctx context.Context, ghanaCardNumber, userID string) (bool, error) {
	r.existsByGhanaCardNumberValue = ghanaCardNumber
	return r.existsByGhanaCardNumber, nil
}

func TestUserServiceUpdateProfileNormalizesFields(t *testing.T) {
	t.Parallel()

	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:              "user-1",
			PhoneNumber:     "+233240000111",
			DisplayName:     "Kwame",
			StaffNumber:     "GES-OLD-1",
			Institution:     "GES",
			GhanaCardNumber: "GHA-OLD-1",
			Role:            "customer",
		},
	}

	service := &UserService{userRepo: repo}
	displayName := " Kwame Mensah "
	phoneNumber := "+233 24-000-0000"
	staffNumber := " GES-2024-0018 "
	institution := " Ghana Education Service "
	ghanaCardNumber := " GHA-123456789-0 "
	password := "password123"

	user, err := service.UpdateProfile(context.Background(), "user-1", UpdateProfileInput{
		DisplayName:     &displayName,
		PhoneNumber:     &phoneNumber,
		StaffNumber:     &staffNumber,
		Institution:     &institution,
		GhanaCardNumber: &ghanaCardNumber,
		Password:        &password,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}

	if repo.updateInput.DisplayName != "Kwame Mensah" {
		t.Fatalf("unexpected display name update: %q", repo.updateInput.DisplayName)
	}
	if repo.updateInput.PhoneNumber != "+233240000000" {
		t.Fatalf("unexpected phone number update: %q", repo.updateInput.PhoneNumber)
	}
	if repo.updateInput.StaffNumber != "GES-2024-0018" {
		t.Fatalf("unexpected staff number update: %q", repo.updateInput.StaffNumber)
	}
	if repo.updateInput.Institution != "Ghana Education Service" {
		t.Fatalf("unexpected institution update: %q", repo.updateInput.Institution)
	}
	if repo.updateInput.GhanaCardNumber != "GHA-123456789-0" {
		t.Fatalf("unexpected Ghana Card update: %q", repo.updateInput.GhanaCardNumber)
	}
	if repo.updateInput.PasswordHash == nil {
		t.Fatal("expected password hash to be set")
	}
	if repo.existsByPhoneNumberValue != "+233240000000" {
		t.Fatalf("expected duplicate check for phone number, got %q", repo.existsByPhoneNumberValue)
	}
	if repo.existsByStaffNumberValue != "GES-2024-0018" {
		t.Fatalf("expected duplicate check for staff number, got %q", repo.existsByStaffNumberValue)
	}
	if repo.existsByGhanaCardNumberValue != "GHA-123456789-0" {
		t.Fatalf("expected duplicate check for Ghana Card number, got %q", repo.existsByGhanaCardNumberValue)
	}
	if user == nil || user.DisplayName != "Kwame Mensah" {
		t.Fatalf("unexpected user response: %#v", user)
	}
}

func TestUserServiceUpdateProfileRejectsDuplicatePhoneNumber(t *testing.T) {
	t.Parallel()

	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:              "user-1",
			PhoneNumber:     "+233240000111",
			DisplayName:     "Kwame",
			StaffNumber:     "GES-OLD-1",
			Institution:     "GES",
			GhanaCardNumber: "GHA-OLD-1",
			Role:            "customer",
		},
		existsByPhoneNumber: true,
	}

	service := &UserService{userRepo: repo}
	phoneNumber := "+233 24-000-0000"

	_, err := service.UpdateProfile(context.Background(), "user-1", UpdateProfileInput{
		PhoneNumber: &phoneNumber,
	})
	if err == nil {
		t.Fatal("expected duplicate phone validation error")
	}
}

func TestUserServiceAdminUpdateUserProtectsConfiguredAdminRole(t *testing.T) {
	t.Parallel()

	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:          "admin-1",
			PhoneNumber: "+233500000001",
			DisplayName: "Admin",
			Role:        "admin",
		},
	}

	service := &UserService{
		userRepo: repo,
		cfg: config.Config{
			AdminPhoneNumber: "+233500000001",
		},
	}

	role := "customer"
	_, err := service.AdminUpdateUser(context.Background(), "admin-1", AdminUpdateUserInput{
		Role: &role,
	})
	if err == nil {
		t.Fatal("expected validation error when demoting configured admin")
	}
	if repo.updateRole != "" {
		t.Fatalf("expected update to be blocked, got role update %q", repo.updateRole)
	}
}

func TestUserServiceUpdateProfileRejectsConfiguredAdminPhoneForNonAdminAccount(t *testing.T) {
	t.Parallel()

	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:              "user-1",
			PhoneNumber:     "+233240000111",
			DisplayName:     "Kwame",
			StaffNumber:     "GES-OLD-1",
			Institution:     "GES",
			GhanaCardNumber: "GHA-OLD-1",
			Role:            "customer",
		},
	}

	service := &UserService{
		userRepo: repo,
		cfg: config.Config{
			AdminPhoneNumber: "0540000001",
		},
	}

	phoneNumber := "233540000001"
	_, err := service.UpdateProfile(context.Background(), "user-1", UpdateProfileInput{
		PhoneNumber: &phoneNumber,
	})
	if err == nil {
		t.Fatal("expected validation error when non-admin claims configured admin phone")
	}
}

func TestUserServiceListUsersRejectsInvalidRoleFilter(t *testing.T) {
	t.Parallel()

	service := &UserService{userRepo: &stubUserServiceRepo{}}
	_, err := service.ListUsers(context.Background(), "superadmin", 0, 20)
	if err == nil {
		t.Fatal("expected invalid role validation error")
	}
}
