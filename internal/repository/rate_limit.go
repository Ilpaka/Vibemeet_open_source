package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"vibemeet/pkg/logger"
)

type RateLimitRepository interface {
	CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
}

type rateLimitRepository struct {
	redis *redis.Client
	log   logger.Logger
}

func NewRateLimitRepository(redis *redis.Client, log logger.Logger) RateLimitRepository {
	return &rateLimitRepository{redis: redis, log: log}
}

func (r *rateLimitRepository) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	count, err := r.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		return true, nil
	}
	if err != nil {
		r.log.Error("Failed to check rate limit", "error", err)
		return false, err
	}
	
	return count < limit, nil
}

func (r *rateLimitRepository) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	count, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		r.log.Error("Failed to increment rate limit", "error", err)
		return 0, err
	}
	
	if count == 1 {
		r.redis.Expire(ctx, key, window)
	}
	
	return count, nil
}

