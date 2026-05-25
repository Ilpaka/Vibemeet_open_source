package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"vibemeet/internal/service"
	"vibemeet/pkg/logger"
)

type StatsHandler struct {
	statsService service.StatsService
	log          logger.Logger
}

func NewStatsHandler(statsService service.StatsService, log logger.Logger) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
		log:          log,
	}
}

func (h *StatsHandler) GetRoomStats(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room ID"})
		return
	}

	stats, err := h.statsService.GetRoomStats(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *StatsHandler) GetParticipantStats(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	stats, err := h.statsService.GetParticipantStats(c.Request.Context(), participantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

