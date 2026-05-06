package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/jackc/pgx/v5"
)

type productRepository interface {
	FindAll(ctx context.Context, category string, offset, limit int) ([]models.Product, error)
	CountAll(ctx context.Context, category string) (int, error)
	FindAllCategories(ctx context.Context) ([]string, error)
	FindByIDForAdmin(ctx context.Context, id string) (*models.Product, error)
	Create(ctx context.Context, name, category, unit string, price int, active bool) (*models.Product, error)
	Update(ctx context.Context, id, name, category, unit string, price int, active bool) (*models.Product, error)
	SetPrimaryImageURL(ctx context.Context, productID, imageURL string) error
}

type productImageRepository interface {
	FindByProductID(ctx context.Context, productID string) ([]models.ProductImage, error)
	FindByProductIDs(ctx context.Context, productIDs []string) (map[string][]models.ProductImage, error)
	FindByID(ctx context.Context, productID, imageID string) (*models.ProductImage, error)
	Create(ctx context.Context, productID, imageURL, cloudinaryPublicID string) (*models.ProductImage, error)
	Delete(ctx context.Context, productID, imageID string) error
}

type ProductService struct {
	productRepo      productRepository
	productImageRepo productImageRepository
	cfg              config.Config
	httpClient       *http.Client
	now              func() time.Time
}

type CreateProductInput struct {
	Name     string
	Category string
	Unit     string
	Price    int
	Active   *bool
}

type UpdateProductInput struct {
	Name     *string
	Category *string
	Unit     *string
	Price    *int
	Active   *bool
}

type ProductImageUploadInput struct {
	Image            io.Reader
	ImageFilename    string
	ImageContentType string
}

type cloudinaryUploadResponse struct {
	SecureURL string `json:"secure_url"`
	PublicID  string `json:"public_id"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type cloudinaryDestroyResponse struct {
	Result string `json:"result"`
}

func NewProductService(
	productRepo *repositories.ProductRepository,
	productImageRepo *repositories.ProductImageRepository,
	cfg config.Config,
) *ProductService {
	return &ProductService{
		productRepo:      productRepo,
		productImageRepo: productImageRepo,
		cfg:              cfg,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		now: time.Now,
	}
}

// GetProducts returns active products with their gallery images attached.
// image_url remains the primary image for backward compatibility, while the
// images array exposes the full gallery for newer clients.
func (s *ProductService) GetProducts(ctx context.Context, category string, offset, limit int) ([]models.Product, error) {
	products, err := s.productRepo.FindAll(ctx, category, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve products", err)
	}

	if err := s.attachImages(ctx, products); err != nil {
		return nil, err
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

// CreateProduct creates product metadata only. Images are managed through the
// dedicated product image endpoints so product creation is no longer limited
// to a single upload.
func (s *ProductService) CreateProduct(ctx context.Context, input CreateProductInput) (*models.Product, error) {
	normalized, err := normalizeCreateProductInput(input)
	if err != nil {
		return nil, err
	}

	active := true
	if normalized.Active != nil {
		active = *normalized.Active
	}

	product, err := s.productRepo.Create(
		ctx,
		normalized.Name,
		normalized.Category,
		normalized.Unit,
		normalized.Price,
		active,
	)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to create product", err)
	}

	product.Images = []models.ProductImage{}
	return product, nil
}

// UpdateProduct applies partial admin edits to the product record itself.
// Product images are managed separately via AddProductImages/DeleteProductImage.
func (s *ProductService) UpdateProduct(ctx context.Context, id string, input UpdateProductInput) (*models.Product, error) {
	current, err := s.productRepo.FindByIDForAdmin(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Product not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product", err)
	}

	if err := normalizeUpdateProductInput(&input); err != nil {
		return nil, err
	}

	name := current.Name
	category := current.Category
	unit := current.Unit
	price := current.Price
	active := current.Active
	changed := false

	if input.Name != nil {
		name = *input.Name
		changed = true
	}
	if input.Category != nil {
		category = *input.Category
		changed = true
	}
	if input.Unit != nil {
		unit = *input.Unit
		changed = true
	}
	if input.Price != nil {
		price = *input.Price
		changed = true
	}
	if input.Active != nil {
		active = *input.Active
		changed = true
	}

	if !changed {
		return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"product": {"at least one field must be provided"},
		})
	}

	product, err := s.productRepo.Update(ctx, current.ID, name, category, unit, price, active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Product not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to update product", err)
	}

	if err := s.attachImagesToProduct(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

// AddProductImages uploads one or more images, creates product_images rows,
// and then resyncs products.image_url to the first gallery image so older
// consumers still receive a representative image URL.
func (s *ProductService) AddProductImages(ctx context.Context, productID string, uploads []ProductImageUploadInput) ([]models.ProductImage, error) {
	productID = strings.TrimSpace(productID)
	if _, err := s.productRepo.FindByIDForAdmin(ctx, productID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Product not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product", err)
	}

	if len(uploads) == 0 {
		return nil, apperrors.New(apperrors.KindValidation, "Validation failed", map[string][]string{
			"images": {"at least one image is required"},
		})
	}

	created := make([]models.ProductImage, 0, len(uploads))
	for _, upload := range uploads {
		normalized, err := normalizeImageUploadInput(upload)
		if err != nil {
			return nil, err
		}

		imageURL, publicID, err := s.uploadProductImage(ctx, productID, normalized)
		if err != nil {
			return nil, err
		}

		image, err := s.productImageRepo.Create(ctx, productID, imageURL, publicID)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to create product image", err)
		}
		created = append(created, *image)
	}

	if err := s.syncPrimaryProductImage(ctx, productID); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *ProductService) GetProductImages(ctx context.Context, productID string) ([]models.ProductImage, error) {
	productID = strings.TrimSpace(productID)
	if _, err := s.productRepo.FindByIDForAdmin(ctx, productID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Product not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product", err)
	}

	images, err := s.productImageRepo.FindByProductID(ctx, productID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product images", err)
	}

	return images, nil
}

// DeleteProductImage removes the image row and destroys the Cloudinary asset.
// If the image being deleted is the primary/first gallery image, the product's
// image_url is updated to the next remaining image or cleared to empty.
func (s *ProductService) DeleteProductImage(ctx context.Context, productID, imageID string) error {
	productID = strings.TrimSpace(productID)
	imageID = strings.TrimSpace(imageID)

	if _, err := s.productRepo.FindByIDForAdmin(ctx, productID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.New(apperrors.KindNotFound, "Product not found", nil)
		}
		return apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product", err)
	}

	image, err := s.productImageRepo.FindByID(ctx, productID, imageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.New(apperrors.KindNotFound, "Product image not found", nil)
		}
		return apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product image", err)
	}

	publicID := strings.TrimSpace(image.CloudinaryPublicID)
	if publicID == "" {
		publicID = deriveCloudinaryPublicID(image.ImageURL)
	}
	if publicID != "" {
		if err := s.deleteCloudinaryImage(ctx, publicID); err != nil {
			return err
		}
	}

	if err := s.productImageRepo.Delete(ctx, productID, imageID); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to delete product image", err)
	}

	if err := s.syncPrimaryProductImage(ctx, productID); err != nil {
		return err
	}

	return nil
}

func normalizeCreateProductInput(input CreateProductInput) (CreateProductInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.Unit = strings.TrimSpace(input.Unit)

	errs := make(map[string][]string)
	if input.Name == "" {
		errs["name"] = []string{"required"}
	}
	if input.Category == "" {
		errs["category"] = []string{"required"}
	}
	if input.Unit == "" {
		errs["unit"] = []string{"required"}
	}
	if input.Price <= 0 {
		errs["price"] = []string{"must be greater than 0"}
	}
	if len(errs) > 0 {
		return CreateProductInput{}, apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	return input, nil
}

func normalizeUpdateProductInput(input *UpdateProductInput) error {
	errs := make(map[string][]string)

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			errs["name"] = []string{"cannot be empty"}
		} else {
			input.Name = &trimmed
		}
	}
	if input.Category != nil {
		trimmed := strings.TrimSpace(*input.Category)
		if trimmed == "" {
			errs["category"] = []string{"cannot be empty"}
		} else {
			input.Category = &trimmed
		}
	}
	if input.Unit != nil {
		trimmed := strings.TrimSpace(*input.Unit)
		if trimmed == "" {
			errs["unit"] = []string{"cannot be empty"}
		} else {
			input.Unit = &trimmed
		}
	}
	if input.Price != nil && *input.Price <= 0 {
		errs["price"] = []string{"must be greater than 0"}
	}

	if len(errs) > 0 {
		return apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	return nil
}

func normalizeImageUploadInput(input ProductImageUploadInput) (ProductImageUploadInput, error) {
	input.ImageFilename = strings.TrimSpace(input.ImageFilename)
	input.ImageContentType = strings.TrimSpace(input.ImageContentType)

	errs := make(map[string][]string)
	if input.Image == nil {
		errs["images"] = []string{"image file is required"}
	}
	if input.ImageContentType != "" && !strings.HasPrefix(strings.ToLower(input.ImageContentType), "image/") {
		errs["images"] = []string{"all uploaded files must be images"}
	}
	if len(errs) > 0 {
		return ProductImageUploadInput{}, apperrors.New(apperrors.KindValidation, "Validation failed", errs)
	}

	if input.ImageFilename == "" {
		input.ImageFilename = "product-image"
	}

	return input, nil
}

func (s *ProductService) attachImages(ctx context.Context, products []models.Product) error {
	productIDs := make([]string, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ID)
	}

	grouped, err := s.productImageRepo.FindByProductIDs(ctx, productIDs)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product images", err)
	}

	for i := range products {
		images := grouped[products[i].ID]
		if images == nil {
			images = []models.ProductImage{}
		}
		products[i].Images = images
		if products[i].ImageURL == "" && len(images) > 0 {
			products[i].ImageURL = images[0].ImageURL
		}
	}

	return nil
}

func (s *ProductService) attachImagesToProduct(ctx context.Context, product *models.Product) error {
	images, err := s.productImageRepo.FindByProductID(ctx, product.ID)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product images", err)
	}
	product.Images = images
	if product.ImageURL == "" && len(images) > 0 {
		product.ImageURL = images[0].ImageURL
	}
	return nil
}

func (s *ProductService) syncPrimaryProductImage(ctx context.Context, productID string) error {
	images, err := s.productImageRepo.FindByProductID(ctx, productID)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve product images", err)
	}

	primaryURL := ""
	if len(images) > 0 {
		primaryURL = images[0].ImageURL
	}

	if err := s.productRepo.SetPrimaryImageURL(ctx, productID, primaryURL); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to sync primary product image", err)
	}

	return nil
}

func (s *ProductService) uploadProductImage(ctx context.Context, productID string, input ProductImageUploadInput) (string, string, error) {
	if s.cfg.CloudinaryCloudName == "" || s.cfg.CloudinaryAPIKey == "" || s.cfg.CloudinaryAPISecret == "" {
		return "", "", apperrors.New(apperrors.KindInternal, "Product image upload is not configured", nil)
	}

	endpoint := fmt.Sprintf(
		"https://api.cloudinary.com/v1_1/%s/image/upload",
		url.PathEscape(s.cfg.CloudinaryCloudName),
	)
	timestamp := strconv.FormatInt(s.now().Unix(), 10)
	folder := "products/" + productID
	signature := signCloudinaryParams(map[string]string{
		"folder":    folder,
		"timestamp": timestamp,
	}, s.cfg.CloudinaryAPISecret)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("api_key", s.cfg.CloudinaryAPIKey); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to prepare image upload", err)
	}
	if err := writer.WriteField("timestamp", timestamp); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to prepare image upload", err)
	}
	if err := writer.WriteField("signature", signature); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to prepare image upload", err)
	}
	if err := writer.WriteField("folder", folder); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to prepare image upload", err)
	}

	part, err := createImagePart(writer, input.ImageFilename, input.ImageContentType)
	if err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to prepare image upload", err)
	}
	if _, err := io.Copy(part, input.Image); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to read image upload", err)
	}
	if err := writer.Close(); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to finalize image upload", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to create image upload request", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to upload product image", err)
	}
	defer resp.Body.Close()

	var uploadResp cloudinaryUploadResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&uploadResp); err != nil {
		return "", "", apperrors.Wrap(apperrors.KindInternal, "Failed to decode image upload response", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if uploadResp.Error != nil && strings.TrimSpace(uploadResp.Error.Message) != "" {
			return "", "", apperrors.New(apperrors.KindInternal, "Failed to upload product image", map[string][]string{
				"images": {uploadResp.Error.Message},
			})
		}
		return "", "", apperrors.New(apperrors.KindInternal, "Failed to upload product image", nil)
	}

	if strings.TrimSpace(uploadResp.SecureURL) == "" || strings.TrimSpace(uploadResp.PublicID) == "" {
		return "", "", apperrors.New(apperrors.KindInternal, "Failed to upload product image", map[string][]string{
			"images": {"cloudinary response did not include the expected asset identifiers"},
		})
	}

	return uploadResp.SecureURL, uploadResp.PublicID, nil
}

func (s *ProductService) deleteCloudinaryImage(ctx context.Context, publicID string) error {
	if s.cfg.CloudinaryCloudName == "" || s.cfg.CloudinaryAPIKey == "" || s.cfg.CloudinaryAPISecret == "" {
		return apperrors.New(apperrors.KindInternal, "Product image deletion is not configured", nil)
	}

	endpoint := fmt.Sprintf(
		"https://api.cloudinary.com/v1_1/%s/image/destroy",
		url.PathEscape(s.cfg.CloudinaryCloudName),
	)
	timestamp := strconv.FormatInt(s.now().Unix(), 10)
	form := url.Values{}
	form.Set("public_id", publicID)
	form.Set("timestamp", timestamp)
	form.Set("invalidate", "true")
	form.Set("api_key", s.cfg.CloudinaryAPIKey)
	form.Set("signature", signCloudinaryParams(map[string]string{
		"invalidate": "true",
		"public_id":  publicID,
		"timestamp":  timestamp,
	}, s.cfg.CloudinaryAPISecret))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to create image delete request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to delete product image asset", err)
	}
	defer resp.Body.Close()

	var destroyResp cloudinaryDestroyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&destroyResp); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "Failed to decode image delete response", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperrors.New(apperrors.KindInternal, "Failed to delete product image asset", map[string][]string{
			"images": {strings.TrimSpace(destroyResp.Result)},
		})
	}

	switch strings.ToLower(strings.TrimSpace(destroyResp.Result)) {
	case "", "ok", "not found":
		return nil
	default:
		return nil
	}
}

func signCloudinaryParams(params map[string]string, apiSecret string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}

	sum := sha1.Sum([]byte(strings.Join(parts, "&") + apiSecret))
	return hex.EncodeToString(sum[:])
}

func deriveCloudinaryPublicID(imageURL string) string {
	u, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil {
		return ""
	}

	const marker = "/upload/"
	idx := strings.Index(u.Path, marker)
	if idx < 0 {
		return ""
	}

	remainder := strings.TrimPrefix(u.Path[idx+len(marker):], "/")
	if remainder == "" {
		return ""
	}

	parts := strings.Split(remainder, "/")
	start := 0
	for i, part := range parts {
		if len(part) > 1 && part[0] == 'v' && isDigits(part[1:]) {
			start = i + 1
			break
		}
	}
	if start >= len(parts) {
		start = 0
	}

	publicPath := strings.Join(parts[start:], "/")
	publicPath = strings.TrimSuffix(publicPath, pathExt(publicPath))
	return strings.Trim(publicPath, "/")
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func pathExt(s string) string {
	lastSlash := strings.LastIndex(s, "/")
	lastDot := strings.LastIndex(s, ".")
	if lastDot <= lastSlash {
		return ""
	}
	return s[lastDot:]
}

func createImagePart(writer *multipart.Writer, filename, contentType string) (io.Writer, error) {
	safeFilename := filepath.Base(filename)
	if safeFilename == "." || safeFilename == "/" || safeFilename == "" {
		safeFilename = "product-image"
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, strings.ReplaceAll(safeFilename, `"`, "")))
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	return writer.CreatePart(header)
}
