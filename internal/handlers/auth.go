package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	// refreshTokenCookiePath scopes the refresh cookie to auth endpoints only.
	// Using the auth prefix instead of the exact refresh route keeps the cookie
	// off the rest of the API while ensuring login/signup/logout/refresh can all
	// reliably set or clear the same cookie path.
	refreshTokenCookiePath = "/api/v1/auth"
)

type AuthHandler struct {
	authService authService
	cfg         config.Config
}

type authService interface {
	Signup(ctx context.Context, input services.SignupInput) (*models.User, error)
	VerifyOTP(ctx context.Context, phoneNumber, otp string) (*models.User, *utils.TokenPair, error)
	ResendOTP(ctx context.Context, phoneNumber string) error
	Login(ctx context.Context, phoneNumber, password string) (*models.User, *utils.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*utils.TokenPair, error)
}

func NewAuthHandler(authService *services.AuthService, cfg config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
	}
}

type signupRequest struct {
	PhoneNumber     string `json:"phone_number"`
	StaffNumber     string `json:"staff_number"`
	Institution     string `json:"institution"`
	GhanaCardNumber string `json:"ghana_card_number"`
	Password        string `json:"password"`
	DisplayName     string `json:"display_name"`
	Name            string `json:"name"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	errs := make(map[string][]string)
	displayName := resolveDisplayName(req.DisplayName, req.Name)
	phoneNumber := utils.NormalizePhone(req.PhoneNumber)

	if !utils.ValidatePhone(phoneNumber) {
		errs["phone_number"] = []string{"must be a valid phone number"}
	}
	if !utils.ValidateRequired(req.StaffNumber) {
		errs["staff_number"] = []string{"required"}
	}
	if !utils.ValidateRequired(req.Institution) {
		errs["institution"] = []string{"required"}
	}
	if !utils.ValidateRequired(req.GhanaCardNumber) {
		errs["ghana_card_number"] = []string{"required"}
	}
	if !utils.ValidatePassword(req.Password) {
		errs["password"] = []string{"must be at least 8 characters"}
	}
	if !utils.ValidateDisplayName(displayName) {
		errs["display_name"] = []string{"must be between 2 and 100 characters"}
	}

	if len(errs) > 0 {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", errs)
		return
	}

	user, err := h.authService.Signup(c.Request.Context(), services.SignupInput{
		Password:        req.Password,
		DisplayName:     displayName,
		PhoneNumber:     phoneNumber,
		StaffNumber:     strings.TrimSpace(req.StaffNumber),
		Institution:     strings.TrimSpace(req.Institution),
		GhanaCardNumber: strings.TrimSpace(req.GhanaCardNumber),
	})
	if err != nil {
		utils.HandleError(c, err, "Something went wrong")
		return
	}

	utils.Success(c, http.StatusAccepted, "Account created. Activation OTP sent.", gin.H{
		"user": gin.H{
			"id":           user.ID,
			"display_name": user.DisplayName,
			"phone_number": user.PhoneNumber,
			"role":         user.Role,
		},
		"verification": gin.H{
			"phone_number":       phoneNumber,
			"expires_in_minutes": int(services.ActivationOTPExpiry / time.Minute),
		},
	})
}

type verifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	errs := make(map[string][]string)
	phoneNumber := utils.NormalizePhone(req.PhoneNumber)
	otp := strings.TrimSpace(req.OTP)

	if !utils.ValidatePhone(phoneNumber) {
		errs["phone_number"] = []string{"must be a valid phone number"}
	}
	if otp == "" {
		errs["otp"] = []string{"required"}
	}
	if len(errs) > 0 {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", errs)
		return
	}

	user, tokenPair, err := h.authService.VerifyOTP(c.Request.Context(), phoneNumber, otp)
	if err != nil {
		utils.HandleError(c, err, "Failed to verify OTP")
		return
	}

	h.setAuthCookies(c, tokenPair)

	utils.Success(c, http.StatusOK, "Account activated successfully", gin.H{
		"user": authUserPayload(user),
	})
}

type resendOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
}

func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req resendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	phoneNumber := utils.NormalizePhone(req.PhoneNumber)
	if !utils.ValidatePhone(phoneNumber) {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", map[string][]string{
			"phone_number": {"must be a valid phone number"},
		})
		return
	}

	if err := h.authService.ResendOTP(c.Request.Context(), phoneNumber); err != nil {
		utils.HandleError(c, err, "Failed to resend OTP")
		return
	}

	utils.Success(c, http.StatusOK, "Activation OTP sent", gin.H{
		"verification": gin.H{
			"phone_number":       phoneNumber,
			"expires_in_minutes": int(services.ActivationOTPExpiry / time.Minute),
		},
	})
}

type loginRequest struct {
	PhoneNumber string `json:"phone_number"`
	Password    string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	errs := make(map[string][]string)
	phoneNumber := utils.NormalizePhone(req.PhoneNumber)

	if !utils.ValidatePhone(phoneNumber) {
		errs["phone_number"] = []string{"must be a valid phone number"}
	}
	if strings.TrimSpace(req.Password) == "" {
		errs["password"] = []string{"required"}
	}
	if len(errs) > 0 {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", errs)
		return
	}

	user, tokenPair, err := h.authService.Login(c.Request.Context(), phoneNumber, req.Password)
	if err != nil {
		utils.HandleError(c, err, "Failed to login")
		return
	}

	h.setAuthCookies(c, tokenPair)

	utils.Success(c, http.StatusOK, "Login successful", gin.H{
		"user": authUserPayload(user),
	})
}

func resolveDisplayName(displayName, name string) string {
	if trimmed := strings.TrimSpace(displayName); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(name)
}

func authUserPayload(user *models.User) gin.H {
	return gin.H{
		"id":           user.ID,
		"display_name": user.DisplayName,
		"phone_number": user.PhoneNumber,
		"role":         user.Role,
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearAuthCookies(c)
	utils.Success(c, http.StatusOK, "Logged out", nil)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshCookie, err := c.Cookie("refresh_token")
	if err != nil {
		utils.HandleError(c, apperrors.New(apperrors.KindUnauthorized, "Refresh token missing", map[string][]string{
			"auth": {"refresh token missing"},
		}), "")
		return
	}

	// Use the service to validate the refresh token and get new tokens.
	// The service handles fetching the user to ensure role is current.
	tokenPair, err := h.authService.RefreshTokens(c.Request.Context(), refreshCookie)
	if err != nil {
		utils.HandleError(c, err, "Failed to refresh tokens")
		return
	}

	h.setAuthCookies(c, tokenPair)

	utils.Success(c, http.StatusOK, "Tokens refreshed", gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
	})
}

func (h *AuthHandler) setAuthCookies(c *gin.Context, tokenPair *utils.TokenPair) {
	c.SetSameSite(parseSameSiteMode(h.cfg.CookieSameSite))
	c.SetCookie("access_token", tokenPair.AccessToken, int((15 * time.Minute).Seconds()), "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie("refresh_token", tokenPair.RefreshToken, int((7 * 24 * time.Hour).Seconds()), refreshTokenCookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}

func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(parseSameSiteMode(h.cfg.CookieSameSite))
	c.SetCookie("access_token", "", -1, "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie("refresh_token", "", -1, refreshTokenCookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}

func parseSameSiteMode(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
