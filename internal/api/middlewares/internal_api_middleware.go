package middleware

import (
	"net/http"

	"github.com/dione-docs-backend/internal/config"
	"github.com/gin-gonic/gin"
)

func InternalAPIMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Internal-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}

		if apiKey != cfg.InternalApiKey {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid API key"})
			return
		}

		c.Next()
	}
}
