package handler

import (
	"net/http"

	"vibemeet/internal/service"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnonymousMediaHandler struct {
	mediaService service.AnonymousMediaService
	log          logger.Logger
}

func NewAnonymousMediaHandler(mediaService service.AnonymousMediaService, log logger.Logger) *AnonymousMediaHandler {
	return &AnonymousMediaHandler{
		mediaService: mediaService,
		log:          log,
	}
}

type GetAnonymousTokenRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

func (h *AnonymousMediaHandler) GetToken(c *gin.Context) {
	// Get participant_id from context (added by ParticipantMiddleware)
	participantID, exists := c.Get("participant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "participant_id is required"})
		return
	}

	participantIDStr, ok := participantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}

	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	var req GetAnonymousTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, url, err := h.mediaService.GetToken(c.Request.Context(), roomID, participantIDStr, req.DisplayName)
	if err != nil {
		h.log.Warn("Failed to get token", "error", err, "room_id", roomID, "participant_id", participantIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "url": url})
}

