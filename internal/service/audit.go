package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"
)

type AuditService interface {
	LogEvent(ctx context.Context, actorUserID *uuid.UUID, actorRole string, roomID *uuid.UUID, eventType string, payload map[string]interface{}) error
}

type auditService struct {
	auditRepo repository.AuditRepository
	log       logger.Logger
}

func NewAuditService(auditRepo repository.AuditRepository, log logger.Logger) AuditService {
	return &auditService{
		auditRepo: auditRepo,
		log:       log,
	}
}

func (s *auditService) LogEvent(ctx context.Context, actorUserID *uuid.UUID, actorRole string, roomID *uuid.UUID, eventType string, payload map[string]interface{}) error {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	auditLog := &domain.AuditLog{
		EventTime:   time.Now(),
		ActorUserID: actorUserID,
		ActorRole:   actorRole,
		RoomID:      roomID,
		EventType:   eventType,
		Payload:     payload,
	}

	return s.auditRepo.CreateLog(ctx, auditLog)
}
