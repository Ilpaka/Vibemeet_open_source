package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"vibemeet/internal/config"
	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
)

type AnonymousMediaService interface {
	GetToken(ctx context.Context, roomID uuid.UUID, participantID string, displayName string) (string, string, error)
}

type anonymousMediaService struct {
	roomRepo repository.AnonymousRoomRepository
	cfg      config.LiveKitConfig
	log      logger.Logger
}

func NewAnonymousMediaService(roomRepo repository.AnonymousRoomRepository, cfg config.LiveKitConfig, log logger.Logger) AnonymousMediaService {
	return &anonymousMediaService{
		roomRepo: roomRepo,
		cfg:      cfg,
		log:      log,
	}
}

func (s *anonymousMediaService) GetToken(ctx context.Context, roomID uuid.UUID, participantID string, displayName string) (string, string, error) {
	// Check that the room exists
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		// If the room doesn't exist, create it automatically (temporary workaround for testing)
		s.log.Info("Room not found, creating automatically", "room_id", roomID, "participant_id", participantID)
		now := time.Now()
		room = &domain.AnonymousRoom{
			ID:              roomID,
			LiveKitRoomName: roomID.String(),
			Title:           "Auto-created room",
			Status:          domain.RoomStatusActive,
			MaxParticipants: 10,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if createErr := s.roomRepo.Create(ctx, room); createErr != nil {
			s.log.Error("Failed to auto-create room", "error", createErr)
			return "", "", errors.New("room not found and failed to create")
		}
	}

	// Verify the room is active (using the constant from domain)
	if room.Status != domain.RoomStatusActive {
		return "", "", errors.New("room is not active")
	}

	// Create a LiveKit access token
	at := auth.NewAccessToken(s.cfg.APIKey, s.cfg.APISecret)
	canPublish := true
	canSubscribe := true
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         room.LiveKitRoomName,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}

	// Use participant_id as identity (must be a valid UUID)
	// If participantID is not a valid UUID, use it as-is (LiveKit accepts strings)
	identity := participantID
	if _, err := uuid.Parse(participantID); err != nil {
		// If not a UUID, derive one from participantID for consistency
		identity = uuid.NewSHA1(uuid.NameSpaceOID, []byte(participantID)).String()
	}

	at.AddGrant(grant).
		SetIdentity(identity).
		SetName(displayName).
		SetValidFor(time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		s.log.Error("Failed to generate LiveKit token", "error", err, "room_id", roomID, "participant_id", participantID)
		return "", "", errors.New("failed to generate token")
	}

	// Build the URL for the frontend
	url := s.buildFrontendURL()

	s.log.Info("LiveKit token generated",
		"room_id", roomID,
		"participant_id", participantID,
		"url", url,
		"host_ip", s.cfg.HostIP,
	)

	return token, url, nil
}

// buildFrontendURL builds the correct URL for the frontend to connect to.
func (s *anonymousMediaService) buildFrontendURL() string {
	// If FrontendURL is explicitly set, use it
	if s.cfg.FrontendURL != "" {
		url := s.cfg.FrontendURL
		// Replace localhost with HostIP if set
		if s.cfg.HostIP != "" && s.cfg.HostIP != "localhost" {
			url = strings.Replace(url, "localhost", s.cfg.HostIP, 1)
		}
		// Replace the Docker hostname
		url = strings.Replace(url, "livekit:", s.cfg.HostIP+":", 1)
		return s.normalizeWSURL(url)
	}

	// Build the URL based on HostIP
	hostIP := s.cfg.HostIP
	if hostIP == "" {
		hostIP = "localhost"
	}

	// LiveKit listens on port 7880 by default
	url := "ws://" + hostIP + ":7880"

	return url
}

// normalizeWSURL ensures the URL uses a valid WebSocket scheme.
func (s *anonymousMediaService) normalizeWSURL(url string) string {
	// Strip trailing slash
	url = strings.TrimSuffix(url, "/")

	// Convert http/https to ws/wss
	if strings.HasPrefix(url, "https://") {
		url = "wss://" + strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "http://") {
		url = "ws://" + strings.TrimPrefix(url, "http://")
	}

	// Add protocol if missing
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		url = "ws://" + url
	}

	return url
}
