package handlers

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categoryService *services.CategoryService
}

func NewCategoryHandler(categoryService *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

type categoryRequest struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sort_order,omitempty"`
	Active    *bool  `json:"active,omitempty"`
}

func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categoryService.List(c.Request.Context(), true)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve categories")
		return
	}

	utils.Success(c, http.StatusOK, "Categories retrieved", gin.H{"categories": categories})
}

func (h *CategoryHandler) Get(c *gin.Context) {
	cat, err := h.categoryService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve category")
		return
	}

	utils.Success(c, http.StatusOK, "Category retrieved", cat)
}

func (h *CategoryHandler) PublicList(c *gin.Context) {
	categories, err := h.categoryService.List(c.Request.Context(), false)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve categories")
		return
	}

	utils.Success(c, http.StatusOK, "Categories retrieved", gin.H{"categories": categories})
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	cat, err := h.categoryService.Create(c.Request.Context(), services.CreateCategoryInput{
		Name:      req.Name,
		SortOrder: req.SortOrder,
		Active:    req.Active,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to create category")
		return
	}

	utils.Success(c, http.StatusCreated, "Category created successfully", cat)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	cat, err := h.categoryService.Update(c.Request.Context(), c.Param("id"), services.UpdateCategoryInput{
		Name:      &req.Name,
		SortOrder: req.SortOrder,
		Active:    req.Active,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to update category")
		return
	}

	utils.Success(c, http.StatusOK, "Category updated successfully", cat)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	cat, softDeleted, err := h.categoryService.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to delete category")
		return
	}

	message := "Category deactivated successfully"
	if softDeleted {
		message = "Category had products, so it was deactivated instead of being deleted"
	}

	utils.Success(c, http.StatusOK, message, gin.H{
		"category":     cat,
		"deactivated":  true,
		"soft_deleted": softDeleted,
		"message":      message,
	})
}
