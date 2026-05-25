package handler

import (
	"net/http"

	"vibemeet/internal/service"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MediaHandler struct {
	mediaService service.MediaService
	log          logger.Logger
}

func NewMediaHandler(mediaService service.MediaService, log logger.Logger) *MediaHandler {
	return &MediaHandler{
		mediaService: mediaService,
		log:          log,
	}
}

type GetTokenRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

func (h *MediaHandler) GetToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	var req GetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, url, err := h.mediaService.GetToken(c.Request.Context(), roomID, userID.(uuid.UUID), req.DisplayName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "url": url})
}
