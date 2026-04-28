package handlers

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	applicationService *services.ApplicationService
}

func NewAdminHandler(applicationService *services.ApplicationService) *AdminHandler {
	return &AdminHandler{applicationService: applicationService}
}

func (h *AdminHandler) ListApplications(c *gin.Context) {
	status := c.Query("status")
	pag := utils.ExtractPaginationParams(c)

	apps, err := h.applicationService.GetAll(c.Request.Context(), status, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve applications")
		return
	}

	total, err := h.applicationService.GetAllCount(c.Request.Context(), status)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve application count")
		return
	}

	utils.Success(c, http.StatusOK, "Applications retrieved", gin.H{
		"applications": apps,
		"meta":         utils.BuildPaginationMeta(total, pag),
	})
}

type updateApplicationRequest struct {
	Status string `json:"status"`
}

func (h *AdminHandler) UpdateApplication(c *gin.Context) {
	var req updateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	appID := c.Param("id")
	app, err := h.applicationService.UpdateStatus(c.Request.Context(), appID, req.Status)
	if err != nil {
		utils.HandleError(c, err, "Failed to update application")
		return
	}

	utils.Success(c, http.StatusOK, "Application updated", app)
}
