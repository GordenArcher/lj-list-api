package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	ListDepartment(ctx context.Context, kind string, includeInactive bool) ([]models.SimplePackage, error)
	FindDepartmentByID(ctx context.Context, kind, id string, includeInactive bool) (*models.SimplePackage, error)
	CreateDepartment(ctx context.Context, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error)
	UpdateDepartment(ctx context.Context, id, kind string, pkg *models.SimplePackage, sortOrder int) (*models.SimplePackage, error)
	DeleteDepartment(ctx context.Context, id string) error
}

type PackageService struct {
	productRepo      packageProductRepository
	productImageRepo packageProductImageRepository
	packageRepo      packageRepository
	cfg              config.Config
}

func NewPackageService(productRepo *repositories.ProductRepository, productImageRepo *repositories.ProductImageRepository, packageRepo *repositories.PackageRepository, cfg config.Config) *PackageService {
	return &PackageService{productRepo: productRepo, productImageRepo: productImageRepo, packageRepo: packageRepo, cfg: cfg}
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

	updated, err := s.packageRepo.UpdateFixed(ctx, current.ID, &normalized, current.SortOrder)
	if err != nil {
		return nil, err
	}
	return s.hydrateFixedPackage(ctx, updated)
}

func (s *PackageService) DeleteFixedPackage(ctx context.Context, id string) error {
	return s.packageRepo.DeleteFixed(ctx, strings.TrimSpace(id))
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
		return nil, err
	}

	sortOrder := s.nextDepartmentSortOrder(ctx, kind)
	created, err := s.packageRepo.CreateDepartment(ctx, kind, &normalized, sortOrder)
	if err != nil {
		return nil, err
	}
	return created, nil
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
	return s.packageRepo.DeleteDepartment(ctx, strings.TrimSpace(id))
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
