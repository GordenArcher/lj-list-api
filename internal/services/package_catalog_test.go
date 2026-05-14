package services

import (
	"context"
	"errors"
	"testing"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
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
	fixed             []models.FixedPackage
	fixedByID         map[string]*models.FixedPackage
	departmentID      map[string]map[string]*models.SimplePackage
	updatedFixedItems []models.PackageItem
}

func (r *stubPackageRepo) ListFixed(ctx context.Context, includeInactive bool) ([]models.FixedPackage, error) {
	return append([]models.FixedPackage(nil), r.fixed...), nil
}

func (r *stubPackageRepo) FindFixedByID(ctx context.Context, id string, includeInactive bool) (*models.FixedPackage, error) {
	if r.fixedByID != nil {
		if pkg, ok := r.fixedByID[id]; ok && pkg != nil {
			copy := *pkg
			return &copy, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (r *stubPackageRepo) CreateFixed(ctx context.Context, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error) {
	return nil, nil
}

func (r *stubPackageRepo) UpdateFixed(ctx context.Context, id string, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error) {
	r.updatedFixedItems = append([]models.PackageItem(nil), pkg.Items...)
	copy := *pkg
	copy.ID = id
	copy.SortOrder = sortOrder
	return &copy, nil
}

func (r *stubPackageRepo) DeleteFixed(ctx context.Context, id string) error {
	return nil
}

func (r *stubPackageRepo) ReactivateFixed(ctx context.Context, id string) (*models.FixedPackage, error) {
	if r.fixedByID != nil {
		if pkg, ok := r.fixedByID[id]; ok && pkg != nil {
			copy := *pkg
			copy.Active = true
			return &copy, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (r *stubPackageRepo) ListDepartment(ctx context.Context, kind string, includeInactive bool) ([]models.SimplePackage, error) {
	return []models.SimplePackage{}, nil
}

func (r *stubPackageRepo) FindDepartmentByID(ctx context.Context, kind, id string, includeInactive bool) (*models.SimplePackage, error) {
	if r.departmentID != nil {
		if byID, ok := r.departmentID[kind]; ok {
			if pkg, ok := byID[id]; ok && pkg != nil {
				copy := *pkg
				return &copy, nil
			}
		}
	}
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

func (r *stubPackageRepo) ReactivateDepartment(ctx context.Context, id, kind string) (*models.SimplePackage, error) {
	if r.departmentID != nil {
		if byID, ok := r.departmentID[kind]; ok {
			if pkg, ok := byID[id]; ok && pkg != nil {
				copy := *pkg
				copy.Active = true
				return &copy, nil
			}
		}
	}
	return nil, pgx.ErrNoRows
}

type stubPackageApplicationRepo struct {
	fixedPackageCount int
}

func (r *stubPackageApplicationRepo) CountByFixedPackage(ctx context.Context, packageID, packageName string) (int, error) {
	return r.fixedPackageCount, nil
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

func TestPackageServiceCreateFixedPackageRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	service := &PackageService{
		packageRepo: &stubPackageRepo{
			fixedByID: map[string]*models.FixedPackage{
				"abusua": {ID: "abusua", Name: "Abusua Asomdwee", Price: "GH₵930", Monthly: "GH₵310/mo"},
			},
		},
	}

	_, err := service.CreateFixedPackage(context.Background(), models.FixedPackage{
		ID:      "abusua",
		Name:    "Abusua Asomdwee",
		Price:   "GH₵930",
		Monthly: "GH₵310/mo",
	})
	if err == nil {
		t.Fatal("expected duplicate package id error")
	}

	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperrors.KindConflict {
		t.Fatalf("expected conflict app error, got %#v", err)
	}
}

func TestPackageServiceUpdateFixedPackageRejectsItemChangesWhenApplicationsExist(t *testing.T) {
	t.Parallel()

	service := &PackageService{
		packageRepo: &stubPackageRepo{
			fixedByID: map[string]*models.FixedPackage{
				"abusua": {
					ID:      "abusua",
					Name:    "Abusua Asomdwee",
					Price:   "GH₵930",
					Monthly: "GH₵310/mo",
					Items: []models.PackageItem{
						{ProductID: "prod-1", Qty: 1, Label: "Rice"},
						{ProductID: "prod-2", Qty: 2, Label: "Oil"},
					},
				},
			},
		},
		applicationRepo: &stubPackageApplicationRepo{fixedPackageCount: 1},
	}

	_, err := service.UpdateFixedPackage(context.Background(), "abusua", models.FixedPackage{
		ID:      "abusua",
		Name:    "Abusua Asomdwee",
		Price:   "GH₵930",
		Monthly: "GH₵310/mo",
		Items: []models.PackageItem{
			{ProductID: "prod-1", Qty: 1, Label: "Rice"},
		},
	})
	if err == nil {
		t.Fatal("expected fixed package item change to be rejected")
	}

	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperrors.KindConflict {
		t.Fatalf("expected conflict app error, got %#v", err)
	}
}

func TestPackageServiceUpdateFixedPackageAllowsItemRemovalWhenUnused(t *testing.T) {
	t.Parallel()

	repo := &stubPackageRepo{
		fixedByID: map[string]*models.FixedPackage{
			"abusua": {
				ID:      "abusua",
				Name:    "Abusua Asomdwee",
				Price:   "GH₵930",
				Monthly: "GH₵310/mo",
				Items: []models.PackageItem{
					{ProductID: "prod-1", Qty: 1, Label: "Rice"},
					{ProductID: "prod-2", Qty: 2, Label: "Oil"},
				},
			},
		},
	}
	service := &PackageService{
		productRepo: &stubPackageProductRepo{
			byID: map[string]*models.Product{
				"prod-1": {ID: "prod-1", Name: "Rice"},
			},
		},
		packageRepo:     repo,
		applicationRepo: &stubPackageApplicationRepo{fixedPackageCount: 0},
	}

	updated, err := service.UpdateFixedPackage(context.Background(), "abusua", models.FixedPackage{
		ID:      "abusua",
		Name:    "Abusua Asomdwee",
		Price:   "GH₵930",
		Monthly: "GH₵310/mo",
		Items: []models.PackageItem{
			{ProductID: "prod-1", Qty: 1, Label: "Rice"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateFixedPackage returned error: %v", err)
	}
	if updated == nil || len(updated.Items) != 1 || updated.Items[0].ProductID != "prod-1" {
		t.Fatalf("expected one remaining item, got %#v", updated)
	}
	if len(repo.updatedFixedItems) != 1 || repo.updatedFixedItems[0].ProductID != "prod-1" {
		t.Fatalf("expected repository update with one item, got %#v", repo.updatedFixedItems)
	}
}

func TestPackageServiceCreateDepartmentPackageRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	service := &PackageService{
		packageRepo: &stubPackageRepo{
			departmentID: map[string]map[string]*models.SimplePackage{
				"provisions": {
					"maakye": {ID: "maakye", Name: "Maakye", Price: 250, Items: "1 Milo tin"},
				},
			},
		},
	}

	_, err := service.CreateDepartmentPackage(context.Background(), "provisions", models.SimplePackage{
		ID:    "maakye",
		Name:  "Maakye",
		Price: 250,
		Items: "1 Milo tin",
	})
	if err == nil {
		t.Fatal("expected duplicate package id error")
	}

	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperrors.KindConflict {
		t.Fatalf("expected conflict app error, got %#v", err)
	}
}

func TestPackageServiceCreateDepartmentPackageRejectsDuplicateIDInOtherKind(t *testing.T) {
	t.Parallel()

	service := &PackageService{
		packageRepo: &stubPackageRepo{
			departmentID: map[string]map[string]*models.SimplePackage{
				"detergents": {
					"shared": {ID: "shared", Name: "Detergent Package", Price: 270, Items: "Soap"},
				},
			},
		},
	}

	_, err := service.CreateDepartmentPackage(context.Background(), "provisions", models.SimplePackage{
		ID:    "shared",
		Name:  "Provision Package",
		Price: 250,
		Items: "Milo",
	})
	if err == nil {
		t.Fatal("expected duplicate package id error")
	}

	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperrors.KindConflict {
		t.Fatalf("expected conflict app error, got %#v", err)
	}
}
