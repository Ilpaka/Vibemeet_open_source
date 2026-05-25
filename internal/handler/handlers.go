package handler

import (
	"vibemeet/internal/config"
	"vibemeet/internal/repository"
	"vibemeet/internal/service"
	"vibemeet/pkg/logger"
)

// Handlers is the registry of every HTTP handler in the application.
// Anonymous handlers may be nil when their backing service is disabled.
type Handlers struct {
	Health         *HealthHandler
	Auth           *AuthHandler
	User           *UserHandler
	Room           *RoomHandler
	Chat           *ChatHandler
	AnonymousChat  *AnonymousChatHandler
	Media          *MediaHandler
	AnonymousMedia *AnonymousMediaHandler
	AnonymousRoom  *AnonymousRoomHandler
	Stats          *StatsHandler
	ScreenShare    *ScreenShareHandler
}

func NewHandlers(services *service.Services, repos *repository.Repositories, cfg *config.Config, log logger.Logger) *Handlers {
	handlers := &Handlers{
		Health:      NewHealthHandler(cfg),
		Auth:        NewAuthHandler(services.Auth, log),
		User:        NewUserHandler(services.User, log),
		Room:        NewRoomHandler(services.Room, log),
		Chat:        NewChatHandler(services.Chat, log),
		Media:       NewMediaHandler(services.Media, log),
		Stats:       NewStatsHandler(services.Stats, log),
		ScreenShare: NewScreenShareHandler(services.ScreenCapture, services.AudioCapture, services.WebRTC, log),
	}

	if services.AnonymousRoom != nil {
		handlers.AnonymousRoom = NewAnonymousRoomHandler(services.AnonymousRoom, log)
		log.Info("AnonymousRoom handler initialized")
	}
	if services.AnonymousMedia != nil {
		handlers.AnonymousMedia = NewAnonymousMediaHandler(services.AnonymousMedia, log)
		log.Info("AnonymousMedia handler initialized")
	}
	if repos.AnonymousChat != nil && repos.AnonymousRoom != nil {
		handlers.AnonymousChat = NewAnonymousChatHandler(repos.AnonymousChat, repos.AnonymousRoom, log)
		log.Info("AnonymousChat handler initialized")
	}

	return handlers
}
