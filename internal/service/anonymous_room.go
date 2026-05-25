package service

import (
	"context"
	"errors"
	"time"

	"vibemeet/internal/config"
	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"

	"github.com/google/uuid"
)

type AnonymousRoomService interface {
	Create(ctx context.Context, title string, description *string, maxParticipants int, participantID string, displayName string) (*domain.AnonymousRoom, *domain.AnonymousParticipant, error)
	GetByID(ctx context.Context, roomID uuid.UUID) (*domain.AnonymousRoom, error)
	Join(ctx context.Context, roomID uuid.UUID, participantID string, displayName string) (*domain.AnonymousParticipant, error)
	Leave(ctx context.Context, roomID uuid.UUID, participantID string) error
	GetParticipants(ctx context.Context, roomID uuid.UUID) ([]*domain.AnonymousParticipant, error)
}

type anonymousRoomService struct {
	roomRepo repository.AnonymousRoomRepository
	cfg      *config.Config
	log      logger.Logger
}

func NewAnonymousRoomService(roomRepo repository.AnonymousRoomRepository, cfg *config.Config, log logger.Logger) AnonymousRoomService {
	return &anonymousRoomService{
		roomRepo: roomRepo,
		cfg:      cfg,
		log:      log,
	}
}

func (s *anonymousRoomService) Create(ctx context.Context, title string, description *string, maxParticipants int, participantID string, displayName string) (*domain.AnonymousRoom, *domain.AnonymousParticipant, error) {
	if maxParticipants <= 0 || maxParticipants > 500 {
		maxParticipants = 10
	}

	if title == "" {
		title = "New room"
	}

	// Create the room
	roomID := uuid.New()
	now := time.Now()

	room := &domain.AnonymousRoom{
		ID:              roomID,
		LiveKitRoomName: roomID.String(),
		Title:           title,
		Description:     description,
		Status:          domain.RoomStatusActive,
		MaxParticipants: maxParticipants,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		s.log.Error("Failed to create anonymous room", "error", err)
		return nil, nil, errors.New("failed to create room")
	}

	// Create the participant (host)
	participant := &domain.AnonymousParticipant{
		ID:            uuid.New(),
		RoomID:        roomID,
		ParticipantID: participantID,
		DisplayName:   displayName,
		Role:          domain.ParticipantRoleHost,
		JoinedAt:      now,
	}

	if err := s.roomRepo.CreateParticipant(ctx, participant); err != nil {
		s.log.Error("Failed to create participant", "error", err)
		// Non-fatal, continue
	}

	s.log.Info("Anonymous room created", "room_id", roomID, "participant_id", participantID)

	return room, participant, nil
}

func (s *anonymousRoomService) GetByID(ctx context.Context, roomID uuid.UUID) (*domain.AnonymousRoom, error) {
	return s.roomRepo.GetByID(ctx, roomID)
}

func (s *anonymousRoomService) Join(ctx context.Context, roomID uuid.UUID, participantID string, displayName string) (*domain.AnonymousParticipant, error) {
	// Check that the room exists
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, errors.New("room not found")
	}

	if room.Status != domain.RoomStatusActive {
		return nil, errors.New("room is not active")
	}

	// Check if this participant has already joined
	existingParticipant, err := s.roomRepo.GetParticipant(ctx, roomID, participantID)
	if err == nil && existingParticipant != nil && existingParticipant.LeftAt == nil {
		// Participant is already in the room
		return existingParticipant, nil
	}

	// Create a new participant
	participant := &domain.AnonymousParticipant{
		ID:            uuid.New(),
		RoomID:        roomID,
		ParticipantID: participantID,
		DisplayName:   displayName,
		Role:          domain.ParticipantRoleParticipant,
		JoinedAt:      time.Now(),
	}

	if err := s.roomRepo.CreateParticipant(ctx, participant); err != nil {
		s.log.Error("Failed to create participant", "error", err)
		return nil, errors.New("failed to join room")
	}

	return participant, nil
}

func (s *anonymousRoomService) Leave(ctx context.Context, roomID uuid.UUID, participantID string) error {
	// Mark the participant as having left
	if err := s.roomRepo.MarkParticipantLeft(ctx, roomID, participantID); err != nil {
		s.log.Error("Failed to mark participant left", "error", err, "room_id", roomID, "participant_id", participantID)
		return err
	}

	// Count remaining active participants
	count, err := s.roomRepo.GetActiveParticipantCount(ctx, roomID)
	if err != nil {
		s.log.Error("Failed to get active participant count", "error", err, "room_id", roomID)
		return nil // Don't block the leave on a count error
	}

	// If no participants remain, deactivate the room
	if count == 0 {
		s.log.Info("Room is empty, deactivating", "room_id", roomID)
		if err := s.roomRepo.SetRoomStatus(ctx, roomID, domain.RoomStatusEnded); err != nil {
			s.log.Error("Failed to deactivate empty room", "error", err, "room_id", roomID)
		}
	}

	return nil
}

func (s *anonymousRoomService) GetParticipants(ctx context.Context, roomID uuid.UUID) ([]*domain.AnonymousParticipant, error) {
	return s.roomRepo.GetParticipantsByRoom(ctx, roomID)
}
