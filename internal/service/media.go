package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"vibemeet/internal/config"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
)

type MediaService interface {
	GetToken(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, displayName string) (string, string, error)
}

type mediaService struct {
	roomRepo repository.RoomRepository
	cfg      config.LiveKitConfig
	log      logger.Logger
}

func NewMediaService(roomRepo repository.RoomRepository, cfg config.LiveKitConfig, log logger.Logger) MediaService {
	return &mediaService{
		roomRepo: roomRepo,
		cfg:      cfg,
		log:      log,
	}
}

func (s *mediaService) GetToken(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, displayName string) (string, string, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return "", "", errors.New("room not found")
	}

	at := auth.NewAccessToken(s.cfg.APIKey, s.cfg.APISecret)
	canPublish := true
	canSubscribe := true
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         room.LiveKitRoomName,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}

	at.AddGrant(grant).
		SetIdentity(userID.String()).
		SetName(displayName).
		SetValidFor(time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		s.log.Error("Failed to generate LiveKit token", "error", err)
		return "", "", errors.New("failed to generate token")
	}

	// Use FrontendURL if set, otherwise fall back to URL
	url := s.cfg.FrontendURL
	if url == "" {
		url = s.cfg.URL
	}

	// If URL is still empty, fall back to the default
	if url == "" {
		url = "ws://localhost:7880"
		s.log.Warn("LiveKit URL not configured, using default", "url", url)
	}

	// Convert the internal Docker URL to a public one for the browser
	// ws://livekit:7880 -> ws://localhost:7880
	// Also handle a few different input formats
	if strings.Contains(url, "livekit:7880") {
		// Preserve the protocol (ws/wss)
		if strings.HasPrefix(url, "wss://") {
			url = strings.Replace(url, "livekit:7880", "localhost:7880", 1)
		} else if strings.HasPrefix(url, "https://") {
			url = strings.Replace(url, "https://livekit:7880", "wss://localhost:7880", 1)
		} else if strings.HasPrefix(url, "http://") {
			url = strings.Replace(url, "http://livekit:7880", "ws://localhost:7880", 1)
		} else {
			url = strings.Replace(url, "livekit:7880", "localhost:7880", 1)
			// If no protocol is present, prepend ws://
			if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
				url = "ws://" + url
			}
		}
	}

	// Make sure the URL uses a valid scheme
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		url = "ws://" + url
	}

	// Log for debugging
	s.log.Info("LiveKit URL resolved for frontend", "url", url, "original", s.cfg.URL, "frontend", s.cfg.FrontendURL)

	return token, url, nil
}
