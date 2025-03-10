package middleware

import (
	"net/http"

	"github.com/dione-docs-backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequireRole(repo repository.PermissionRepository, requiredRole string, docIDKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		userID, err := uuid.Parse(userIDStr.(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		docIDStr := c.Param(docIDKey)
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
			return
		}

		perm, err := repo.GetByDocumentAndUser(userID, docID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}

		if perm.AccessType != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}

		c.Next()
	}
}
