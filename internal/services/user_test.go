package services

import (
	"context"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
)

type stubUserServiceRepo struct {
	currentUser       *models.User
	updatedUser       *models.User
	updateDisplayName string
	updatePhone       *string
	updateRole        string
}

func (r *stubUserServiceRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	if r.currentUser == nil {
		return nil, nil
	}
	userCopy := *r.currentUser
	if r.currentUser.Phone != nil {
		phoneCopy := *r.currentUser.Phone
		userCopy.Phone = &phoneCopy
	}
	return &userCopy, nil
}

func (r *stubUserServiceRepo) FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error) {
	return nil, nil
}

func (r *stubUserServiceRepo) CountAll(ctx context.Context, role string) (int, error) {
	return 0, nil
}

func (r *stubUserServiceRepo) Update(ctx context.Context, id, displayName string, phone *string, role string) (*models.User, error) {
	r.updateDisplayName = displayName
	r.updatePhone = cloneStringPtr(phone)
	r.updateRole = role

	if r.updatedUser != nil {
		userCopy := *r.updatedUser
		if r.updatedUser.Phone != nil {
			phoneCopy := *r.updatedUser.Phone
			userCopy.Phone = &phoneCopy
		}
		return &userCopy, nil
	}

	return &models.User{
		ID:          id,
		Email:       "user@example.com",
		DisplayName: displayName,
		Phone:       cloneStringPtr(phone),
		Role:        role,
		CreatedAt:   time.Unix(1, 0).UTC(),
		UpdatedAt:   time.Unix(2, 0).UTC(),
	}, nil
}

func TestUserServiceUpdateProfileNormalizesFields(t *testing.T) {
	t.Parallel()

	currentPhone := "0240000000"
	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:          "user-1",
			Email:       "kwame@example.com",
			DisplayName: "Kwame",
			Phone:       &currentPhone,
			Role:        "customer",
		},
	}

	service := &UserService{userRepo: repo}
	displayName := " Kwame Mensah "
	phone := "+233 24-000-0000"

	user, err := service.UpdateProfile(context.Background(), "user-1", UpdateProfileInput{
		DisplayName: &displayName,
		Phone:       &phone,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}

	if repo.updateDisplayName != "Kwame Mensah" {
		t.Fatalf("unexpected display name update: %q", repo.updateDisplayName)
	}
	if repo.updatePhone == nil || *repo.updatePhone != "+233240000000" {
		t.Fatalf("unexpected phone update: %#v", repo.updatePhone)
	}
	if user == nil || user.DisplayName != "Kwame Mensah" {
		t.Fatalf("unexpected user response: %#v", user)
	}
}

func TestUserServiceUpdateProfileAllowsClearingPhone(t *testing.T) {
	t.Parallel()

	currentPhone := "0240000000"
	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:          "user-1",
			Email:       "kwame@example.com",
			DisplayName: "Kwame",
			Phone:       &currentPhone,
			Role:        "customer",
		},
	}

	service := &UserService{userRepo: repo}
	phone := "   "

	_, err := service.UpdateProfile(context.Background(), "user-1", UpdateProfileInput{
		Phone: &phone,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}

	if repo.updatePhone != nil {
		t.Fatalf("expected phone to be cleared, got %#v", repo.updatePhone)
	}
}

func TestUserServiceAdminUpdateUserProtectsConfiguredAdminRole(t *testing.T) {
	t.Parallel()

	repo := &stubUserServiceRepo{
		currentUser: &models.User{
			ID:          "admin-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        "admin",
		},
	}

	service := &UserService{
		userRepo: repo,
		cfg: config.Config{
			AdminEmail: "admin@example.com",
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

func TestUserServiceListUsersRejectsInvalidRoleFilter(t *testing.T) {
	t.Parallel()

	service := &UserService{userRepo: &stubUserServiceRepo{}}
	_, err := service.ListUsers(context.Background(), "superadmin", 0, 20)
	if err == nil {
		t.Fatal("expected invalid role validation error")
	}
}
