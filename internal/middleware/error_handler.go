package middleware

import (
	"github.com/gin-gonic/gin"
	"vibemeet/pkg/errors"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check whether any errors were recorded
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Determine the status code
			statusCode := errors.HTTPStatusFromError(err.Err)

			c.JSON(statusCode, gin.H{
				"error": err.Error(),
			})
		}
	}
}
