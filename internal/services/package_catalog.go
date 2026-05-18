package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/jackc/pgx/v5"
)

type packageProductRepository interface {
	FindByID(ctx context.Context, id string) (*models.Product, error)
	FindByLegacyID(ctx context.Context, legacyID int) (*models.Product, error)
}

type packageProductImageRepository interface {
	FindByProductID(ctx context.Context, productID string) ([]models.ProductImage, error)
}

type packageRepository interface {
	ListFixed(ctx context.Context, includeInactive bool) ([]models.FixedPackage, error)
	FindFixedByID(ctx context.Context, id string, includeInactive bool) (*models.FixedPackage, error)
	CreateFixed(ctx context.Context, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error)
	UpdateFixed(ctx context.Context, id string, pkg *models.FixedPackage, sortOrder int) (*models.FixedPackage, error)
	DeleteFixed(ctx context.Context, id string) error
	HardDeleteFixed(ctx context.Context, id string) error
	ReactivateFixed(ctx context.Context, id string) (*models.FixedPackage, error)
	ListDepartment(ctx context.Context, kind string, includeInactive bool) ([]models.SimplePackage, error)
	FindDepartmentByID(ctx context.Context, kind, id string, includeInactive bool) (*models.SimplePackage, error)
	CreateDepartment(ctx context.Context, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error)
	UpdateDepartment(ctx context.Context, id, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error)
	DeleteDepartment(ctx context.Context, id string) error
	ReactivateDepartment(ctx context.Context, id, kind string) (*models.SimplePackage, error)
}

type packageApplicationRepository interface {
	CountByFixedPackage(ctx context.Context, packageID, packageName string) (int, error)
}

type PackageService struct {
	productRepo      packageProductRepository
	productImageRepo packageProductImageRepository
	packageRepo      packageRepository
	applicationRepo  packageApplicationRepository
	cfg              config.Config
}

func NewPackageService(productRepo *repositories.ProductRepository, productImageRepo *repositories.ProductImageRepository, packageRepo *repositories.PackageRepository, applicationRepo *repositories.ApplicationRepository, cfg config.Config) *PackageService {
	return &PackageService{productRepo: productRepo, productImageRepo: productImageRepo, packageRepo: packageRepo, applicationRepo: applicationRepo, cfg: cfg}
}

func (s *PackageService) GetCatalog(ctx context.Context) (*models.PackageCatalog, error) {
	fixed, err := s.GetFixedPackages(ctx)
	if err != nil {
		return nil, err
	}

	return &models.PackageCatalog{
		MinOrder:           s.cfg.MinOrder,
		PackageOptions:     packageOptions(fixed),
		FixedPackages:      fixed,
		ProvisionsPackages: s.GetProvisionsPackages(ctx),
		DetergentPackages:  s.GetDetergentPackages(ctx),
	}, nil
}

func (s *PackageService) GetAdminCatalog(ctx context.Context) (*models.PackageCatalog, error) {
	activeFixed, err := s.GetFixedPackages(ctx)
	if err != nil {
		return nil, err
	}

	allFixed, err := s.GetFixedPackagesAdmin(ctx)
	if err != nil {
		return nil, err
	}

	return &models.PackageCatalog{
		MinOrder:           s.cfg.MinOrder,
		PackageOptions:     packageOptions(activeFixed),
		FixedPackages:      allFixed,
		ProvisionsPackages: s.GetProvisionsPackagesAdmin(ctx),
		DetergentPackages:  s.GetDetergentPackagesAdmin(ctx),
	}, nil
}

func (s *PackageService) GetFixedPackages(ctx context.Context) ([]models.FixedPackage, error) {
	packages, err := s.packageRepo.ListFixed(ctx, false)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackages(ctx, packages)
}

func (s *PackageService) GetFixedPackagesAdmin(ctx context.Context) ([]models.FixedPackage, error) {
	packages, err := s.packageRepo.ListFixed(ctx, true)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackages(ctx, packages)
}

func (s *PackageService) GetFixedPackage(ctx context.Context, id string) (*models.FixedPackage, error) {
	pkg, err := s.packageRepo.FindFixedByID(ctx, strings.TrimSpace(id), false)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackage(ctx, pkg)
}

func (s *PackageService) GetFixedPackageAdmin(ctx context.Context, id string) (*models.FixedPackage, error) {
	pkg, err := s.packageRepo.FindFixedByID(ctx, strings.TrimSpace(id), true)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackage(ctx, pkg)
}

func (s *PackageService) GetProvisionsPackages(ctx context.Context) []models.SimplePackage {
	packages, err := s.packageRepo.ListDepartment(ctx, "provisions", false)
	if err != nil {
		return []models.SimplePackage{}
	}
	return packages
}

func (s *PackageService) GetDetergentPackages(ctx context.Context) []models.SimplePackage {
	packages, err := s.packageRepo.ListDepartment(ctx, "detergents", false)
	if err != nil {
		return []models.SimplePackage{}
	}
	return packages
}

func (s *PackageService) GetProvisionsPackagesAdmin(ctx context.Context) []models.SimplePackage {
	packages, err := s.packageRepo.ListDepartment(ctx, "provisions", true)
	if err != nil {
		return []models.SimplePackage{}
	}
	return packages
}

func (s *PackageService) GetDetergentPackagesAdmin(ctx context.Context) []models.SimplePackage {
	packages, err := s.packageRepo.ListDepartment(ctx, "detergents", true)
	if err != nil {
		return []models.SimplePackage{}
	}
	return packages
}

func (s *PackageService) CreateFixedPackage(ctx context.Context, pkg models.FixedPackage) (*models.FixedPackage, error) {
	normalized, err := normalizeFixedPackage(pkg)
	if err != nil {
		return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"package": {err.Error()},
		})
	}

	if _, err := s.packageRepo.FindFixedByID(ctx, normalized.ID, true); err == nil {
		return nil, apperrors.New(apperrors.KindConflict, "Package ID already exists", map[string][]string{
			"id": {"this package id is already in use"},
		})
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	sortOrder := s.nextFixedSortOrder(ctx)
	created, err := s.packageRepo.CreateFixed(ctx, &normalized, sortOrder)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackage(ctx, created)
}

func (s *PackageService) UpdateFixedPackage(ctx context.Context, id string, pkg models.FixedPackage) (*models.FixedPackage, error) {
	current, err := s.packageRepo.FindFixedByID(ctx, strings.TrimSpace(id), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("fixed package not found")
		}
		return nil, err
	}

	normalized, err := normalizeFixedPackage(pkg)
	if err != nil {
		return nil, err
	}

	if fixedPackageItemsChanged(current.Items, normalized.Items) {
		if s.applicationRepo == nil {
			return nil, fmt.Errorf("application repository not configured")
		}

		count, err := s.applicationRepo.CountByFixedPackage(ctx, current.ID, current.Name)
		if err != nil {
			return nil, err
		}
		if count > 0 && !s.cfg.AllowCatalogHardDeleteWithApplications {
			return nil, apperrors.New(apperrors.KindConflict, "Fixed package items cannot be changed after applications have been submitted", map[string][]string{
				"items": {"this fixed package is already used by one or more applications"},
			})
		}
	}

	updated, err := s.packageRepo.UpdateFixed(ctx, current.ID, &normalized, current.SortOrder)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackage(ctx, updated)
}

type DeleteFixedPackageResult struct {
	HardDeleted bool
	Message     string
}

func (s *PackageService) DeleteFixedPackage(ctx context.Context, id string) (*DeleteFixedPackageResult, error) {
	id = strings.TrimSpace(id)
	deleteFn := s.packageRepo.DeleteFixed
	result := &DeleteFixedPackageResult{
		HardDeleted: false,
		Message:     "Fixed package deactivated successfully",
	}
	if s.cfg.AllowCatalogHardDeleteWithApplications {
		deleteFn = s.packageRepo.HardDeleteFixed
		result.HardDeleted = true
		result.Message = "Fixed package deleted successfully"
	}

	if err := deleteFn(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Fixed package not found", map[string][]string{
				"id": {"unknown fixed package"},
			})
		}
		return nil, err
	}
	return result, nil
}

func (s *PackageService) ReactivateFixedPackage(ctx context.Context, id string) (*models.FixedPackage, error) {
	pkg, err := s.packageRepo.ReactivateFixed(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Fixed package not found", map[string][]string{
				"id": {"unknown fixed package"},
			})
		}
		return nil, err
	}
	return s.hydrateFixedPackage(ctx, pkg)
}

func (s *PackageService) GetDepartmentPackage(ctx context.Context, kind, id string, includeInactive bool) (*models.SimplePackage, error) {
	kind, err := normalizePackageKind(kind)
	if err != nil {
		return nil, err
	}

	pkg, err := s.packageRepo.FindDepartmentByID(ctx, kind, strings.TrimSpace(id), includeInactive)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

func (s *PackageService) CreateDepartmentPackage(ctx context.Context, kind string, pkg models.SimplePackage) (*models.SimplePackage, error) {
	kind, err := normalizePackageKind(kind)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeSimplePackage(pkg)
	if err != nil {
		return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"package": {err.Error()},
		})
	}

	if exists, err := s.departmentPackageIDExists(ctx, normalized.ID); err != nil {
		return nil, err
	} else if exists {
		return nil, apperrors.New(apperrors.KindConflict, "Package ID already exists", map[string][]string{
			"id": {"this package id is already in use"},
		})
	}

	sortOrder := s.nextDepartmentSortOrder(ctx, kind)
	created, err := s.packageRepo.CreateDepartment(ctx, kind, &normalized, sortOrder)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *PackageService) departmentPackageIDExists(ctx context.Context, id string) (bool, error) {
	for _, kind := range []string{"provisions", "detergents"} {
		if _, err := s.packageRepo.FindDepartmentByID(ctx, kind, id, true); err == nil {
			return true, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
	}
	return false, nil
}

func (s *PackageService) UpdateDepartmentPackage(ctx context.Context, kind, id string, pkg models.SimplePackage) (*models.SimplePackage, error) {
	kind, err := normalizePackageKind(kind)
	if err != nil {
		return nil, err
	}

	current, err := s.packageRepo.FindDepartmentByID(ctx, kind, strings.TrimSpace(id), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("department package not found")
		}
		return nil, err
	}

	normalized, err := normalizeSimplePackage(pkg)
	if err != nil {
		return nil, err
	}

	updated, err := s.packageRepo.UpdateDepartment(ctx, strings.TrimSpace(id), kind, &normalized, current.SortOrder)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *PackageService) DeleteDepartmentPackage(ctx context.Context, id string) error {
	if err := s.packageRepo.DeleteDepartment(ctx, strings.TrimSpace(id)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.New(apperrors.KindNotFound, "Department package not found", map[string][]string{
				"id": {"unknown department package"},
			})
		}
		return err
	}
	return nil
}

func (s *PackageService) ReactivateDepartmentPackage(ctx context.Context, kind, id string) (*models.SimplePackage, error) {
	kind, err := normalizePackageKind(kind)
	if err != nil {
		return nil, err
	}

	pkg, err := s.packageRepo.ReactivateDepartment(ctx, strings.TrimSpace(id), kind)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Department package not found", map[string][]string{
				"id": {"unknown department package"},
			})
		}
		return nil, err
	}
	return pkg, nil
}

func (s *PackageService) hydrateFixedPackages(ctx context.Context, packages []models.FixedPackage) ([]models.FixedPackage, error) {
	out := make([]models.FixedPackage, 0, len(packages))
	for _, pkg := range packages {
		updated, err := s.hydrateFixedPackage(ctx, &pkg)
		if err != nil {
			return nil, err
		}
		out = append(out, *updated)
	}
	return out, nil
}

func (s *PackageService) hydrateFixedPackage(ctx context.Context, pkg *models.FixedPackage) (*models.FixedPackage, error) {
	if pkg == nil {
		return nil, nil
	}

	for i := range pkg.Items {
		product, err := s.productByItemID(ctx, pkg.Items[i].ProductID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		pkg.Items[i].Product = product
		if pkg.Items[i].ImageURL == "" {
			pkg.Items[i].ImageURL = product.ImageURL
		}
	}
	return pkg, nil
}

func (s *PackageService) productByItemID(ctx context.Context, productID string) (*models.Product, error) {
	trimmed := strings.TrimSpace(productID)
	if trimmed == "" {
		return nil, pgx.ErrNoRows
	}

	product, err := s.productRepo.FindByID(ctx, trimmed)
	if err == nil {
		return product, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if legacyID, err := strconv.Atoi(trimmed); err == nil {
		product, err := s.productRepo.FindByLegacyID(ctx, legacyID)
		if err != nil {
			return nil, err
		}
		return product, nil
	}

	return nil, pgx.ErrNoRows
}

func (s *PackageService) nextFixedSortOrder(ctx context.Context) int {
	packages, err := s.packageRepo.ListFixed(ctx, true)
	if err != nil {
		return 1
	}
	max := 0
	for _, pkg := range packages {
		if pkg.SortOrder > max {
			max = pkg.SortOrder
		}
	}
	return max + 1
}

func (s *PackageService) nextDepartmentSortOrder(ctx context.Context, kind string) int {
	packages, err := s.packageRepo.ListDepartment(ctx, kind, true)
	if err != nil {
		return 1
	}
	max := 0
	for _, pkg := range packages {
		if pkg.SortOrder > max {
			max = pkg.SortOrder
		}
	}
	return max + 1
}

func normalizeFixedPackage(pkg models.FixedPackage) (models.FixedPackage, error) {
	pkg.ID = strings.TrimSpace(pkg.ID)
	pkg.Name = strings.TrimSpace(pkg.Name)
	pkg.Tagline = strings.TrimSpace(pkg.Tagline)
	pkg.Price = strings.TrimSpace(pkg.Price)
	pkg.Monthly = strings.TrimSpace(pkg.Monthly)
	pkg.Tag = strings.TrimSpace(pkg.Tag)
	pkg.RiceOptions = strings.TrimSpace(pkg.RiceOptions)
	if pkg.ID == "" || pkg.Name == "" || pkg.Price == "" || pkg.Monthly == "" {
		return models.FixedPackage{}, fmt.Errorf("invalid fixed package")
	}
	if pkg.Items == nil {
		pkg.Items = []models.PackageItem{}
	}
	return pkg, nil
}

func normalizeSimplePackage(pkg models.SimplePackage) (models.SimplePackage, error) {
	pkg.ID = strings.TrimSpace(pkg.ID)
	pkg.Name = strings.TrimSpace(pkg.Name)
	pkg.Items = strings.TrimSpace(pkg.Items)
	if pkg.ID == "" || pkg.Name == "" || pkg.Items == "" || pkg.Price <= 0 {
		return models.SimplePackage{}, fmt.Errorf("invalid package")
	}
	return pkg, nil
}

func normalizePackageKind(kind string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "provisions", "detergents":
		return kind, nil
	default:
		return "", fmt.Errorf("invalid package kind")
	}
}

func packageOptions(fixed []models.FixedPackage) []string {
	opts := make([]string, 0, len(fixed)+1)
	for _, pkg := range fixed {
		opts = append(opts, pkg.Name+" ("+pkg.Price+")")
	}
	opts = append(opts, "CUSTOMIZED REQUEST (Call/WhatsApp 0244854206)")
	return opts
}

func cloneFixedPackages(pkgs []models.FixedPackage) []models.FixedPackage {
	if pkgs == nil {
		return []models.FixedPackage{}
	}
	out := make([]models.FixedPackage, len(pkgs))
	copy(out, pkgs)
	return out
}

func cloneSimplePackages(pkgs []models.SimplePackage) []models.SimplePackage {
	if pkgs == nil {
		return []models.SimplePackage{}
	}
	out := make([]models.SimplePackage, len(pkgs))
	copy(out, pkgs)
	return out
}

func unmarshalItems(data []byte) ([]models.PackageItem, error) {
	var items []models.PackageItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func fixedPackageItemsChanged(current, next []models.PackageItem) bool {
	if len(current) != len(next) {
		return true
	}

	currentCounts := make(map[string]int, len(current))
	for _, item := range current {
		currentCounts[fixedPackageItemKey(item)]++
	}

	for _, item := range next {
		key := fixedPackageItemKey(item)
		if currentCounts[key] == 0 {
			return true
		}
		currentCounts[key]--
	}

	return false
}

func fixedPackageItemKey(item models.PackageItem) string {
	return strings.Join([]string{
		strings.TrimSpace(item.ProductID),
		strconv.Itoa(item.Qty),
		strings.TrimSpace(item.Label),
		strings.TrimSpace(item.Emoji),
	}, "\x00")
}
