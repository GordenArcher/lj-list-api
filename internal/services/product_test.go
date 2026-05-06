package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
)

type stubProductRepo struct {
	currentProduct     *models.Product
	createdName        string
	createdCategoryID  string
	createdCategory    string
	createdUnit        string
	createdPrice       int
	createdOldPrice    *int
	createdTag         string
	createdActive      bool
	updatedProduct     *models.Product
	primaryImageURL    string
	setPrimaryCalls    int
	findErr            error
	createErr          error
	updateErr          error
	setPrimaryImageErr error
}

func (r *stubProductRepo) FindAll(ctx context.Context, categoryID string, offset, limit int) ([]models.Product, error) {
	return nil, nil
}

func (r *stubProductRepo) FindAllAdmin(ctx context.Context, categoryID string, offset, limit int) ([]models.Product, error) {
	return nil, nil
}

func (r *stubProductRepo) CountAll(ctx context.Context, categoryID string) (int, error) {
	return 0, nil
}

func (r *stubProductRepo) CountAllAdmin(ctx context.Context, categoryID string) (int, error) {
	return 0, nil
}

func (r *stubProductRepo) FindAllCategories(ctx context.Context) ([]models.Category, error) {
	return nil, nil
}

func (r *stubProductRepo) FindCategoryByID(ctx context.Context, id string) (*models.Category, error) {
	return &models.Category{ID: id, Name: "Rice, Spaghetti & Grains"}, nil
}

func (r *stubProductRepo) FindCategoryByName(ctx context.Context, name string) (*models.Category, error) {
	return &models.Category{ID: "cat-1", Name: name}, nil
}

func (r *stubProductRepo) FindByID(ctx context.Context, id string) (*models.Product, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.currentProduct == nil {
		return nil, pgx.ErrNoRows
	}

	productCopy := *r.currentProduct
	return &productCopy, nil
}

func (r *stubProductRepo) FindByIDForAdmin(ctx context.Context, id string) (*models.Product, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.currentProduct == nil {
		return nil, pgx.ErrNoRows
	}

	productCopy := *r.currentProduct
	return &productCopy, nil
}

func (r *stubProductRepo) Create(ctx context.Context, name, categoryID, categoryName, unit string, price int, oldPrice *int, tag string, active bool) (*models.Product, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}

	r.createdName = name
	r.createdCategoryID = categoryID
	r.createdCategory = categoryName
	r.createdUnit = unit
	r.createdPrice = price
	r.createdOldPrice = oldPrice
	r.createdTag = tag
	r.createdActive = active

	product := &models.Product{
		ID:         "prod-1",
		CategoryID: categoryID,
		Name:       name,
		Category:   categoryName,
		Price:      price,
		OldPrice:   oldPrice,
		Tag:        tag,
		ImageURL:   "",
		Unit:       unit,
		Active:     active,
	}
	r.currentProduct = product

	return cloneProduct(product), nil
}

func (r *stubProductRepo) Update(ctx context.Context, id, name, categoryID, categoryName, unit string, price int, oldPrice *int, tag string, active bool) (*models.Product, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}

	imageURL := ""
	if r.currentProduct != nil {
		imageURL = r.currentProduct.ImageURL
	}

	product := &models.Product{
		ID:         id,
		CategoryID: categoryID,
		Name:       name,
		Category:   categoryName,
		Price:      price,
		OldPrice:   oldPrice,
		Tag:        tag,
		ImageURL:   imageURL,
		Unit:       unit,
		Active:     active,
	}
	r.updatedProduct = cloneProduct(product)
	r.currentProduct = cloneProduct(product)

	return cloneProduct(product), nil
}

func (r *stubProductRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *stubProductRepo) CountApplicationsByProductID(ctx context.Context, productID string) (int, error) {
	return 0, nil
}

func (r *stubProductRepo) SetPrimaryImageURL(ctx context.Context, productID, imageURL string) error {
	if r.setPrimaryImageErr != nil {
		return r.setPrimaryImageErr
	}

	r.primaryImageURL = imageURL
	r.setPrimaryCalls++
	if r.currentProduct != nil && r.currentProduct.ID == productID {
		r.currentProduct.ImageURL = imageURL
	}

	return nil
}

type stubProductImageRepo struct {
	imagesByProduct map[string][]models.ProductImage
	findErr         error
	createErr       error
	deleteErr       error
	nextID          int
}

func (r *stubProductImageRepo) FindByProductID(ctx context.Context, productID string) ([]models.ProductImage, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}

	images := r.imagesByProduct[productID]
	if images == nil {
		return []models.ProductImage{}, nil
	}

	return append([]models.ProductImage(nil), images...), nil
}

func (r *stubProductImageRepo) FindByProductIDs(ctx context.Context, productIDs []string) (map[string][]models.ProductImage, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}

	grouped := make(map[string][]models.ProductImage, len(productIDs))
	for _, productID := range productIDs {
		grouped[productID] = append([]models.ProductImage(nil), r.imagesByProduct[productID]...)
	}

	return grouped, nil
}

func (r *stubProductImageRepo) FindByID(ctx context.Context, productID, imageID string) (*models.ProductImage, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}

	for _, image := range r.imagesByProduct[productID] {
		if image.ID == imageID {
			imageCopy := image
			return &imageCopy, nil
		}
	}

	return nil, pgx.ErrNoRows
}

func (r *stubProductImageRepo) Create(ctx context.Context, productID, imageURL, cloudinaryPublicID string) (*models.ProductImage, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}

	r.nextID++
	image := models.ProductImage{
		ID:                 fmt.Sprintf("img-%d", r.nextID),
		ProductID:          productID,
		ImageURL:           imageURL,
		CloudinaryPublicID: cloudinaryPublicID,
		CreatedAt:          time.Unix(int64(r.nextID), 0).UTC(),
	}
	r.imagesByProduct[productID] = append(r.imagesByProduct[productID], image)

	imageCopy := image
	return &imageCopy, nil
}

func (r *stubProductImageRepo) Delete(ctx context.Context, productID, imageID string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}

	images := r.imagesByProduct[productID]
	for i, image := range images {
		if image.ID == imageID {
			r.imagesByProduct[productID] = append(images[:i:i], images[i+1:]...)
			return nil
		}
	}

	return pgx.ErrNoRows
}

func TestProductServiceCreateProductTrimsMetadataAndDefaultsActive(t *testing.T) {
	t.Parallel()

	repo := &stubProductRepo{}
	imageRepo := &stubProductImageRepo{imagesByProduct: map[string][]models.ProductImage{}}
	service := &ProductService{
		productRepo:      repo,
		productImageRepo: imageRepo,
	}

	product, err := service.CreateProduct(context.Background(), CreateProductInput{
		Name:       " Royal Aroma Rice 5kg ",
		CategoryID: "11111111-1111-1111-1111-111111111111",
		Unit:       " bag ",
		Price:      120,
		OldPrice:   productIntPtr(150),
		Tag:        " In Stock ",
	})
	if err != nil {
		t.Fatalf("CreateProduct returned error: %v", err)
	}

	if repo.createdName != "Royal Aroma Rice 5kg" {
		t.Fatalf("unexpected created name: %q", repo.createdName)
	}
	if repo.createdCategoryID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected created category id: %q", repo.createdCategoryID)
	}
	if repo.createdCategory != "Rice, Spaghetti & Grains" {
		t.Fatalf("unexpected created category name: %q", repo.createdCategory)
	}
	if repo.createdUnit != "bag" {
		t.Fatalf("unexpected created unit: %q", repo.createdUnit)
	}
	if repo.createdPrice != 120 {
		t.Fatalf("unexpected created price: %d", repo.createdPrice)
	}
	if repo.createdOldPrice == nil || *repo.createdOldPrice != 150 {
		t.Fatalf("unexpected created old price: %#v", repo.createdOldPrice)
	}
	if repo.createdTag != "In Stock" {
		t.Fatalf("unexpected created tag: %q", repo.createdTag)
	}
	if !repo.createdActive {
		t.Fatal("expected create to default active to true")
	}
	if product == nil || product.ID != "prod-1" {
		t.Fatalf("unexpected product result: %#v", product)
	}
	if product.ImageURL != "" {
		t.Fatalf("expected empty primary image on create, got %q", product.ImageURL)
	}
	if len(product.Images) != 0 {
		t.Fatalf("expected no gallery images on create, got %#v", product.Images)
	}
}

func TestProductServiceUpdateProductAttachesExistingImages(t *testing.T) {
	t.Parallel()

	repo := &stubProductRepo{
		currentProduct: &models.Product{
			ID:         "prod-1",
			CategoryID: "11111111-1111-1111-1111-111111111111",
			Name:       "Royal Aroma Rice 5kg",
			Category:   "Rice, Spaghetti & Grains",
			Price:      120,
			ImageURL:   "https://res.cloudinary.com/demo-cloud/image/upload/v1/products/prod-1/rice.jpg",
			Unit:       "bag",
			Active:     true,
		},
	}
	imageRepo := &stubProductImageRepo{
		imagesByProduct: map[string][]models.ProductImage{
			"prod-1": {
				{
					ID:                 "img-1",
					ProductID:          "prod-1",
					ImageURL:           "https://res.cloudinary.com/demo-cloud/image/upload/v1/products/prod-1/rice.jpg",
					CloudinaryPublicID: "products/prod-1/rice",
					CreatedAt:          time.Unix(1, 0).UTC(),
				},
			},
		},
	}
	service := &ProductService{
		productRepo:      repo,
		productImageRepo: imageRepo,
	}

	newPrice := 125
	newOldPrice := 160
	active := false
	product, err := service.UpdateProduct(context.Background(), "prod-1", UpdateProductInput{
		Price:    &newPrice,
		OldPrice: &newOldPrice,
		Tag:      productStringPtr("Premium"),
		Active:   &active,
	})
	if err != nil {
		t.Fatalf("UpdateProduct returned error: %v", err)
	}

	if repo.updatedProduct == nil {
		t.Fatal("expected repository update to be called")
	}
	if repo.updatedProduct.Price != 125 {
		t.Fatalf("expected updated price, got %d", repo.updatedProduct.Price)
	}
	if repo.updatedProduct.OldPrice == nil || *repo.updatedProduct.OldPrice != 160 {
		t.Fatalf("expected updated old price, got %#v", repo.updatedProduct.OldPrice)
	}
	if repo.updatedProduct.Tag != "Premium" {
		t.Fatalf("expected updated tag, got %q", repo.updatedProduct.Tag)
	}
	if repo.updatedProduct.Active {
		t.Fatalf("expected updated active flag to be false, got %v", repo.updatedProduct.Active)
	}
	if product == nil || len(product.Images) != 1 {
		t.Fatalf("expected product images to be attached, got %#v", product)
	}
	if product.Images[0].ID != "img-1" {
		t.Fatalf("unexpected attached image: %#v", product.Images[0])
	}
}

func TestProductServiceAddProductImagesUploadsGalleryAndSyncsPrimary(t *testing.T) {
	t.Parallel()

	repo := &stubProductRepo{
		currentProduct: &models.Product{
			ID:         "prod-1",
			CategoryID: "11111111-1111-1111-1111-111111111111",
			Name:       "Royal Aroma Rice 5kg",
			Category:   "Rice, Spaghetti & Grains",
			Price:      120,
			Unit:       "bag",
			Active:     true,
		},
	}
	imageRepo := &stubProductImageRepo{
		imagesByProduct: map[string][]models.ProductImage{},
	}

	uploadCalls := 0
	service := &ProductService{
		productRepo:      repo,
		productImageRepo: imageRepo,
		cfg: config.Config{
			CloudinaryCloudName: "demo-cloud",
			CloudinaryAPIKey:    "api-key",
			CloudinaryAPISecret: "api-secret",
		},
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				uploadCalls++

				if r.Method != http.MethodPost {
					t.Fatalf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/v1_1/demo-cloud/image/upload" {
					t.Fatalf("unexpected upload path: %s", r.URL.Path)
				}

				fields, filename, fileField, err := readMultipartRequest(r)
				if err != nil {
					t.Fatalf("failed to read multipart request: %v", err)
				}
				if fields["folder"] != "products/prod-1" {
					t.Fatalf("unexpected folder field: %q", fields["folder"])
				}
				if fields["api_key"] != "api-key" {
					t.Fatalf("unexpected api_key: %q", fields["api_key"])
				}
				if fields["timestamp"] == "" || fields["signature"] == "" {
					t.Fatalf("expected signed upload fields, got %#v", fields)
				}
				if fileField != "file" {
					t.Fatalf("expected file part named 'file', got %q", fileField)
				}

				switch uploadCalls {
				case 1:
					if filename != "rice-front.jpg" {
						t.Fatalf("unexpected first filename: %q", filename)
					}
					return jsonResponse(http.StatusOK, `{
						"secure_url":"https://res.cloudinary.com/demo-cloud/image/upload/v1714422000/products/prod-1/rice-front.jpg",
						"public_id":"products/prod-1/rice-front"
					}`), nil
				case 2:
					if filename != "rice-back.jpg" {
						t.Fatalf("unexpected second filename: %q", filename)
					}
					return jsonResponse(http.StatusOK, `{
						"secure_url":"https://res.cloudinary.com/demo-cloud/image/upload/v1714422001/products/prod-1/rice-back.jpg",
						"public_id":"products/prod-1/rice-back"
					}`), nil
				default:
					t.Fatalf("unexpected extra upload call: %d", uploadCalls)
					return nil, nil
				}
			}),
		},
		now: func() time.Time {
			return time.Unix(1714422000, 0).UTC()
		},
	}

	images, err := service.AddProductImages(context.Background(), "prod-1", []ProductImageUploadInput{
		{
			Image:            strings.NewReader("front-image-bytes"),
			ImageFilename:    "rice-front.jpg",
			ImageContentType: "image/jpeg",
		},
		{
			Image:            strings.NewReader("back-image-bytes"),
			ImageFilename:    "rice-back.jpg",
			ImageContentType: "image/jpeg",
		},
	})
	if err != nil {
		t.Fatalf("AddProductImages returned error: %v", err)
	}

	if uploadCalls != 2 {
		t.Fatalf("expected 2 upload calls, got %d", uploadCalls)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 created images, got %#v", images)
	}
	if repo.setPrimaryCalls != 1 {
		t.Fatalf("expected primary image sync once, got %d", repo.setPrimaryCalls)
	}
	if repo.primaryImageURL != "https://res.cloudinary.com/demo-cloud/image/upload/v1714422000/products/prod-1/rice-front.jpg" {
		t.Fatalf("unexpected primary image url: %q", repo.primaryImageURL)
	}
}

func TestProductServiceDeleteProductImageDeletesAssetAndPromotesNextImage(t *testing.T) {
	t.Parallel()

	repo := &stubProductRepo{
		currentProduct: &models.Product{
			ID:       "prod-1",
			Name:     "Royal Aroma Rice 5kg",
			Category: "Rice, Spaghetti & Grains",
			Price:    120,
			ImageURL: "https://res.cloudinary.com/demo-cloud/image/upload/v1714422000/products/prod-1/rice-front.jpg",
			Unit:     "bag",
			Active:   true,
		},
	}
	imageRepo := &stubProductImageRepo{
		imagesByProduct: map[string][]models.ProductImage{
			"prod-1": {
				{
					ID:                 "img-1",
					ProductID:          "prod-1",
					ImageURL:           "https://res.cloudinary.com/demo-cloud/image/upload/v1714422000/products/prod-1/rice-front.jpg",
					CloudinaryPublicID: "",
					CreatedAt:          time.Unix(1, 0).UTC(),
				},
				{
					ID:                 "img-2",
					ProductID:          "prod-1",
					ImageURL:           "https://res.cloudinary.com/demo-cloud/image/upload/v1714422001/products/prod-1/rice-back.jpg",
					CloudinaryPublicID: "products/prod-1/rice-back",
					CreatedAt:          time.Unix(2, 0).UTC(),
				},
			},
		},
	}

	destroyCalls := 0
	service := &ProductService{
		productRepo:      repo,
		productImageRepo: imageRepo,
		cfg: config.Config{
			CloudinaryCloudName: "demo-cloud",
			CloudinaryAPIKey:    "api-key",
			CloudinaryAPISecret: "api-secret",
		},
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				destroyCalls++

				if r.Method != http.MethodPost {
					t.Fatalf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/v1_1/demo-cloud/image/destroy" {
					t.Fatalf("unexpected destroy path: %s", r.URL.Path)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read destroy body: %v", err)
				}
				values, err := url.ParseQuery(string(body))
				if err != nil {
					t.Fatalf("failed to parse destroy form: %v", err)
				}
				if values.Get("public_id") != "products/prod-1/rice-front" {
					t.Fatalf("unexpected public_id: %q", values.Get("public_id"))
				}
				if values.Get("invalidate") != "true" {
					t.Fatalf("expected invalidate=true, got %q", values.Get("invalidate"))
				}
				if values.Get("api_key") != "api-key" || values.Get("timestamp") == "" || values.Get("signature") == "" {
					t.Fatalf("unexpected destroy form values: %#v", values)
				}

				return jsonResponse(http.StatusOK, `{"result":"ok"}`), nil
			}),
		},
		now: func() time.Time {
			return time.Unix(1714422000, 0).UTC()
		},
	}

	if err := service.DeleteProductImage(context.Background(), "prod-1", "img-1"); err != nil {
		t.Fatalf("DeleteProductImage returned error: %v", err)
	}

	if destroyCalls != 1 {
		t.Fatalf("expected 1 destroy call, got %d", destroyCalls)
	}
	if repo.primaryImageURL != "https://res.cloudinary.com/demo-cloud/image/upload/v1714422001/products/prod-1/rice-back.jpg" {
		t.Fatalf("unexpected primary image url after delete: %q", repo.primaryImageURL)
	}
	if got := imageRepo.imagesByProduct["prod-1"]; len(got) != 1 || got[0].ID != "img-2" {
		t.Fatalf("unexpected remaining images: %#v", got)
	}
}

func cloneProduct(product *models.Product) *models.Product {
	if product == nil {
		return nil
	}

	productCopy := *product
	if product.Images != nil {
		productCopy.Images = append([]models.ProductImage(nil), product.Images...)
	}

	return &productCopy
}

func productIntPtr(v int) *int {
	return &v
}

func productStringPtr(v string) *string {
	return &v
}

func readMultipartRequest(r *http.Request) (map[string]string, string, string, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, "", "", err
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		return nil, "", "", fmt.Errorf("unexpected media type: %s", mediaType)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, "", "", err
	}

	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	fields := make(map[string]string)
	var filename string
	var fileField string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", "", err
		}

		payload, err := io.ReadAll(part)
		if err != nil {
			return nil, "", "", err
		}

		if part.FileName() != "" {
			filename = part.FileName()
			fileField = part.FormName()
			continue
		}

		fields[part.FormName()] = string(payload)
	}

	return fields, filename, fileField, nil
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
