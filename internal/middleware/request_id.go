package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID generates a UUID for every incoming request and attaches it to
// the response header and Gin context. Handlers and services pull it from
// the context for logging and the response envelope. The header lets the
// frontend trace a specific request when reporting issues, they can send
// us the X-Request-ID and we grep the logs for it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the client already sent a request ID (e.g., from a
		// mobile app that generates its own). If not, create one.
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
