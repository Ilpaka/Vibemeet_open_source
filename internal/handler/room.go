package handler

import (
	"net/http"

	"vibemeet/internal/service"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RoomHandler struct {
	roomService service.RoomService
	log         logger.Logger
}

func NewRoomHandler(roomService service.RoomService, log logger.Logger) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
		log:         log,
	}
}

type CreateRoomRequest struct {
	Title           string  `json:"title" binding:"required"`
	Description     *string `json:"description,omitempty"`
	MaxParticipants int     `json:"max_participants"`
}

func (h *RoomHandler) Create(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room, err := h.roomService.Create(c.Request.Context(), userID.(uuid.UUID), req.Title, req.Description, req.MaxParticipants)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")
	rooms, err := h.roomService.List(c.Request.Context(), userID.(uuid.UUID), 20, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

func (h *RoomHandler) GetByID(c *gin.Context) {
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

type UpdateRoomRequest struct {
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	MaxParticipants *int    `json:"max_participants,omitempty"`
}

func (h *RoomHandler) Update(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	var req UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room, err := h.roomService.Update(c.Request.Context(), roomID, userID.(uuid.UUID), req.Title, req.Description, req.MaxParticipants)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *RoomHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	if err := h.roomService.Delete(c.Request.Context(), roomID, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Room deleted"})
}

type JoinRoomRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

func (h *RoomHandler) Join(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	var req JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participant, err := h.roomService.Join(c.Request.Context(), roomID, userID.(uuid.UUID), req.DisplayName)
	if err != nil {
		if err.Error() == "waiting for approval" {
			c.JSON(http.StatusAccepted, gin.H{"message": "Waiting for approval"})
			return
		}
		// Check if room not found - return 404
		if err.Error() == "room not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, participant)
}

func (h *RoomHandler) Leave(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	if err := h.roomService.Leave(c.Request.Context(), roomID, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left room"})
}

func (h *RoomHandler) CreateInvite(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	// TODO: add parameters for invite
	invite, err := h.roomService.CreateInvite(c.Request.Context(), roomID, userID.(uuid.UUID), nil, nil, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invite)
}

func (h *RoomHandler) GetParticipants(c *gin.Context) {
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
