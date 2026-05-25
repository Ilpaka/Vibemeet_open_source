package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"vibemeet/internal/service"
	"vibemeet/pkg/logger"
)

type AuthMiddleware struct {
	authService service.AuthService
	log         logger.Logger
}

func NewAuthMiddleware(authService service.AuthService, log logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		log:         log,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		user, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("user_email", user.Email)
		c.Set("user_role", user.GlobalRole)
		c.Next()
	}
}

