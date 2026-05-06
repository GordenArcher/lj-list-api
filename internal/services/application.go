package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/jackc/pgx/v5"
)

type ApplicationService struct {
	applicationRepo applicationRepository
	productRepo     applicationProductRepository
	packageRepo     applicationPackageRepository
	userRepo        applicationUserRepository
	cfg             config.Config
}

type applicationUserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
}

type applicationRepository interface {
	Create(ctx context.Context, app *models.Application) (*models.Application, error)
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]models.Application, error)
	CountByUserID(ctx context.Context, userID string) (int, error)
	FindByID(ctx context.Context, id string) (*models.Application, error)
	FindAll(ctx context.Context, status string, offset, limit int) ([]models.Application, error)
	CountAll(ctx context.Context, status string) (int, error)
	UpdateStatus(ctx context.Context, id, status string) (*models.Application, error)
}

type applicationProductRepository interface {
	FindByID(ctx context.Context, id string) (*models.Product, error)
	FindByLegacyID(ctx context.Context, legacyID int) (*models.Product, error)
}

type applicationPackageRepository interface {
	FindFixedByName(ctx context.Context, name string, includeInactive bool) (*models.FixedPackage, error)
}

func NewApplicationService(
	applicationRepo *repositories.ApplicationRepository,
	productRepo *repositories.ProductRepository,
	packageRepo *repositories.PackageRepository,
	userRepo *repositories.UserRepository,
	cfg config.Config,
) *ApplicationService {
	return &ApplicationService{
		applicationRepo: applicationRepo,
		productRepo:     productRepo,
		packageRepo:     packageRepo,
		userRepo:        userRepo,
		cfg:             cfg,
	}
}

// Submit validates and creates an application. For custom packages, it
// fetches current product details from the catalog and freezes them into
// the cart items so the application is a point-in-time snapshot. The minimum
// order threshold is enforced here, the handler layer also checks, but
// the service is the authoritative gate.
func (s *ApplicationService) Submit(ctx context.Context, userID, packageType, packageName string, cartItems []CartItemInput, staffNumber, mandateNumber, institution, ghanaCardNumber string) (*models.Application, error) {
	if packageType != "fixed" && packageType != "custom" {
		return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"package_type": {"must be 'fixed' or 'custom'"},
		})
	}

	user, err := s.userRepo.FindByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "User not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve user profile", err)
	}

	resolvedMandateNumber := strings.TrimSpace(mandateNumber)
	resolvedStaffNumber := resolveApplicationIdentityField(staffNumber, user.StaffNumber)
	resolvedInstitution := resolveApplicationIdentityField(institution, user.Institution)
	resolvedGhanaCardNumber := resolveApplicationIdentityField(ghanaCardNumber, user.GhanaCardNumber)

	errs := make(map[string][]string)
	if !utils.ValidateRequired(resolvedStaffNumber) {
		errs["staff_number"] = []string{"required on the request or user profile"}
	}
	if !utils.ValidateRequired(resolvedMandateNumber) {
		errs["mandate_number"] = []string{"required"}
	}
	if !utils.ValidateRequired(resolvedInstitution) {
		errs["institution"] = []string{"required on the request or user profile"}
	}
	if !utils.ValidateRequired(resolvedGhanaCardNumber) {
		errs["ghana_card_number"] = []string{"required on the request or user profile"}
	}
	if len(errs) > 0 {
		return nil, apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	var items []models.CartItem
	var total int

	if packageType == "custom" {
		// Validate cart items against the product catalog. Each product
		// must exist and be active. We fetch prices from the database at
		// submission time so the application is a frozen snapshot — if the
		// admin changes a price tomorrow, this application keeps the old one.
		for _, ci := range cartItems {
			if ci.Quantity <= 0 {
				return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
					"cart_items": {"quantity must be greater than 0"},
				})
			}

			product, err := s.findCartProduct(ctx, ci.ProductID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
						"cart_items": {"one or more products do not exist or are inactive"},
					})
				}
				return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to validate cart items", err)
			}

			subtotal := product.Price * ci.Quantity
			total += subtotal

			items = append(items, models.CartItem{
				ProductID: product.ID,
				Name:      product.Name,
				ImageURL:  product.ImageURL,
				Price:     product.Price,
				Quantity:  ci.Quantity,
				Subtotal:  subtotal,
			})
		}

		if len(items) == 0 {
			return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
				"cart_items": {"custom package must have at least one item"},
			})
		}
	} else {
		if strings.TrimSpace(packageName) == "" {
			return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
				"package_name": {"required for fixed packages"},
			})
		}

		// Fixed packages have no cart items. The total is resolved from the
		// active fixed_packages table so pricing stays in the database.
		pkg, err := s.packageRepo.FindFixedByName(ctx, packageName, false)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
					"package_name": {"unknown fixed package"},
				})
			}
			return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to resolve fixed package", err)
		}

		total, err = parseCurrencyAmount(pkg.Price)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to parse fixed package price", err)
		}
	}

	if total < s.cfg.MinOrder {
		msg := fmt.Sprintf("total must be at least GHC%d, got GHC%d", s.cfg.MinOrder, total)
		return nil, apperrors.New(apperrors.KindMinimumOrder, msg, map[string][]string{
			"cart": {msg},
		})
	}

	monthly := total / 3
	if total%3 != 0 {
		monthly++
	}

	app := &models.Application{
		UserID:          userID,
		PackageType:     packageType,
		PackageName:     packageName,
		CartItems:       items,
		TotalAmount:     total,
		MonthlyAmount:   monthly,
		StaffNumber:     resolvedStaffNumber,
		MandateNumber:   resolvedMandateNumber,
		Institution:     resolvedInstitution,
		GhanaCardNumber: resolvedGhanaCardNumber,
	}

	created, err := s.applicationRepo.Create(ctx, app)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to create application", err)
	}

	return created, nil
}

func (s *ApplicationService) findCartProduct(ctx context.Context, productID string) (*models.Product, error) {
	trimmed := strings.TrimSpace(productID)
	if trimmed == "" {
		return nil, pgx.ErrNoRows
	}

	if legacyID, err := strconv.Atoi(trimmed); err == nil && strconv.Itoa(legacyID) == trimmed {
		product, err := s.productRepo.FindByLegacyID(ctx, legacyID)
		if err == nil {
			return product, nil
		}
		return nil, err
	}

	return s.productRepo.FindByID(ctx, trimmed)
}

func (s *ApplicationService) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]models.Application, error) {
	apps, err := s.applicationRepo.FindByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve applications", err)
	}
	return apps, nil
}

// GetByUserIDCount returns the total count of applications for a user.
func (s *ApplicationService) GetByUserIDCount(ctx context.Context, userID string) (int, error) {
	count, err := s.applicationRepo.CountByUserID(ctx, userID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve application count", err)
	}
	return count, nil
}

func (s *ApplicationService) GetByID(ctx context.Context, id, userID string) (*models.Application, error) {
	app, err := s.applicationRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Application not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve application", err)
	}

	// Customers can only see their own applications. Admins bypass this.
	if userID != "" && app.UserID != userID {
		return nil, apperrors.New(apperrors.KindNotFound, "Application not found", nil)
	}

	return app, nil
}

func (s *ApplicationService) GetAll(ctx context.Context, status string, offset, limit int) ([]models.Application, error) {
	apps, err := s.applicationRepo.FindAll(ctx, status, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve applications", err)
	}
	return apps, nil
}

// GetAllCount returns the total count of all applications, optionally filtered by status.
func (s *ApplicationService) GetAllCount(ctx context.Context, status string) (int, error) {
	count, err := s.applicationRepo.CountAll(ctx, status)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve application count", err)
	}
	return count, nil
}

func (s *ApplicationService) UpdateStatus(ctx context.Context, id, status string) (*models.Application, error) {
	validStatuses := map[string]bool{"pending": true, "reviewed": true, "approved": true, "declined": true}
	if !validStatuses[status] {
		return nil, apperrors.New(apperrors.KindValidation, "Invalid status", map[string][]string{
			"status": {"must be pending, reviewed, approved, or declined"},
		})
	}

	app, err := s.applicationRepo.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Application not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to update application status", err)
	}

	return app, nil
}

// CartItemInput is the raw request shape for a cart item. It's separate
// from models.CartItem because the caller only sends product_id and
// quantity — the rest is populated by the service from the product catalog.
type CartItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func resolveApplicationIdentityField(requestValue, profileValue string) string {
	if trimmed := strings.TrimSpace(requestValue); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(profileValue)
}

func parseCurrencyAmount(value string) (int, error) {
	var digits strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}

	if digits.Len() == 0 {
		return 0, fmt.Errorf("no numeric amount in %q", value)
	}

	amount, err := strconv.Atoi(digits.String())
	if err != nil {
		return 0, err
	}

	return amount, nil
}
