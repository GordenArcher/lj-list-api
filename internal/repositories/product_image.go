package repositories

import (
	"context"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductImageRepository struct {
	pool *pgxpool.Pool
}

func NewProductImageRepository(pool *pgxpool.Pool) *ProductImageRepository {
	return &ProductImageRepository{pool: pool}
}

// FindByProductID returns every image for a product ordered oldest-first so
// the first uploaded image becomes the gallery's natural primary image unless
// an admin deletes it later.
func (r *ProductImageRepository) FindByProductID(ctx context.Context, productID string) ([]models.ProductImage, error) {
	query := `
		SELECT id, product_id, image_url, cloudinary_public_id, created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY created_at, id
	`

	rows, err := r.pool.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("query product images: %w", err)
	}
	defer rows.Close()

	var images []models.ProductImage
	for rows.Next() {
		var image models.ProductImage
		if err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.ImageURL,
			&image.CloudinaryPublicID,
			&image.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}
		images = append(images, image)
	}

	if images == nil {
		images = []models.ProductImage{}
	}

	return images, nil
}

// FindByProductIDs groups product images in one query so product list
// responses can include image galleries without doing one query per product.
func (r *ProductImageRepository) FindByProductIDs(ctx context.Context, productIDs []string) (map[string][]models.ProductImage, error) {
	grouped := make(map[string][]models.ProductImage, len(productIDs))
	if len(productIDs) == 0 {
		return grouped, nil
	}

	query := `
		SELECT id, product_id, image_url, cloudinary_public_id, created_at
		FROM product_images
		WHERE product_id = ANY($1)
		ORDER BY product_id, created_at, id
	`

	rows, err := r.pool.Query(ctx, query, productIDs)
	if err != nil {
		return nil, fmt.Errorf("query product images by product ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var image models.ProductImage
		if err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.ImageURL,
			&image.CloudinaryPublicID,
			&image.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}
		grouped[image.ProductID] = append(grouped[image.ProductID], image)
	}

	return grouped, nil
}

func (r *ProductImageRepository) FindByID(ctx context.Context, productID, imageID string) (*models.ProductImage, error) {
	query := `
		SELECT id, product_id, image_url, cloudinary_public_id, created_at
		FROM product_images
		WHERE product_id = $1 AND id = $2
	`

	var image models.ProductImage
	err := r.pool.QueryRow(ctx, query, productID, imageID).Scan(
		&image.ID,
		&image.ProductID,
		&image.ImageURL,
		&image.CloudinaryPublicID,
		&image.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find product image by id: %w", err)
	}

	return &image, nil
}

func (r *ProductImageRepository) Create(ctx context.Context, productID, imageURL, cloudinaryPublicID string) (*models.ProductImage, error) {
	query := `
		INSERT INTO product_images (product_id, image_url, cloudinary_public_id)
		VALUES ($1, $2, $3)
		RETURNING id, product_id, image_url, cloudinary_public_id, created_at
	`

	var image models.ProductImage
	err := r.pool.QueryRow(ctx, query, productID, imageURL, cloudinaryPublicID).Scan(
		&image.ID,
		&image.ProductID,
		&image.ImageURL,
		&image.CloudinaryPublicID,
		&image.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert product image: %w", err)
	}

	return &image, nil
}

func (r *ProductImageRepository) Delete(ctx context.Context, productID, imageID string) error {
	query := `
		DELETE FROM product_images
		WHERE product_id = $1 AND id = $2
	`

	result, err := r.pool.Exec(ctx, query, productID, imageID)
	if err != nil {
		return fmt.Errorf("delete product image: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete product image: no rows affected")
	}

	return nil
}
