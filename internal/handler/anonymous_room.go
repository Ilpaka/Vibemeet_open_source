package handler

import (
	"net/http"
	"strings"

	"vibemeet/internal/service"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnonymousRoomHandler struct {
	roomService service.AnonymousRoomService
	log         logger.Logger
}

func NewAnonymousRoomHandler(roomService service.AnonymousRoomService, log logger.Logger) *AnonymousRoomHandler {
	return &AnonymousRoomHandler{
		roomService: roomService,
		log:         log,
	}
}

// normalizeDisplayName normalizes and validates displayName.
func normalizeDisplayName(displayName string) string {
	// Trim surrounding whitespace
	displayName = strings.TrimSpace(displayName)

	// If empty or whitespace-only, fall back to the default value
	if displayName == "" {
		return "User"
	}

	// Cap the length at 50 characters
	const maxLength = 50
	if len(displayName) > maxLength {
		displayName = displayName[:maxLength]
	}

	return displayName
}

type CreateAnonymousRoomRequest struct {
	Title           string  `json:"title"`
	Description     *string `json:"description,omitempty"`
	MaxParticipants int     `json:"max_participants"`
	DisplayName     string  `json:"display_name"`
}

func (h *AnonymousRoomHandler) Create(c *gin.Context) {
	// Get participant_id from context
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

	var req CreateAnonymousRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input and apply default values
	req.DisplayName = normalizeDisplayName(req.DisplayName)
	if req.Title == "" {
		req.Title = "New room"
	}
	if req.MaxParticipants <= 0 {
		req.MaxParticipants = 10
	}

	room, participant, err := h.roomService.Create(
		c.Request.Context(),
		req.Title,
		req.Description,
		req.MaxParticipants,
		participantIDStr,
		req.DisplayName,
	)
	if err != nil {
		h.log.Warn("Failed to create room", "error", err, "participant_id", participantIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return the room together with additional information
	response := gin.H{
		"id":                room.ID,
		"livekit_room_name": room.LiveKitRoomName,
		"title":             room.Title,
		"description":       room.Description,
		"status":            room.Status,
		"max_participants":  room.MaxParticipants,
		"created_at":        room.CreatedAt,
		"updated_at":        room.UpdatedAt,
		"participant":       participant,
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AnonymousRoomHandler) GetByID(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	room, err := h.roomService.GetByID(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

type JoinAnonymousRoomRequest struct {
	DisplayName string `json:"display_name"`
}

func (h *AnonymousRoomHandler) Join(c *gin.Context) {
	// Get participant_id from context
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

	var req JoinAnonymousRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate display_name and apply the default value if missing
	req.DisplayName = normalizeDisplayName(req.DisplayName)

	participant, err := h.roomService.Join(c.Request.Context(), roomID, participantIDStr, req.DisplayName)
	if err != nil {
		h.log.Warn("Failed to join room", "error", err, "room_id", roomID, "participant_id", participantIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, participant)
}

func (h *AnonymousRoomHandler) Leave(c *gin.Context) {
	// Get participant_id from context
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

	if err := h.roomService.Leave(c.Request.Context(), roomID, participantIDStr); err != nil {
		h.log.Warn("Failed to leave room", "error", err, "room_id", roomID, "participant_id", participantIDStr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully left room"})
}

func (h *AnonymousRoomHandler) GetParticipants(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	participants, err := h.roomService.GetParticipants(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, participants)
}
