package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
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
	authService *services.AuthService
	cfg         config.Config
}

func NewAuthHandler(authService *services.AuthService, cfg config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
	}
}

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
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

	if !utils.ValidateEmail(req.Email) {
		errs["email"] = []string{"valid email is required"}
	}
	if !utils.ValidatePassword(req.Password) {
		errs["password"] = []string{"must be at least 8 characters"}
	}
	if !utils.ValidateDisplayName(req.DisplayName) {
		errs["display_name"] = []string{"must be between 2 and 100 characters"}
	}

	if len(errs) > 0 {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", errs)
		return
	}

	user, tokenPair, err := h.authService.Signup(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		utils.HandleError(c, err, "Something went wrong")
		return
	}

	h.setAuthCookies(c, tokenPair)

	utils.Success(c, http.StatusCreated, "Account created successfully", gin.H{
		"user": gin.H{
			"id":           user.ID,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"phone":        user.Phone,
			"role":         user.Role,
		},
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	if !utils.ValidateEmail(req.Email) {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Valid email is required", map[string][]string{
			"email": {"valid email is required"},
		})
		return
	}

	user, tokenPair, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		utils.HandleError(c, err, "Failed to login")
		return
	}

	h.setAuthCookies(c, tokenPair)

	utils.Success(c, http.StatusOK, "Login successful", gin.H{
		"user": gin.H{
			"id":           user.ID,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"phone":        user.Phone,
			"role":         user.Role,
		},
	})
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
