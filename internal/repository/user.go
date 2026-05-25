package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	CreateSession(ctx context.Context, session *domain.UserSession) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.UserSession, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID, reason string) error
	GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
	UpdateSettings(ctx context.Context, settings *domain.UserSettings) error
	CreateVideoProfile(ctx context.Context, profile *domain.UserVideoProfile) error
	GetVideoProfiles(ctx context.Context, userID uuid.UUID) ([]*domain.UserVideoProfile, error)
}

type userRepository struct {
	db  *pgxpool.Pool
	log logger.Logger
}

func NewUserRepository(db *pgxpool.Pool, log logger.Logger) UserRepository {
	return &userRepository{db: db, log: log}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	// Derive username from display_name for compatibility with the DB schema
	// Truncate to 50 characters to fit the VARCHAR(50) column
	username := user.DisplayName
	if len(username) > 50 {
		username = username[:50]
	}

	// Include username in the INSERT since it is required in the production DB schema
	query := `
		INSERT INTO users (
			id, username, email, password_hash, display_name, avatar_url, 
			global_role, is_active, is_email_verified, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	var avatarURL *string
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		avatarURL = user.AvatarURL
	}

	err := r.db.QueryRow(ctx, query,
		user.ID, username, user.Email, user.PasswordHash, user.DisplayName, avatarURL,
		user.GlobalRole, user.IsActive, user.IsEmailVerified, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for a PostgreSQL unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Code 23505 = unique_violation
			if pgErr.Code == "23505" {
				r.log.Warn("User already exists (unique violation)", "email", user.Email, "constraint", pgErr.ConstraintName)
				return errors.New("user with this email already exists")
			}
			// Other database errors
			r.log.Error("Database error creating user", "error", err, "code", pgErr.Code, "email", user.Email)
			return fmt.Errorf("database error: %s", pgErr.Message)
		}

		// Fallback: inspect the error string
		errStr := err.Error()
		if strings.Contains(errStr, "duplicate key") ||
			strings.Contains(errStr, "unique constraint") ||
			strings.Contains(errStr, "already exists") {
			r.log.Warn("User already exists", "email", user.Email)
			return errors.New("user with this email already exists")
		}

		r.log.Error("Failed to create user", "error", err, "email", user.Email)
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	// Query works with both schemas (with or without the username column)
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, global_role, is_active, 
		       is_email_verified, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	var lastLoginAt *time.Time
	var avatarURL *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &avatarURL,
		&user.GlobalRole, &user.IsActive, &user.IsEmailVerified, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		r.log.Error("Failed to get user by ID", "error", err)
		return nil, err
	}

	user.LastLoginAt = lastLoginAt
	user.AvatarURL = avatarURL
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Normalize email (lowercase, trim)
	email = strings.ToLower(strings.TrimSpace(email))

	query := `
		SELECT id, email, password_hash, display_name, avatar_url, global_role, is_active, 
		       is_email_verified, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	var lastLoginAt *time.Time
	var avatarURL *string

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &avatarURL,
		&user.GlobalRole, &user.IsActive, &user.IsEmailVerified, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		r.log.Error("Failed to get user by email", "error", err, "email", email)
		return nil, err
	}

	user.LastLoginAt = lastLoginAt
	user.AvatarURL = avatarURL
	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $2, display_name = $3, avatar_url = $4, is_active = $5, 
		    is_email_verified = $6, last_login_at = $7, updated_at = $8
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.ID, user.Email, user.DisplayName, user.AvatarURL, user.IsActive,
		user.IsEmailVerified, user.LastLoginAt, time.Now(),
	).Scan(&user.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to update user", "error", err)
		return err
	}

	return nil
}

func (r *userRepository) CreateSession(ctx context.Context, session *domain.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, created_at, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		session.ID, session.UserID, session.RefreshTokenHash,
		session.CreatedAt, session.ExpiresAt, session.IPAddress, session.UserAgent,
	)

	if err != nil {
		r.log.Error("Failed to create session", "error", err)
		return err
	}

	return nil
}

func (r *userRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.UserSession, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, created_at, expires_at, revoked_at, revoked_reason, ip_address, user_agent
		FROM user_sessions
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`

	session := &domain.UserSession{}
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID, &session.UserID, &session.RefreshTokenHash,
		&session.CreatedAt, &session.ExpiresAt, &session.RevokedAt,
		&session.RevokedReason, &session.IPAddress, &session.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		r.log.Error("Failed to get session", "error", err)
		return nil, err
	}

	return session, nil
}

func (r *userRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW(), revoked_reason = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, sessionID, reason)
	if err != nil {
		r.log.Error("Failed to revoke session", "error", err)
		return err
	}

	return nil
}

func (r *userRepository) GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	query := `
		SELECT user_id, default_camera_device_id, default_microphone_device_id, default_speaker_device_id,
		       preferred_video_quality, preferred_theme, mute_mic_on_join, disable_camera_on_join,
		       language_code, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`

	settings := &domain.UserSettings{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.UserID, &settings.DefaultCameraDeviceID, &settings.DefaultMicrophoneDeviceID,
		&settings.DefaultSpeakerDeviceID, &settings.PreferredVideoQuality, &settings.PreferredTheme,
		&settings.MuteMicOnJoin, &settings.DisableCameraOnJoin, &settings.LanguageCode,
		&settings.CreatedAt, &settings.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create default settings
			return r.createDefaultSettings(ctx, userID)
		}
		r.log.Error("Failed to get user settings", "error", err)
		return nil, err
	}

	return settings, nil
}

func (r *userRepository) createDefaultSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	settings := &domain.UserSettings{
		UserID:                userID,
		PreferredVideoQuality: domain.VideoQualityAuto,
		PreferredTheme:        domain.ThemeSystem,
		LanguageCode:          "en",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	query := `
		INSERT INTO user_settings (user_id, preferred_video_quality, preferred_theme, language_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		settings.UserID, settings.PreferredVideoQuality, settings.PreferredTheme,
		settings.LanguageCode, settings.CreatedAt, settings.UpdatedAt,
	).Scan(&settings.CreatedAt, &settings.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to create default settings", "error", err)
		return nil, err
	}

	return settings, nil
}

func (r *userRepository) UpdateSettings(ctx context.Context, settings *domain.UserSettings) error {
	query := `
		UPDATE user_settings
		SET default_camera_device_id = $2, default_microphone_device_id = $3, default_speaker_device_id = $4,
		    preferred_video_quality = $5, preferred_theme = $6, mute_mic_on_join = $7,
		    disable_camera_on_join = $8, language_code = $9, updated_at = $10
		WHERE user_id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		settings.UserID, settings.DefaultCameraDeviceID, settings.DefaultMicrophoneDeviceID,
		settings.DefaultSpeakerDeviceID, settings.PreferredVideoQuality, settings.PreferredTheme,
		settings.MuteMicOnJoin, settings.DisableCameraOnJoin, settings.LanguageCode, time.Now(),
	).Scan(&settings.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to update settings", "error", err)
		return err
	}

	return nil
}

func (r *userRepository) CreateVideoProfile(ctx context.Context, profile *domain.UserVideoProfile) error {
	query := `
		INSERT INTO user_video_profiles (id, user_id, name, background_type, background_image_url, noise_suppression_level, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query,
		profile.ID, profile.UserID, profile.Name, profile.BackgroundType,
		profile.BackgroundImageURL, profile.NoiseSuppressionLevel, profile.IsDefault, profile.CreatedAt,
	)

	if err != nil {
		r.log.Error("Failed to create video profile", "error", err)
		return err
	}

	return nil
}

func (r *userRepository) GetVideoProfiles(ctx context.Context, userID uuid.UUID) ([]*domain.UserVideoProfile, error) {
	query := `
		SELECT id, user_id, name, background_type, background_image_url, noise_suppression_level, is_default, created_at
		FROM user_video_profiles
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to get video profiles", "error", err)
		return nil, err
	}
	defer rows.Close()

	var profiles []*domain.UserVideoProfile
	for rows.Next() {
		profile := &domain.UserVideoProfile{}
		err := rows.Scan(
			&profile.ID, &profile.UserID, &profile.Name, &profile.BackgroundType,
			&profile.BackgroundImageURL, &profile.NoiseSuppressionLevel, &profile.IsDefault, &profile.CreatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan video profile", "error", err)
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}
