package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"
)

type ChatService interface {
	SendMessage(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, content string) (*domain.ChatMessage, error)
	GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*domain.ChatMessage, error)
	EditMessage(ctx context.Context, messageID int64, userID uuid.UUID, content string) (*domain.ChatMessage, error)
	DeleteMessage(ctx context.Context, messageID int64, userID uuid.UUID) error
}

type chatService struct {
	chatRepo  repository.ChatRepository
	roomRepo  repository.RoomRepository
	auditRepo repository.AuditRepository
	log       logger.Logger
}

func NewChatService(chatRepo repository.ChatRepository, roomRepo repository.RoomRepository, auditRepo repository.AuditRepository, log logger.Logger) ChatService {
	return &chatService{
		chatRepo:  chatRepo,
		roomRepo:  roomRepo,
		auditRepo: auditRepo,
		log:       log,
	}
}

func (s *chatService) SendMessage(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, content string) (*domain.ChatMessage, error) {
	// Verify the room exists
	_, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, errors.New("room not found")
	}

	// Get or create the participant
	participant, err := s.roomRepo.GetParticipant(ctx, roomID, userID)
	if err != nil {
		// If the participant is not found, create one
		participant = &domain.RoomParticipant{
			ID:           uuid.New(),
			RoomID:       roomID,
			UserID:       &userID,
			Role:         domain.ParticipantRoleParticipant,
			DisplayName:  "User",
			JoinedAt:     time.Now(),
			InitialMuted: false,
		}
		if err := s.roomRepo.CreateParticipant(ctx, participant); err != nil {
			return nil, errors.New("failed to create participant")
		}
	}

	message := &domain.ChatMessage{
		RoomID:              roomID,
		SenderParticipantID: &participant.ID,
		MessageType:         domain.MessageTypeUser,
		Content:             content,
		CreatedAt:           time.Now(),
	}

	if err := s.chatRepo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *chatService) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*domain.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.chatRepo.GetMessages(ctx, roomID, limit, offset)
}

func (s *chatService) EditMessage(ctx context.Context, messageID int64, userID uuid.UUID, content string) (*domain.ChatMessage, error) {
	message, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if message.SenderParticipantID == nil {
		return nil, errors.New("message has no sender")
	}

	// Look up the participant by ID
	participant, err := s.roomRepo.GetParticipantByID(ctx, *message.SenderParticipantID)
	if err != nil || participant.UserID == nil || *participant.UserID != userID {
		return nil, errors.New("only sender can edit message")
	}

	message.Content = content
	if err := s.chatRepo.UpdateMessage(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *chatService) DeleteMessage(ctx context.Context, messageID int64, userID uuid.UUID) error {
	message, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	if message.SenderParticipantID == nil {
		return errors.New("message has no sender")
	}

	// Look up the participant by ID
	participant, err := s.roomRepo.GetParticipantByID(ctx, *message.SenderParticipantID)
	if err != nil || participant.UserID == nil || *participant.UserID != userID {
		return errors.New("only sender can delete message")
	}

	return s.chatRepo.DeleteMessage(ctx, messageID, *message.SenderParticipantID)
}

