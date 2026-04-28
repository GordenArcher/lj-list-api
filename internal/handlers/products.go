package handlers

import (
	"net/http"

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

func (h *ProductHandler) List(c *gin.Context) {
	category := c.Query("category")

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
