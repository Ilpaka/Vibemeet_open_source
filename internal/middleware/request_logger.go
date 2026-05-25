package middleware

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		fmt.Fprintf(gin.DefaultWriter.(io.Writer), "[%s] %s %s %d %s\n",
			clientIP,
			method,
			path,
			statusCode,
			latency,
		)
	}
}

