package handlers

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type PackageHandler struct {
	packageService *services.PackageService
}

func NewPackageHandler(packageService *services.PackageService) *PackageHandler {
	return &PackageHandler{packageService: packageService}
}

type fixedPackageRequest struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Tagline     string               `json:"tagline"`
	Price       string               `json:"price"`
	Monthly     string               `json:"monthly"`
	Tag         string               `json:"tag"`
	Popular     bool                 `json:"popular"`
	RiceOptions string               `json:"rice_options"`
	Items       []models.PackageItem `json:"items"`
}

type simplePackageRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
	Items string `json:"items"`
}

func (h *PackageHandler) List(c *gin.Context) {
	catalog, err := h.packageService.GetCatalog(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve package catalog")
		return
	}

	utils.Success(c, http.StatusOK, "Package catalog retrieved", catalog)
}

func (h *PackageHandler) AdminList(c *gin.Context) {
	catalog, err := h.packageService.GetAdminCatalog(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve package catalog")
		return
	}

	utils.Success(c, http.StatusOK, "Package catalog retrieved", catalog)
}

func (h *PackageHandler) Fixed(c *gin.Context) {
	packages, err := h.packageService.GetFixedPackages(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve fixed packages")
		return
	}

	utils.Success(c, http.StatusOK, "Fixed packages retrieved", gin.H{
		"fixed_packages": packages,
	})
}

func (h *PackageHandler) FixedAdmin(c *gin.Context) {
	packages, err := h.packageService.GetFixedPackagesAdmin(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve fixed packages")
		return
	}

	utils.Success(c, http.StatusOK, "Fixed packages retrieved", gin.H{
		"fixed_packages": packages,
	})
}

func (h *PackageHandler) FixedAdminByID(c *gin.Context) {
	pkg, err := h.packageService.GetFixedPackageAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve fixed package")
		return
	}

	utils.Success(c, http.StatusOK, "Fixed package retrieved", pkg)
}

func (h *PackageHandler) Provisions(c *gin.Context) {
	utils.Success(c, http.StatusOK, "Provisions packages retrieved", gin.H{
		"provisions_packages": h.packageService.GetProvisionsPackages(c.Request.Context()),
	})
}

func (h *PackageHandler) ProvisionsAdmin(c *gin.Context) {
	utils.Success(c, http.StatusOK, "Provisions packages retrieved", gin.H{
		"provisions_packages": h.packageService.GetProvisionsPackagesAdmin(c.Request.Context()),
	})
}

func (h *PackageHandler) ProvisionsAdminByID(c *gin.Context) {
	pkg, err := h.packageService.GetDepartmentPackage(c.Request.Context(), "provisions", c.Param("id"), true)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve provisions package")
		return
	}

	utils.Success(c, http.StatusOK, "Provisions package retrieved", pkg)
}

func (h *PackageHandler) Detergents(c *gin.Context) {
	utils.Success(c, http.StatusOK, "Detergent packages retrieved", gin.H{
		"detergent_packages": h.packageService.GetDetergentPackages(c.Request.Context()),
	})
}

func (h *PackageHandler) DetergentsAdmin(c *gin.Context) {
	utils.Success(c, http.StatusOK, "Detergent packages retrieved", gin.H{
		"detergent_packages": h.packageService.GetDetergentPackagesAdmin(c.Request.Context()),
	})
}

func (h *PackageHandler) DetergentsAdminByID(c *gin.Context) {
	pkg, err := h.packageService.GetDepartmentPackage(c.Request.Context(), "detergents", c.Param("id"), true)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve detergent package")
		return
	}

	utils.Success(c, http.StatusOK, "Detergent package retrieved", pkg)
}

func (h *PackageHandler) CreateFixed(c *gin.Context) {
	var req fixedPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	pkg, err := h.packageService.CreateFixedPackage(c.Request.Context(), models.FixedPackage{
		ID:          req.ID,
		Name:        req.Name,
		Tagline:     req.Tagline,
		Price:       req.Price,
		Monthly:     req.Monthly,
		Tag:         req.Tag,
		Popular:     req.Popular,
		RiceOptions: req.RiceOptions,
		Items:       req.Items,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to create fixed package")
		return
	}

	utils.Success(c, http.StatusCreated, "Fixed package created successfully", pkg)
}

func (h *PackageHandler) UpdateFixed(c *gin.Context) {
	var req fixedPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	pkg, err := h.packageService.UpdateFixedPackage(c.Request.Context(), c.Param("id"), models.FixedPackage{
		ID:          req.ID,
		Name:        req.Name,
		Tagline:     req.Tagline,
		Price:       req.Price,
		Monthly:     req.Monthly,
		Tag:         req.Tag,
		Popular:     req.Popular,
		RiceOptions: req.RiceOptions,
		Items:       req.Items,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to update fixed package")
		return
	}

	utils.Success(c, http.StatusOK, "Fixed package updated successfully", pkg)
}

func (h *PackageHandler) DeleteFixed(c *gin.Context) {
	if err := h.packageService.DeleteFixedPackage(c.Request.Context(), c.Param("id")); err != nil {
		utils.HandleError(c, err, "Failed to delete fixed package")
		return
	}

	utils.Success(c, http.StatusOK, "Fixed package deactivated successfully", nil)
}

func (h *PackageHandler) CreateProvisions(c *gin.Context) {
	h.createDepartmentPackage(c, "provisions")
}

func (h *PackageHandler) UpdateProvisions(c *gin.Context) {
	h.updateDepartmentPackage(c, "provisions")
}

func (h *PackageHandler) DeleteProvisions(c *gin.Context) {
	if err := h.packageService.DeleteDepartmentPackage(c.Request.Context(), c.Param("id")); err != nil {
		utils.HandleError(c, err, "Failed to delete provisions package")
		return
	}

	utils.Success(c, http.StatusOK, "Provisions package deactivated successfully", nil)
}

func (h *PackageHandler) CreateDetergents(c *gin.Context) {
	h.createDepartmentPackage(c, "detergents")
}

func (h *PackageHandler) UpdateDetergents(c *gin.Context) {
	h.updateDepartmentPackage(c, "detergents")
}

func (h *PackageHandler) DeleteDetergents(c *gin.Context) {
	if err := h.packageService.DeleteDepartmentPackage(c.Request.Context(), c.Param("id")); err != nil {
		utils.HandleError(c, err, "Failed to delete detergent package")
		return
	}

	utils.Success(c, http.StatusOK, "Detergent package deactivated successfully", nil)
}

func (h *PackageHandler) createDepartmentPackage(c *gin.Context, kind string) {
	var req simplePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	pkg, err := h.packageService.CreateDepartmentPackage(c.Request.Context(), kind, models.SimplePackage{
		ID:    req.ID,
		Name:  req.Name,
		Price: req.Price,
		Items: req.Items,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to create package")
		return
	}

	utils.Success(c, http.StatusCreated, "Package created successfully", pkg)
}

func (h *PackageHandler) updateDepartmentPackage(c *gin.Context, kind string) {
	var req simplePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	pkg, err := h.packageService.UpdateDepartmentPackage(c.Request.Context(), kind, c.Param("id"), models.SimplePackage{
		ID:    req.ID,
		Name:  req.Name,
		Price: req.Price,
		Items: req.Items,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to update package")
		return
	}

	utils.Success(c, http.StatusOK, "Package updated successfully", pkg)
}
