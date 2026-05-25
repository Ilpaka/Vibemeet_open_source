package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"vibemeet/internal/domain"
	"vibemeet/internal/repository"
	"vibemeet/pkg/logger"
)

type UserService interface {
	GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateMe(ctx context.Context, userID uuid.UUID, displayName string, avatarURL *string) (*domain.User, error)
	GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
	UpdateSettings(ctx context.Context, userID uuid.UUID, settings *domain.UserSettings) error
}

type userService struct {
	userRepo  repository.UserRepository
	auditRepo repository.AuditRepository
	log       logger.Logger
}

func NewUserService(userRepo repository.UserRepository, auditRepo repository.AuditRepository, log logger.Logger) UserService {
	return &userService{
		userRepo:  userRepo,
		auditRepo: auditRepo,
		log:       log,
	}
}

func (s *userService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *userService) UpdateMe(ctx context.Context, userID uuid.UUID, displayName string, avatarURL *string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.DisplayName = displayName
	if avatarURL != nil {
		user.AvatarURL = avatarURL
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return user, nil
}

func (s *userService) GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	return s.userRepo.GetSettings(ctx, userID)
}

func (s *userService) UpdateSettings(ctx context.Context, userID uuid.UUID, settings *domain.UserSettings) error {
	settings.UserID = userID
	settings.UpdatedAt = time.Now()
	return s.userRepo.UpdateSettings(ctx, settings)
}
