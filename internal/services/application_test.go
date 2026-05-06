package services

import (
	"context"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
)

type stubApplicationRepo struct {
	createInput *models.Application
}

func (r *stubApplicationRepo) Create(ctx context.Context, app *models.Application) (*models.Application, error) {
	copy := *app
	r.createInput = &copy
	copy.ID = "app-1"
	copy.CreatedAt = time.Unix(1, 0).UTC()
	copy.UpdatedAt = time.Unix(2, 0).UTC()
	return &copy, nil
}

func (r *stubApplicationRepo) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]models.Application, error) {
	return nil, nil
}

func (r *stubApplicationRepo) CountByUserID(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (r *stubApplicationRepo) FindByID(ctx context.Context, id string) (*models.Application, error) {
	return nil, pgx.ErrNoRows
}

func (r *stubApplicationRepo) FindAll(ctx context.Context, status string, offset, limit int) ([]models.Application, error) {
	return nil, nil
}

func (r *stubApplicationRepo) CountAll(ctx context.Context, status string) (int, error) {
	return 0, nil
}

func (r *stubApplicationRepo) UpdateStatus(ctx context.Context, id, status string) (*models.Application, error) {
	return nil, pgx.ErrNoRows
}

type stubApplicationProductRepo struct{}

func (r *stubApplicationProductRepo) FindByID(ctx context.Context, id string) (*models.Product, error) {
	return nil, pgx.ErrNoRows
}

type stubApplicationUserRepo struct {
	user *models.User
	err  error
}

func (r *stubApplicationUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.user == nil {
		return nil, pgx.ErrNoRows
	}
	copy := *r.user
	return &copy, nil
}

func TestApplicationServiceSubmitFallsBackToUserProfileIdentityFields(t *testing.T) {
	t.Parallel()

	appRepo := &stubApplicationRepo{}
	service := &ApplicationService{
		applicationRepo: appRepo,
		productRepo:     &stubApplicationProductRepo{},
		userRepo: &stubApplicationUserRepo{
			user: &models.User{
				ID:              "user-1",
				StaffNumber:     "GES-2024-0018",
				Institution:     "Ghana Education Service",
				GhanaCardNumber: "GHA-123456789-0",
			},
		},
		cfg: config.Config{MinOrder: 549},
	}

	app, err := service.Submit(
		context.Background(),
		"user-1",
		"fixed",
		"Abusua Package",
		nil,
		"",
		"MND-001",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if appRepo.createInput == nil {
		t.Fatal("expected application to be created")
	}
	if appRepo.createInput.StaffNumber != "GES-2024-0018" {
		t.Fatalf("expected staff number fallback, got %q", appRepo.createInput.StaffNumber)
	}
	if appRepo.createInput.Institution != "Ghana Education Service" {
		t.Fatalf("expected institution fallback, got %q", appRepo.createInput.Institution)
	}
	if appRepo.createInput.GhanaCardNumber != "GHA-123456789-0" {
		t.Fatalf("expected Ghana Card fallback, got %q", appRepo.createInput.GhanaCardNumber)
	}
	if app == nil || app.ID != "app-1" {
		t.Fatalf("unexpected created application: %#v", app)
	}
}

func TestApplicationServiceSubmitPrefersRequestIdentityFields(t *testing.T) {
	t.Parallel()

	appRepo := &stubApplicationRepo{}
	service := &ApplicationService{
		applicationRepo: appRepo,
		productRepo:     &stubApplicationProductRepo{},
		userRepo: &stubApplicationUserRepo{
			user: &models.User{
				ID:              "user-1",
				StaffNumber:     "OLD-STAFF",
				Institution:     "Old Institution",
				GhanaCardNumber: "OLD-CARD",
			},
		},
		cfg: config.Config{MinOrder: 549},
	}

	_, err := service.Submit(
		context.Background(),
		"user-1",
		"fixed",
		"Abusua Package",
		nil,
		"NEW-STAFF",
		"MND-001",
		"New Institution",
		"NEW-CARD",
	)
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if appRepo.createInput == nil {
		t.Fatal("expected application to be created")
	}
	if appRepo.createInput.StaffNumber != "NEW-STAFF" {
		t.Fatalf("expected request staff number to win, got %q", appRepo.createInput.StaffNumber)
	}
	if appRepo.createInput.Institution != "New Institution" {
		t.Fatalf("expected request institution to win, got %q", appRepo.createInput.Institution)
	}
	if appRepo.createInput.GhanaCardNumber != "NEW-CARD" {
		t.Fatalf("expected request Ghana Card to win, got %q", appRepo.createInput.GhanaCardNumber)
	}
}

func TestApplicationServiceSubmitRejectsMissingIdentityFieldsWhenProfileAlsoMissing(t *testing.T) {
	t.Parallel()

	service := &ApplicationService{
		applicationRepo: &stubApplicationRepo{},
		productRepo:     &stubApplicationProductRepo{},
		userRepo: &stubApplicationUserRepo{
			user: &models.User{ID: "user-1"},
		},
		cfg: config.Config{MinOrder: 549},
	}

	_, err := service.Submit(
		context.Background(),
		"user-1",
		"fixed",
		"Abusua Package",
		nil,
		"",
		"MND-001",
		"",
		"",
	)
	if err == nil {
		t.Fatal("expected validation error for missing identity fields")
	}
}
