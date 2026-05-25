package handler

import (
	"net/http"
	"strconv"
	"time"

	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnonymousChatHandler struct {
	chatRepo repository.AnonymousChatRepository
	roomRepo repository.AnonymousRoomRepository
	log      logger.Logger
}

func NewAnonymousChatHandler(
	chatRepo repository.AnonymousChatRepository,
	roomRepo repository.AnonymousRoomRepository,
	log logger.Logger,
) *AnonymousChatHandler {
	return &AnonymousChatHandler{
		chatRepo: chatRepo,
		roomRepo: roomRepo,
		log:      log,
	}
}

type AnonymousSendMessageRequest struct {
	Content     string `json:"content" binding:"required"`
	DisplayName string `json:"display_name"`
}

func (h *AnonymousChatHandler) SendMessage(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	// Get participant_id from context (set by ParticipantMiddleware)
	participantID, exists := c.Get("participant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "participant_id is required"})
		return
	}

	participantIDStr, ok := participantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}

	// Verify the room exists via AnonymousRoomRepository (PostgreSQL)
	room, err := h.roomRepo.GetByID(c.Request.Context(), roomID)
	if err != nil {
		h.log.Warn("Room not found for chat", "roomID", roomID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	var req AnonymousSendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Take display_name from the request or fall back to a default
	displayName := req.DisplayName
	if displayName == "" {
		displayName = "User"
	}

	// Build the message
	message := &domain.AnonymousChatMessage{
		ID:            uuid.New().String(),
		RoomID:        roomID,
		ParticipantID: participantIDStr,
		DisplayName:   displayName,
		Content:       req.Content,
		CreatedAt:     time.Now(),
	}

	if err := h.chatRepo.SaveMessage(c.Request.Context(), roomID, message); err != nil {
		h.log.Error("Failed to save message", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save message"})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *AnonymousChatHandler) GetMessages(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	// Verify the room exists via AnonymousRoomRepository (PostgreSQL)
	room, err := h.roomRepo.GetByID(c.Request.Context(), roomID)
	if err != nil || room == nil {
		// Room not found - return an empty list instead of an error
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := h.chatRepo.GetMessages(c.Request.Context(), roomID, limit)
	if err != nil {
		h.log.Error("Failed to get messages", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *AnonymousChatHandler) DeleteMessage(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message ID is required"})
		return
	}

	// Get participant_id
	participantID, exists := c.Get("participant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "participant_id is required"})
		return
	}

	participantIDStr, _ := participantID.(string)

	// Load messages so we can verify the author
	messages, err := h.chatRepo.GetMessages(c.Request.Context(), roomID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Make sure the message belongs to this participant
	var found bool
	for _, msg := range messages {
		if msg.ID == messageID {
			if msg.ParticipantID != participantIDStr {
				c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own messages"})
				return
			}
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	if err := h.chatRepo.DeleteMessage(c.Request.Context(), roomID, messageID); err != nil {
		h.log.Error("Failed to delete message", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

