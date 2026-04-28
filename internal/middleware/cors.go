package middleware

import (
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

// CORS configures cross-origin resource sharing. Unlike naive implementations
// that echo back any Origin header (security risk), this validates the incoming
// origin against a whitelist. In production, specify your frontend domain(s) in
// the AllowedOrigins config. Development uses permissive settings for localhost.
func CORS(cfg config.Config) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		// Get the origin from the request header
		origin := c.GetHeader("Origin")

		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				utils.Error(c, 403, "FORBIDDEN", "Origin is not allowed", map[string][]string{
					"origin": {"not allowed"},
				})
				c.Abort()
				return
			}

			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
