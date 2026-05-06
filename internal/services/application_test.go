package services

import (
	"context"
	"fmt"
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

type stubApplicationProductRepo struct {
	productByID       map[string]*models.Product
	productByLegacyID map[int]*models.Product
}

func (r *stubApplicationProductRepo) FindByID(ctx context.Context, id string) (*models.Product, error) {
	if r.productByID != nil {
		if product, ok := r.productByID[id]; ok && product != nil {
			copy := *product
			return &copy, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (r *stubApplicationProductRepo) FindByLegacyID(ctx context.Context, legacyID int) (*models.Product, error) {
	if r.productByLegacyID != nil {
		if product, ok := r.productByLegacyID[legacyID]; ok && product != nil {
			copy := *product
			return &copy, nil
		}
	}
	return nil, pgx.ErrNoRows
}

type stubApplicationPackageRepo struct {
	fixedByName map[string]*models.FixedPackage
}

func (r *stubApplicationPackageRepo) FindFixedByName(ctx context.Context, name string, includeInactive bool) (*models.FixedPackage, error) {
	if r.fixedByName != nil {
		if pkg, ok := r.fixedByName[name]; ok && pkg != nil {
			copy := *pkg
			return &copy, nil
		}
	}
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
		packageRepo: &stubApplicationPackageRepo{
			fixedByName: map[string]*models.FixedPackage{
				"Abusua Asomdwee":       &models.FixedPackage{ID: "abusua", Name: "Abusua Asomdwee", Price: "GH₵569"},
				"Medaase Medo":          &models.FixedPackage{ID: "medaase", Name: "Medaase Medo", Price: "GH₵769"},
				"You Do All":            &models.FixedPackage{ID: "youdo", Name: "You Do All", Price: "GH₵900"},
				"Super Love":            &models.FixedPackage{ID: "superlove", Name: "Super Love", Price: "GH₵1,289"},
				"Super Love Gye Wo Two": &models.FixedPackage{ID: "superlovegye", Name: "Super Love Gye Wo Two", Price: "GH₵1,980"},
				"Love Package":          &models.FixedPackage{ID: "valentine", Name: "Love Package", Price: "GH₵1,260"},
			},
		},
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
		"Abusua Asomdwee",
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
		packageRepo: &stubApplicationPackageRepo{
			fixedByName: map[string]*models.FixedPackage{
				"Abusua Asomdwee": &models.FixedPackage{ID: "abusua", Name: "Abusua Asomdwee", Price: "GH₵569"},
			},
		},
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
		"Abusua Asomdwee",
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
		packageRepo:     &stubApplicationPackageRepo{},
		userRepo: &stubApplicationUserRepo{
			user: &models.User{ID: "user-1"},
		},
		cfg: config.Config{MinOrder: 549},
	}

	_, err := service.Submit(
		context.Background(),
		"user-1",
		"fixed",
		"Abusua Asomdwee",
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

func TestApplicationServiceSubmitUsesFixedPackagePricing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		packageName string
		want        int
	}{
		{name: "abusua", packageName: "Abusua Asomdwee", want: 569},
		{name: "medaase", packageName: "Medaase Medo", want: 769},
		{name: "you do all", packageName: "You Do All", want: 900},
		{name: "super love", packageName: "Super Love", want: 1289},
		{name: "super love gye wo two", packageName: "Super Love Gye Wo Two", want: 1980},
		{name: "love package", packageName: "Love Package", want: 1260},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			appRepo := &stubApplicationRepo{}
			service := &ApplicationService{
				applicationRepo: appRepo,
				productRepo:     &stubApplicationProductRepo{},
				packageRepo: &stubApplicationPackageRepo{
					fixedByName: map[string]*models.FixedPackage{
						tc.packageName: &models.FixedPackage{ID: "pkg-1", Name: tc.packageName, Price: fmt.Sprintf("GH₵%d", tc.want)},
					},
				},
				userRepo: &stubApplicationUserRepo{
					user: &models.User{
						ID:              "user-1",
						StaffNumber:     "GES-2024-0018",
						Institution:     "Ghana Education Service",
						GhanaCardNumber: "GHA-123456789-0",
					},
				},
				cfg: config.Config{MinOrder: 1},
			}

			app, err := service.Submit(
				context.Background(),
				"user-1",
				"fixed",
				tc.packageName,
				nil,
				"",
				"MND-001",
				"",
				"",
			)
			if err != nil {
				t.Fatalf("Submit returned error: %v", err)
			}

			if app == nil || app.TotalAmount != tc.want {
				t.Fatalf("expected total amount %d, got %#v", tc.want, app)
			}
		})
	}
}

func TestApplicationServiceSubmitResolvesLegacyNumericProductIDs(t *testing.T) {
	t.Parallel()

	appRepo := &stubApplicationRepo{}
	productRepo := &stubApplicationProductRepo{
		productByLegacyID: map[int]*models.Product{
			101: &models.Product{
				ID:       "prod-uuid",
				LegacyID: intPtr(101),
				Name:     "Royal Aroma 25kg (5*5)",
				Price:    400,
				Unit:     "bag",
				Active:   true,
			},
		},
	}
	service := &ApplicationService{
		applicationRepo: appRepo,
		productRepo:     productRepo,
		packageRepo:     &stubApplicationPackageRepo{},
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
		"custom",
		"",
		[]CartItemInput{{ProductID: "101", Quantity: 2}},
		"",
		"MND-001",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if app == nil || app.ID != "app-1" {
		t.Fatalf("unexpected created application: %#v", app)
	}
	if appRepo.createInput == nil {
		t.Fatal("expected application to be created")
	}
	if got := appRepo.createInput.CartItems; len(got) != 1 || got[0].ProductID != "prod-uuid" {
		t.Fatalf("expected legacy product to resolve to uuid-backed row, got %#v", got)
	}
	if appRepo.createInput.TotalAmount != 800 {
		t.Fatalf("expected total amount 800, got %d", appRepo.createInput.TotalAmount)
	}
}

func intPtr(v int) *int {
	return &v
}
