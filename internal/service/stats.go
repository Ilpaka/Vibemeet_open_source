package service

import (
	"context"

	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"

	"github.com/google/uuid"
)

type StatsService interface {
	GetParticipantStats(ctx context.Context, participantID uuid.UUID) (*domain.ParticipantStats, error)
	GetRoomStats(ctx context.Context, roomID uuid.UUID) ([]*domain.ParticipantStats, error)
	SaveParticipantStats(ctx context.Context, stats *domain.ParticipantStats) error
}

type statsService struct {
	statsRepo repository.StatsRepository
	log       logger.Logger
}

func NewStatsService(statsRepo repository.StatsRepository, log logger.Logger) StatsService {
	return &statsService{
		statsRepo: statsRepo,
		log:       log,
	}
}

func (s *statsService) GetParticipantStats(ctx context.Context, participantID uuid.UUID) (*domain.ParticipantStats, error) {
	return s.statsRepo.GetParticipantStats(ctx, participantID)
}

func (s *statsService) GetRoomStats(ctx context.Context, roomID uuid.UUID) ([]*domain.ParticipantStats, error) {
	return s.statsRepo.GetRoomStats(ctx, roomID)
}

func (s *statsService) SaveParticipantStats(ctx context.Context, stats *domain.ParticipantStats) error {
	return s.statsRepo.CreateParticipantStats(ctx, stats)
}
