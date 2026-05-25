package handler

import (
	"net/http"
	"strings"

	"vibemeet/internal/service"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
	log         logger.Logger
}

func NewAuthHandler(authService service.AuthService, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		log:         log,
	}
}

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("Invalid registration request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		// Choose the status code based on the error type
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}
		h.log.Warn("Registration failed", "error", err, "email", req.Email)
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
	c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("Invalid login request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	response, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.log.Warn("Login failed", "error", err, "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("User logged in successfully", "user_id", response.User.ID, "email", response.User.Email)
	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

