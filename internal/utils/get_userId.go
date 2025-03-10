package utils

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, errors.New("kullanıcı kimliği bulunamadı")
	}

	userID, ok := userIDStr.(string)
	if !ok {
		return uuid.Nil, errors.New("geçersiz kullanıcı kimliği formatı")
	}

	return uuid.Parse(userID)
}
