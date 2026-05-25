package service

import (
	"context"
	"time"

	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"
)

type RateLimitService interface {
	CheckLimit(ctx context.Context, key string, limit int, windowSeconds int) (bool, error)
	Increment(ctx context.Context, key string, windowSeconds int) (int64, error)
}

type rateLimitService struct {
	rateLimitRepo repository.RateLimitRepository
	log           logger.Logger
}

func NewRateLimitService(rateLimitRepo repository.RateLimitRepository, log logger.Logger) RateLimitService {
	return &rateLimitService{
		rateLimitRepo: rateLimitRepo,
		log:           log,
	}
}

func (s *rateLimitService) CheckLimit(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	return s.rateLimitRepo.CheckLimit(ctx, key, limit, time.Duration(windowSeconds)*time.Second)
}

func (s *rateLimitService) Increment(ctx context.Context, key string, windowSeconds int) (int64, error) {
	return s.rateLimitRepo.Increment(ctx, key, time.Duration(windowSeconds)*time.Second)
}

