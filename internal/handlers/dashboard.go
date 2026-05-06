package handlers

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboardService *services.DashboardService
}

func NewDashboardHandler(dashboardService *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.dashboardService.GetStats(
		c.Request.Context(),
		c.Query("range"),
		c.Query("from"),
		c.Query("to"),
	)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve dashboard stats")
		return
	}

	utils.Success(c, http.StatusOK, "Dashboard stats retrieved", stats)
}
