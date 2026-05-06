package services

import (
	"context"
	"errors"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/jackc/pgx/v5"
)

type categoryRepository interface {
	List(ctx context.Context, includeInactive bool) ([]models.Category, error)
	FindByID(ctx context.Context, id string, includeInactive bool) (*models.Category, error)
	FindByName(ctx context.Context, name string, includeInactive bool) (*models.Category, error)
	Create(ctx context.Context, cat *models.Category) (*models.Category, error)
	Update(ctx context.Context, id string, cat *models.Category) (*models.Category, error)
	Deactivate(ctx context.Context, id string) error
	CountProductsByCategoryID(ctx context.Context, categoryID string) (int, error)
	UpdateProductCategoryName(ctx context.Context, categoryID, categoryName string) error
}

type CreateCategoryInput struct {
	Name      string
	SortOrder *int
	Active    *bool
}

type UpdateCategoryInput struct {
	Name      *string
	SortOrder *int
	Active    *bool
}

type CategoryService struct {
	categoryRepo categoryRepository
}

func NewCategoryService(categoryRepo *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

func (s *CategoryService) List(ctx context.Context, includeInactive bool) ([]models.Category, error) {
	cats, err := s.categoryRepo.List(ctx, includeInactive)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve categories", err)
	}
	return cats, nil
}

func (s *CategoryService) Get(ctx context.Context, id string) (*models.Category, error) {
	current, err := s.categoryRepo.FindByID(ctx, strings.TrimSpace(id), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Category not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve category", err)
	}
	return current, nil
}

func (s *CategoryService) Create(ctx context.Context, input CreateCategoryInput) (*models.Category, error) {
	active := true
	if input.Active != nil {
		active = *input.Active
	}

	normalized, err := normalizeCategoryInput(input.Name, input.SortOrder, &active)
	if err != nil {
		return nil, err
	}

	if normalized.SortOrder == 0 {
		max, err := s.nextSortOrder(ctx)
		if err != nil {
			return nil, err
		}
		normalized.SortOrder = max + 1
	}

	created, err := s.categoryRepo.Create(ctx, &normalized)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to create category", err)
	}
	return created, nil
}

func (s *CategoryService) Update(ctx context.Context, id string, input UpdateCategoryInput) (*models.Category, error) {
	current, err := s.categoryRepo.FindByID(ctx, strings.TrimSpace(id), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Category not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve category", err)
	}

	active := current.Active
	if input.Active != nil {
		active = *input.Active
	}

	sortOrder := current.SortOrder
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	name := current.Name
	if input.Name != nil && strings.TrimSpace(*input.Name) != "" {
		name = strings.TrimSpace(*input.Name)
	}

	normalized, err := normalizeCategoryInput(name, &sortOrder, &active)
	if err != nil {
		return nil, err
	}

	updated, err := s.categoryRepo.Update(ctx, current.ID, &normalized)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to update category", err)
	}

	if normalized.Name != current.Name {
		if err := s.categoryRepo.UpdateProductCategoryName(ctx, current.ID, normalized.Name); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to sync category name to products", err)
		}
	}

	return updated, nil
}

func (s *CategoryService) Delete(ctx context.Context, id string) (*models.Category, bool, error) {
	current, err := s.categoryRepo.FindByID(ctx, strings.TrimSpace(id), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, apperrors.New(apperrors.KindNotFound, "Category not found", nil)
		}
		return nil, false, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve category", err)
	}

	products, err := s.categoryRepo.CountProductsByCategoryID(ctx, current.ID)
	if err != nil {
		return nil, false, apperrors.Wrap(apperrors.KindInternal, "Failed to check category usage", err)
	}

	if products > 0 {
		if err := s.categoryRepo.Deactivate(ctx, current.ID); err != nil {
			return nil, false, apperrors.Wrap(apperrors.KindInternal, "Failed to deactivate category", err)
		}
		current.Active = false
		return current, true, nil
	}

	if err := s.categoryRepo.Deactivate(ctx, current.ID); err != nil {
		return nil, false, apperrors.Wrap(apperrors.KindInternal, "Failed to deactivate category", err)
	}
	current.Active = false
	return current, false, nil
}

func (s *CategoryService) nextSortOrder(ctx context.Context) (int, error) {
	cats, err := s.categoryRepo.List(ctx, true)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve categories", err)
	}

	max := 0
	for _, cat := range cats {
		if cat.SortOrder > max {
			max = cat.SortOrder
		}
	}
	return max, nil
}

func normalizeCategoryInput(name string, sortOrder *int, active *bool) (models.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Category{}, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"name": {"required"},
		})
	}
	cat := models.Category{Name: name}
	if sortOrder != nil {
		cat.SortOrder = *sortOrder
		if cat.SortOrder < 0 {
			return models.Category{}, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
				"sort_order": {"must be greater than or equal to 0"},
			})
		}
	}
	if active != nil {
		cat.Active = *active
	}
	return cat, nil
}
