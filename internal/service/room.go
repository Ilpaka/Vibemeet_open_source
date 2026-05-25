package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"vibemeet/internal/config"
	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"
)

type RoomService interface {
	Create(ctx context.Context, hostUserID uuid.UUID, title string, description *string, maxParticipants int) (*domain.Room, error)
	GetByID(ctx context.Context, roomID uuid.UUID) (*domain.Room, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error)
	Update(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, title *string, description *string, maxParticipants *int) (*domain.Room, error)
	Delete(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error
	Join(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, displayName string) (*domain.RoomParticipant, error)
	Leave(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error
	CreateInvite(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, label *string, expiresAt *time.Time, maxUses *int) (*domain.RoomInvite, error)
	GetParticipants(ctx context.Context, roomID uuid.UUID) ([]*domain.RoomParticipant, error)
}

type roomService struct {
	roomRepo repository.RoomRepository
	auditRepo repository.AuditRepository
	cfg      *config.Config
	log      logger.Logger
}

func NewRoomService(roomRepo repository.RoomRepository, auditRepo repository.AuditRepository, cfg *config.Config, log logger.Logger) RoomService {
	return &roomService{
		roomRepo:  roomRepo,
		auditRepo: auditRepo,
		cfg:       cfg,
		log:       log,
	}
}

func (s *roomService) Create(ctx context.Context, hostUserID uuid.UUID, title string, description *string, maxParticipants int) (*domain.Room, error) {
	if maxParticipants <= 0 || maxParticipants > 500 {
		maxParticipants = 10
	}

	room := &domain.Room{
		ID:                 uuid.New(),
		LiveKitRoomName:    uuid.New().String(),
		HostUserID:         hostUserID,
		Title:              title,
		Description:        description,
		Status:             domain.RoomStatusScheduled,
		MaxParticipants:    maxParticipants,
		WaitingRoomEnabled: true,
		IsLocked:           false,
		Settings:           make(map[string]interface{}),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		s.log.Error("Failed to create room", "error", err)
		return nil, errors.New("failed to create room")
	}

	// Audit
	s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		EventTime:   time.Now(),
		ActorUserID: &hostUserID,
		ActorRole:   domain.ActorRoleHost,
		RoomID:      &room.ID,
		EventType:   domain.EventTypeRoomCreated,
		Payload:     map[string]interface{}{"title": title},
	})

	return room, nil
}

func (s *roomService) GetByID(ctx context.Context, roomID uuid.UUID) (*domain.Room, error) {
	return s.roomRepo.GetByID(ctx, roomID)
}

func (s *roomService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.roomRepo.List(ctx, userID, limit, offset)
}

func (s *roomService) Update(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, title *string, description *string, maxParticipants *int) (*domain.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if room.HostUserID != userID {
		return nil, errors.New("only host can update room")
	}

	if title != nil {
		room.Title = *title
	}
	if description != nil {
		room.Description = description
	}
	if maxParticipants != nil {
		if *maxParticipants > 0 && *maxParticipants <= 500 {
			room.MaxParticipants = *maxParticipants
		}
	}
	room.UpdatedAt = time.Now()

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *roomService) Delete(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return err
	}

	if room.HostUserID != userID {
		return errors.New("only host can delete room")
	}

	return s.roomRepo.Delete(ctx, roomID)
}

func (s *roomService) Join(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, displayName string) (*domain.RoomParticipant, error) {
	// First check if room exists in database
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		s.log.Error("Failed to get room by ID", "room_id", roomID, "error", err)
		// Return the error as-is (will be "room not found" if room doesn't exist)
		return nil, err
	}

	s.log.Info("Room found in database", "room_id", roomID, "title", room.Title, "status", room.Status)

	if room.Status != domain.RoomStatusActive && room.Status != domain.RoomStatusScheduled {
		return nil, errors.New("room is not available")
	}

	// Check whether the participant already exists
	existingParticipant, err := s.roomRepo.GetParticipant(ctx, roomID, userID)
	if err == nil && existingParticipant.LeftAt == nil {
		// Participant is already in the room
		return existingParticipant, nil
	}

	// Waiting room check
	if room.WaitingRoomEnabled && room.HostUserID != userID {
		entry := &domain.WaitingRoomEntry{
			ID:          uuid.New(),
			RoomID:      roomID,
			UserID:      &userID,
			DisplayName: displayName,
			Status:      domain.WaitingRoomStatusPending,
			RequestedAt: time.Now(),
		}
		if err := s.roomRepo.CreateWaitingRoomEntry(ctx, entry); err != nil {
			return nil, err
		}
		return nil, errors.New("waiting for approval")
	}

	// Determine the role
	role := domain.ParticipantRoleParticipant
	if room.HostUserID == userID {
		role = domain.ParticipantRoleHost
	}

	// Create the participant
	participant := &domain.RoomParticipant{
		ID:           uuid.New(),
		RoomID:       roomID,
		UserID:       &userID,
		Role:         role,
		DisplayName:  displayName,
		JoinedAt:     time.Now(),
		InitialMuted: false,
	}

	if err := s.roomRepo.CreateParticipant(ctx, participant); err != nil {
		return nil, err
	}

	// Promote the room to active on the first join
	if room.Status == domain.RoomStatusScheduled {
		room.Status = domain.RoomStatusActive
		now := time.Now()
		room.ActualStartAt = &now
		if err := s.roomRepo.Update(ctx, room); err != nil {
			s.log.Warn("Failed to update room status", "error", err)
		}
	}

	return participant, nil
}

func (s *roomService) Leave(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error {
	participant, err := s.roomRepo.GetParticipant(ctx, roomID, userID)
	if err != nil {
		return err
	}

	now := time.Now()
	participant.LeftAt = &now
	reason := "left"
	participant.LeaveReason = &reason

	return s.roomRepo.UpdateParticipant(ctx, participant)
}

func (s *roomService) CreateInvite(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, label *string, expiresAt *time.Time, maxUses *int) (*domain.RoomInvite, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if room.HostUserID != userID {
		return nil, errors.New("only host can create invites")
	}

	invite := &domain.RoomInvite{
		ID:            uuid.New(),
		RoomID:        roomID,
		CreatedByUserID: userID,
		LinkToken:     uuid.New().String(),
		Label:          label,
		ExpiresAt:     expiresAt,
		MaxUses:       maxUses,
		UsedCount:     0,
		CreatedAt:     time.Now(),
	}

	if err := s.roomRepo.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

func (s *roomService) GetParticipants(ctx context.Context, roomID uuid.UUID) ([]*domain.RoomParticipant, error) {
	return s.roomRepo.GetParticipantsByRoom(ctx, roomID)
}

