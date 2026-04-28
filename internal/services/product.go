package services

import (
	"context"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
)

type ProductService struct {
	productRepo *repositories.ProductRepository
}

func NewProductService(productRepo *repositories.ProductRepository) *ProductService {
	return &ProductService{productRepo: productRepo}
}

// GetProducts returns a paginated list of active products, optionally filtered by category.
func (s *ProductService) GetProducts(ctx context.Context, category string, offset, limit int) ([]models.Product, error) {
	products, err := s.productRepo.FindAll(ctx, category, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve products", err)
	}
	return products, nil
}

// GetProductsCount returns the total count of active products, optionally filtered by category.
func (s *ProductService) GetProductsCount(ctx context.Context, category string) (int, error) {
	count, err := s.productRepo.CountAll(ctx, category)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product count", err)
	}
	return count, nil
}

func (s *ProductService) GetCategories(ctx context.Context) ([]string, error) {
	categories, err := s.productRepo.FindAllCategories(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve categories", err)
	}
	return categories, nil
}
