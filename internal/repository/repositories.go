package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"vibemeet/pkg/logger"
)

type Repositories struct {
	User           UserRepository
	Room           RoomRepository
	AnonymousRoom  AnonymousRoomRepository
	AnonymousChat  AnonymousChatRepository
	Chat           ChatRepository
	Stats          StatsRepository
	Audit          AuditRepository
	RateLimit      RateLimitRepository
}

func NewRepositories(db *pgxpool.Pool, redis *redis.Client, log logger.Logger) *Repositories {
	repos := &Repositories{
		User:          NewUserRepository(db, log),
		Room:          NewRoomRepository(db, log),
		AnonymousRoom: NewAnonymousRoomRepository(db, log),
		AnonymousChat: NewAnonymousChatRepository(redis, log),
		Chat:          NewChatRepository(db, log),
		Stats:         NewStatsRepository(db, log),
		Audit:         NewAuditRepository(db, log),
		RateLimit:     NewRateLimitRepository(redis, log),
	}
	
	if repos.AnonymousRoom != nil {
		log.Info("AnonymousRoom repository initialized")
	} else {
		log.Warn("AnonymousRoom repository is nil")
	}
	
	log.Info("AnonymousChat repository initialized")
	
	return repos
}

