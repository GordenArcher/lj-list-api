package services

import (
	"context"
	"testing"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
)

type stubPackageProductRepo struct {
	byID       map[string]*models.Product
	byLegacyID map[int]*models.Product
}

func (r *stubPackageProductRepo) FindByID(ctx context.Context, id string) (*models.Product, error) {
	if product, ok := r.byID[id]; ok && product != nil {
		copy := *product
		return &copy, nil
	}
	return nil, pgx.ErrNoRows
}

func (r *stubPackageProductRepo) FindByLegacyID(ctx context.Context, legacyID int) (*models.Product, error) {
	if product, ok := r.byLegacyID[legacyID]; ok && product != nil {
		copy := *product
		return &copy, nil
	}
	return nil, pgx.ErrNoRows
}

type stubPackageImageRepo struct {
	byProductID map[string][]models.ProductImage
}

func (r *stubPackageImageRepo) FindByProductID(ctx context.Context, productID string) ([]models.ProductImage, error) {
	if images, ok := r.byProductID[productID]; ok {
		return append([]models.ProductImage(nil), images...), nil
	}
	return []models.ProductImage{}, nil
}

type stubPackageRepo struct {
	fixed []models.FixedPackage
}

func (r *stubPackageRepo) ListFixed(ctx context.Context, includeInactive bool) ([]models.FixedPackage, error) {
	return append([]models.FixedPackage(nil), r.fixed...), nil
}

func (r *stubPackageRepo) FindFixedByID(ctx context.Context, id string, includeInactive bool) (*models.FixedPackage, error) {
	return nil, pgx.ErrNoRows
}

func (r *stubPackageRepo) CreateFixed(ctx context.Context, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error) {
	return nil, nil
}

func (r *stubPackageRepo) UpdateFixed(ctx context.Context, id string, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error) {
	return nil, nil
}

func (r *stubPackageRepo) DeleteFixed(ctx context.Context, id string) error {
	return nil
}

func (r *stubPackageRepo) ListDepartment(ctx context.Context, kind string, includeInactive bool) ([]models.SimplePackage, error) {
	return []models.SimplePackage{}, nil
}

func (r *stubPackageRepo) FindDepartmentByID(ctx context.Context, kind, id string, includeInactive bool) (*models.SimplePackage, error) {
	return nil, pgx.ErrNoRows
}

func (r *stubPackageRepo) CreateDepartment(ctx context.Context, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error) {
	return nil, nil
}

func (r *stubPackageRepo) UpdateDepartment(ctx context.Context, id, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error) {
	return nil, nil
}

func (r *stubPackageRepo) DeleteDepartment(ctx context.Context, id string) error {
	return nil
}

func TestPackageServiceGetFixedPackagesHydratesImages(t *testing.T) {
	t.Parallel()

	service := &PackageService{
		productRepo: &stubPackageProductRepo{
			byLegacyID: map[int]*models.Product{
				113: {
					ID:       "prod-113",
					Name:     "Ginny Viet 25kg (5*5)",
					ImageURL: "https://cdn.example.com/ginny.png",
				},
			},
			byID: map[string]*models.Product{
				"prod-113": {
					ID:       "prod-113",
					Name:     "Ginny Viet 25kg (5*5)",
					ImageURL: "https://cdn.example.com/ginny.png",
				},
			},
		},
		productImageRepo: &stubPackageImageRepo{
			byProductID: map[string][]models.ProductImage{
				"prod-113": {
					{
						ID:        "img-1",
						ProductID: "prod-113",
						ImageURL:  "https://cdn.example.com/ginny.png",
					},
				},
			},
		},
		packageRepo: &stubPackageRepo{
			fixed: []models.FixedPackage{
				{
					ID:    "abusua",
					Name:  "Abusua Asomdwee",
					Price: "GH₵569",
					Items: []models.PackageItem{{ProductID: "prod-113", Qty: 1}},
				},
			},
		},
		cfg: config.Config{MinOrder: 549},
	}

	packages, err := service.GetFixedPackages(context.Background())
	if err != nil {
		t.Fatalf("GetFixedPackages returned error: %v", err)
	}

	if len(packages) == 0 {
		t.Fatal("expected fixed packages")
	}
	if got := packages[0].Items[0].ImageURL; got != "https://cdn.example.com/ginny.png" {
		t.Fatalf("expected hydrated image url, got %q", got)
	}
	if packages[0].Items[0].Product == nil || packages[0].Items[0].Product.ID != "prod-113" {
		t.Fatalf("expected hydrated product snapshot, got %#v", packages[0].Items[0].Product)
	}
}
