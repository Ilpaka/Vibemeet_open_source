package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"`
	DisplayName     string    `json:"display_name"`
	AvatarURL       *string   `json:"avatar_url,omitempty"`
	GlobalRole      string    `json:"global_role"`
	IsActive        bool      `json:"is_active"`
	IsEmailVerified bool      `json:"is_email_verified"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserSession struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	RefreshTokenHash string    `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	RevokedReason  *string    `json:"revoked_reason,omitempty"`
	IPAddress      *string    `json:"ip_address,omitempty"`
	UserAgent      *string    `json:"user_agent,omitempty"`
}

type UserSettings struct {
	UserID                    uuid.UUID `json:"user_id"`
	DefaultCameraDeviceID     *string   `json:"default_camera_device_id,omitempty"`
	DefaultMicrophoneDeviceID *string   `json:"default_microphone_device_id,omitempty"`
	DefaultSpeakerDeviceID    *string   `json:"default_speaker_device_id,omitempty"`
	PreferredVideoQuality     string    `json:"preferred_video_quality"`
	PreferredTheme            string    `json:"preferred_theme"`
	MuteMicOnJoin             bool      `json:"mute_mic_on_join"`
	DisableCameraOnJoin       bool      `json:"disable_camera_on_join"`
	LanguageCode              string    `json:"language_code"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type UserVideoProfile struct {
	ID                   uuid.UUID `json:"id"`
	UserID               uuid.UUID `json:"user_id"`
	Name                 string    `json:"name"`
	BackgroundType       string    `json:"background_type"`
	BackgroundImageURL   *string   `json:"background_image_url,omitempty"`
	NoiseSuppressionLevel string    `json:"noise_suppression_level"`
	IsDefault            bool      `json:"is_default"`
	CreatedAt            time.Time `json:"created_at"`
}

const (
	GlobalRoleUser          = "user"
	GlobalRoleTechnicalAdmin = "technical_admin"
)

const (
	VideoQuality1080p = "1080p"
	VideoQuality720p  = "720p"
	VideoQuality480p  = "480p"
	VideoQuality360p  = "360p"
	VideoQualityAuto  = "auto"
)

const (
	ThemeLight  = "light"
	ThemeDark   = "dark"
	ThemeSystem = "system"
)

const (
	BackgroundTypeNone   = "none"
	BackgroundTypeBlur   = "blur"
	BackgroundTypeImage  = "image"
)

const (
	NoiseSuppressionOff    = "off"
	NoiseSuppressionLow    = "low"
	NoiseSuppressionMedium = "medium"
	NoiseSuppressionHigh   = "high"
)

