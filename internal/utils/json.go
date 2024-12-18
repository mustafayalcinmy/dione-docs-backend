package utils

import (
	"log"

	"github.com/gin-gonic/gin"
)

func RespondWithError(c *gin.Context, code int, msg string) {
	if code > 499 {
		log.Printf("Responding with 5XX error: %v\n", msg)
	}

	// Respond with the error message as JSON
	c.JSON(code, gin.H{"error": msg})
}
