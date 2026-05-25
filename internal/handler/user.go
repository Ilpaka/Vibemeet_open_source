package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"vibemeet/internal/service"
	"vibemeet/pkg/logger"
)

type UserHandler struct {
	userService service.UserService
	log         logger.Logger
}

func NewUserHandler(userService service.UserService, log logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		log:         log,
	}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	user, err := h.userService.GetMe(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

type UpdateMeRequest struct {
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.UpdateMe(c.Request.Context(), userID.(uuid.UUID), req.DisplayName, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	settings, err := h.userService.GetSettings(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

func (h *UserHandler) UpdateSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req struct {
		PreferredVideoQuality string `json:"preferred_video_quality"`
		PreferredTheme        string `json:"preferred_theme"`
		MuteMicOnJoin         bool   `json:"mute_mic_on_join"`
		DisableCameraOnJoin   bool   `json:"disable_camera_on_join"`
		LanguageCode          string `json:"language_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load the current settings
	currentSettings, err := h.userService.GetSettings(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update only the fields that were supplied
	if req.PreferredVideoQuality != "" {
		currentSettings.PreferredVideoQuality = req.PreferredVideoQuality
	}
	if req.PreferredTheme != "" {
		currentSettings.PreferredTheme = req.PreferredTheme
	}
	currentSettings.MuteMicOnJoin = req.MuteMicOnJoin
	currentSettings.DisableCameraOnJoin = req.DisableCameraOnJoin
	if req.LanguageCode != "" {
		currentSettings.LanguageCode = req.LanguageCode
	}

	if err := h.userService.UpdateSettings(c.Request.Context(), userID.(uuid.UUID), currentSettings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, currentSettings)
}

