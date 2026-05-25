package service

import (
	"vibemeet/internal/config"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"
)

type Services struct {
	Auth             AuthService
	User             UserService
	Room             RoomService
	Chat             ChatService
	Media            MediaService
	AnonymousMedia   AnonymousMediaService
	AnonymousRoom    AnonymousRoomService
	Stats            StatsService
	RateLimit        RateLimitService
	Audit            AuditService
	ScreenCapture    ScreenCaptureService
	AudioCapture     AudioCaptureService
	WebRTC           WebRTCService
}

func NewServices(repos *repository.Repositories, cfg *config.Config, log logger.Logger) *Services {
	services := &Services{
		Auth:          NewAuthService(repos.User, cfg.JWT, log),
		User:          NewUserService(repos.User, repos.Audit, log),
		Room:          NewRoomService(repos.Room, repos.Audit, cfg, log),
		Chat:          NewChatService(repos.Chat, repos.Room, repos.Audit, log),
		Media:         NewMediaService(repos.Room, cfg.LiveKit, log),
		Stats:         NewStatsService(repos.Stats, log),
		RateLimit:     NewRateLimitService(repos.RateLimit, log),
		Audit:         NewAuditService(repos.Audit, log),
		ScreenCapture: NewScreenCaptureService(log),
		AudioCapture:  NewAudioCaptureService(log),
		WebRTC:        NewWebRTCService(log),
	}
	
	// Initialize anonymous services only if the AnonymousRoom repository is available
	if repos.AnonymousRoom != nil {
		log.Info("Creating AnonymousRoom service...")
		services.AnonymousRoom = NewAnonymousRoomService(repos.AnonymousRoom, cfg, log)
		log.Info("AnonymousRoom service initialized")
		services.AnonymousMedia = NewAnonymousMediaService(repos.AnonymousRoom, cfg.LiveKit, log)
		log.Info("AnonymousMedia service initialized")
	} else {
		log.Warn("AnonymousRoom repository is nil, anonymous services not initialized")
	}
	
	return services
}

