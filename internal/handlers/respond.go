package handlers

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse sends a standardized error response and logs at caller if needed
func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
