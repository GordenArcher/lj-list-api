package handlers

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type ApplicationHandler struct {
	applicationService *services.ApplicationService
}

func NewApplicationHandler(applicationService *services.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{applicationService: applicationService}
}

type submitApplicationRequest struct {
	PackageType     string                   `json:"package_type"`
	PackageName     string                   `json:"package_name"`
	CartItems       []services.CartItemInput `json:"cart_items"`
	StaffNumber     string                   `json:"staff_number"`
	MandateNumber   string                   `json:"mandate_number"`
	Institution     string                   `json:"institution"`
	GhanaCardNumber string                   `json:"ghana_card_number"`
}

func (h *ApplicationHandler) Create(c *gin.Context) {
	var req submitApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	errs := make(map[string][]string)

	if req.PackageType != "fixed" && req.PackageType != "custom" {
		errs["package_type"] = []string{"must be 'fixed' or 'custom'"}
	}
	if req.PackageType == "fixed" && req.PackageName == "" {
		errs["package_name"] = []string{"required for fixed packages"}
	}
	if !utils.ValidateRequired(req.StaffNumber) {
		errs["staff_number"] = []string{"required"}
	}
	if !utils.ValidateRequired(req.MandateNumber) {
		errs["mandate_number"] = []string{"required"}
	}
	if !utils.ValidateRequired(req.Institution) {
		errs["institution"] = []string{"required"}
	}
	if !utils.ValidateRequired(req.GhanaCardNumber) {
		errs["ghana_card_number"] = []string{"required"}
	}

	if len(errs) > 0 {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", errs)
		return
	}

	userID := utils.GetUserIDFromContext(c)

	app, err := h.applicationService.Submit(
		c.Request.Context(),
		userID,
		req.PackageType,
		req.PackageName,
		req.CartItems,
		req.StaffNumber,
		req.MandateNumber,
		req.Institution,
		req.GhanaCardNumber,
	)
	if err != nil {
		utils.HandleError(c, err, "Failed to submit application")
		return
	}

	utils.Success(c, http.StatusCreated, "Application submitted successfully", app)
}

func (h *ApplicationHandler) List(c *gin.Context) {
	userID := utils.GetUserIDFromContext(c)

	// Extract pagination parameters from query string.
	// Default: page=1, limit=20. Max limit is 100 to prevent abuse.
	pag := utils.ExtractPaginationParams(c)

	apps, err := h.applicationService.GetByUserID(c.Request.Context(), userID, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve applications")
		return
	}

	// Get the total count for pagination metadata.
	total, err := h.applicationService.GetByUserIDCount(c.Request.Context(), userID)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve application count")
		return
	}

	utils.Success(c, http.StatusOK, "Applications retrieved", gin.H{
		"applications": apps,
		"meta":         utils.BuildPaginationMeta(total, pag),
	})
}

func (h *ApplicationHandler) Get(c *gin.Context) {
	userID := utils.GetUserIDFromContext(c)
	appID := c.Param("id")

	app, err := h.applicationService.GetByID(c.Request.Context(), appID, userID)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve application")
		return
	}

	utils.Success(c, http.StatusOK, "Application retrieved", app)
}
