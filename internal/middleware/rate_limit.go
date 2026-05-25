package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"vibemeet/internal/service"
	"vibemeet/pkg/logger"
)

type RateLimitMiddleware struct {
	rateLimitService service.RateLimitService
	log              logger.Logger
}

func NewRateLimitMiddleware(rateLimitService service.RateLimitService, log logger.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		rateLimitService: rateLimitService,
		log:              log,
	}
}

func (m *RateLimitMiddleware) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		limit := 100
		window := 60 // seconds

		allowed, err := m.rateLimitService.CheckLimit(c.Request.Context(), key, limit, window)
		if err != nil {
			m.log.Error("Rate limit check failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		if !allowed {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		count, err := m.rateLimitService.Increment(c.Request.Context(), key, window)
		if err != nil {
			m.log.Error("Rate limit increment failed", "error", err)
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limit-int(count)))
		c.Next()
	}
}
