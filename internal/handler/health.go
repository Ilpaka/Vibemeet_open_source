package handler

import (
	"net/http"

	"vibemeet/internal/config"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	hostIP      string
	liveKitPort string
}

func NewHealthHandler(cfg *config.Config) *HealthHandler {
	hostIP := cfg.LiveKit.HostIP
	if hostIP == "" {
		hostIP = config.GetLocalIP()
	}

	// Use the port from config, or fall back to the default
	liveKitPort := cfg.LiveKit.Port
	if liveKitPort == "" {
		liveKitPort = "7880"
	}

	return &HealthHandler{
		hostIP:      hostIP,
		liveKitPort: liveKitPort,
	}
}

func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "video-conference",
	})
}

// ServerInfo returns server information for clients.
func (h *HealthHandler) ServerInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"host_ip":       h.hostIP,
		"livekit_port":  h.liveKitPort,
		"livekit_url":   "ws://" + h.hostIP + ":" + h.liveKitPort,
		"api_base":      "/api/v1",
	})
}

