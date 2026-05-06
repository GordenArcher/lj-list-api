package handlers

import (
	"net/http"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productService *services.ProductService
}

func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

type createProductRequest struct {
	Name       string `json:"name"`
	CategoryID string `json:"category_id"`
	Price      int    `json:"price"`
	OldPrice   *int   `json:"old_price,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Unit       string `json:"unit"`
	Active     *bool  `json:"active,omitempty"`
}

type updateProductRequest struct {
	Name       *string `json:"name,omitempty"`
	CategoryID *string `json:"category_id,omitempty"`
	Price      *int    `json:"price,omitempty"`
	OldPrice   *int    `json:"old_price,omitempty"`
	Tag        *string `json:"tag,omitempty"`
	Unit       *string `json:"unit,omitempty"`
	Active     *bool   `json:"active,omitempty"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	product, err := h.productService.CreateProduct(c.Request.Context(), services.CreateProductInput{
		Name:       req.Name,
		CategoryID: req.CategoryID,
		Price:      req.Price,
		OldPrice:   req.OldPrice,
		Tag:        req.Tag,
		Unit:       req.Unit,
		Active:     req.Active,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to create product")
		return
	}

	utils.Success(c, http.StatusCreated, "Product created successfully", product)
}

func (h *ProductHandler) Update(c *gin.Context) {
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	product, err := h.productService.UpdateProduct(c.Request.Context(), c.Param("id"), services.UpdateProductInput{
		Name:       req.Name,
		CategoryID: req.CategoryID,
		Price:      req.Price,
		OldPrice:   req.OldPrice,
		Tag:        req.Tag,
		Unit:       req.Unit,
		Active:     req.Active,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to update product")
		return
	}

	utils.Success(c, http.StatusOK, "Product updated successfully", product)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	result, err := h.productService.DeleteProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to delete product")
		return
	}

	utils.Success(c, http.StatusOK, result.Message, gin.H{
		"product":     result.Product,
		"deactivated": result.SoftDeleted,
		"message":     result.Message,
	})
}

func (h *ProductHandler) AddImages(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse form data", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	uploads := make([]services.ProductImageUploadInput, 0)
	if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
		for _, fieldName := range []string{"images", "image"} {
			for _, fileHeader := range c.Request.MultipartForm.File[fieldName] {
				file, err := fileHeader.Open()
				if err != nil {
					utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to read uploaded image", map[string][]string{
						"images": {"unable to open one or more uploaded files"},
					})
					return
				}
				defer file.Close()

				uploads = append(uploads, services.ProductImageUploadInput{
					Image:            file,
					ImageFilename:    fileHeader.Filename,
					ImageContentType: fileHeader.Header.Get("Content-Type"),
				})
			}
		}
	}

	images, err := h.productService.AddProductImages(c.Request.Context(), c.Param("id"), uploads)
	if err != nil {
		utils.HandleError(c, err, "Failed to add product images")
		return
	}

	utils.Success(c, http.StatusCreated, "Product images added successfully", gin.H{
		"images": images,
	})
}

func (h *ProductHandler) ListImages(c *gin.Context) {
	images, err := h.productService.GetProductImages(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve product images")
		return
	}

	utils.Success(c, http.StatusOK, "Product images retrieved", gin.H{
		"images": images,
	})
}

func (h *ProductHandler) DeleteImage(c *gin.Context) {
	if err := h.productService.DeleteProductImage(c.Request.Context(), c.Param("id"), c.Param("imageId")); err != nil {
		utils.HandleError(c, err, "Failed to delete product image")
		return
	}

	utils.Success(c, http.StatusOK, "Product image deleted", nil)
}

func (h *ProductHandler) List(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))

	// Extract pagination parameters from query string.
	// Default: page=1, limit=20. Max limit is 100 to prevent abuse.
	pag := utils.ExtractPaginationParams(c)

	products, err := h.productService.GetProducts(c.Request.Context(), category, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve products")
		return
	}

	// Get the total count for pagination metadata.
	total, err := h.productService.GetProductsCount(c.Request.Context(), category)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve product count")
		return
	}

	utils.Success(c, http.StatusOK, "Products retrieved", gin.H{
		"products": products,
		"meta":     utils.BuildPaginationMeta(total, pag),
	})
}

func (h *ProductHandler) Get(c *gin.Context) {
	product, err := h.productService.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve product")
		return
	}

	utils.Success(c, http.StatusOK, "Product retrieved", product)
}

func (h *ProductHandler) AdminList(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))

	pag := utils.ExtractPaginationParams(c)

	products, err := h.productService.GetAdminProducts(c.Request.Context(), category, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve products")
		return
	}

	total, err := h.productService.GetAdminProductsCount(c.Request.Context(), category)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve product count")
		return
	}

	utils.Success(c, http.StatusOK, "Products retrieved", gin.H{
		"products": products,
		"meta":     utils.BuildPaginationMeta(total, pag),
	})
}

func (h *ProductHandler) AdminGet(c *gin.Context) {
	product, err := h.productService.GetProductAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve product")
		return
	}

	utils.Success(c, http.StatusOK, "Product retrieved", product)
}

func (h *ProductHandler) Categories(c *gin.Context) {
	categories, err := h.productService.GetCategories(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve categories")
		return
	}

	utils.Success(c, http.StatusOK, "Categories retrieved", gin.H{
		"categories": categories,
	})
}
