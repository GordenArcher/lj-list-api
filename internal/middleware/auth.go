package middleware

import (
	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthRequired returns a middleware that validates JWT tokens from either:
// 1. The httpOnly "access_token" cookie (preferred for XSS protection)
// 2. The Authorization header with "Bearer <token>" format (for API clients)
//
// The middleware sets user_id and user_role on the context so handlers can
// access the authenticated user's identity. If both token sources are missing
// or the token is invalid, the request is aborted with 401.
func AuthRequired(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Try cookie first (preferred for browser-based clients)
		cookie, err := c.Cookie("access_token")
		if err == nil {
			token = cookie
		} else {
			// Fall back to Authorization header for API clients
			auth := c.GetHeader("Authorization")
			if auth != "" {
				// Extract "Bearer <token>" format
				if len(auth) > 7 && auth[:7] == "Bearer " {
					token = auth[7:]
				}
			}
		}

		if token == "" {
			utils.HandleError(c, apperrors.New(
				apperrors.KindUnauthorized,
				"Authentication required",
				map[string][]string{"auth": {"missing access token"}},
			), "")
			c.Abort()
			return
		}

		// Validate the token using the JWT secret from config
		claims, err := utils.ParseToken(token, cfg.JWTSecret, utils.AccessTokenType)
		if err != nil {
			utils.HandleError(c, apperrors.New(
				apperrors.KindUnauthorized,
				"Invalid or expired access token",
				map[string][]string{"auth": {"invalid token"}},
			), "")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// AdminRequired is a middleware that checks for admin role. It MUST run after
// AuthRequired in the middleware chain, since it depends on "user_role" being
// set in the context. If the user is authenticated but lacks admin role, the
// request is rejected with 403 Forbidden.
func AdminRequired(c *gin.Context) {
	role, exists := c.Get("user_role")
	if !exists || role.(string) != "admin" {
		utils.HandleError(c, apperrors.New(
			apperrors.KindForbidden,
			"Admin access required",
			map[string][]string{"auth": {"admin role required"}},
		), "")
		c.Abort()
		return
	}
	c.Next()
}
