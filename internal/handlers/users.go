package handlers

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type updateProfileRequest struct {
	DisplayName     *string `json:"display_name,omitempty"`
	PhoneNumber     *string `json:"phone_number,omitempty"`
	StaffNumber     *string `json:"staff_number,omitempty"`
	Institution     *string `json:"institution,omitempty"`
	GhanaCardNumber *string `json:"ghana_card_number,omitempty"`
	Password        *string `json:"password,omitempty"`
}

type adminUpdateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	Role        *string `json:"role,omitempty"`
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	user, err := h.userService.GetProfile(c.Request.Context(), utils.GetUserIDFromContext(c))
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve profile")
		return
	}

	utils.Success(c, http.StatusOK, "Profile retrieved", gin.H{
		"user": profileUserPayload(user),
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	user, err := h.userService.UpdateProfile(c.Request.Context(), utils.GetUserIDFromContext(c), services.UpdateProfileInput{
		DisplayName:     req.DisplayName,
		PhoneNumber:     req.PhoneNumber,
		StaffNumber:     req.StaffNumber,
		Institution:     req.Institution,
		GhanaCardNumber: req.GhanaCardNumber,
		Password:        req.Password,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to update profile")
		return
	}

	utils.Success(c, http.StatusOK, "Profile updated", gin.H{
		"user": profileUserPayload(user),
	})
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	pag := utils.ExtractPaginationParams(c)

	users, err := h.userService.ListUsers(c.Request.Context(), role, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve users")
		return
	}

	total, err := h.userService.CountUsers(c.Request.Context(), role)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve user count")
		return
	}

	utils.Success(c, http.StatusOK, "Users retrieved", gin.H{
		"users": users,
		"meta":  utils.BuildPaginationMeta(total, pag),
	})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req adminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	user, err := h.userService.AdminUpdateUser(c.Request.Context(), c.Param("id"), services.AdminUpdateUserInput{
		DisplayName: req.DisplayName,
		PhoneNumber: req.PhoneNumber,
		Role:        req.Role,
	})
	if err != nil {
		utils.HandleError(c, err, "Failed to update user")
		return
	}

	utils.Success(c, http.StatusOK, "User updated", gin.H{
		"user": user,
	})
}

func profileUserPayload(user *models.User) gin.H {
	return gin.H{
		"id":                user.ID,
		"display_name":      user.DisplayName,
		"phone_number":      user.PhoneNumber,
		"staff_number":      user.StaffNumber,
		"institution":       user.Institution,
		"ghana_card_number": user.GhanaCardNumber,
		"role":              user.Role,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
	}
}
