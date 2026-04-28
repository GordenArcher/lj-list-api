package utils

import (
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/gin-gonic/gin"
)

// Success writes a standardized success response. It pulls the request ID
// from the Gin context (set by RequestID middleware) so every response
// carries the same ID that's in the request header and logs.
func Success(c *gin.Context, status int, message string, data any) {
	requestID := GetRequestIDFromContext(c)
	c.JSON(status, models.NewSuccessResponse(requestID, message, data))
}

// Error writes a standardized error response. Field-level validation errors
// go in the errs map, the frontend uses the field names to highlight
// specific inputs. For non-validation errors, pass a single key like "auth"
// or "server".
func Error(c *gin.Context, status int, code, message string, errs map[string][]string) {
	requestID := GetRequestIDFromContext(c)
	c.JSON(status, models.NewErrorResponse(requestID, code, message, errs))
}

// GetRequestIDFromContext is a small helper so handlers don't need to
// import the middleware package just to extract the request ID. This keeps
// handler imports clean.
func GetRequestIDFromContext(c *gin.Context) string {
	id, exists := c.Get("request_id")
	if !exists {
		return "unknown"
	}
	return id.(string)
}

// GetUserIDFromContext extracts the authenticated user's ID from the
// context. Returns empty string if not set (middleware didn't run).
func GetUserIDFromContext(c *gin.Context) string {
	id, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	return id.(string)
}

// GetUserRoleFromContext extracts the authenticated user's role from the
// context. Returns empty string if not set.
func GetUserRoleFromContext(c *gin.Context) string {
	role, exists := c.Get("user_role")
	if !exists {
		return ""
	}
	return role.(string)
}
