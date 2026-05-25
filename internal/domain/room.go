package domain

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID                 uuid.UUID              `json:"id"`
	LiveKitRoomName    string                 `json:"livekit_room_name"`
	HostUserID         uuid.UUID              `json:"host_user_id"`
	Title              string                 `json:"title"`
	Description        *string                `json:"description,omitempty"`
	Status             string                 `json:"status"`
	ScheduledStartAt   *time.Time             `json:"scheduled_start_at,omitempty"`
	ScheduledEndAt     *time.Time             `json:"scheduled_end_at,omitempty"`
	ActualStartAt      *time.Time             `json:"actual_start_at,omitempty"`
	ActualEndAt        *time.Time             `json:"actual_end_at,omitempty"`
	MaxParticipants    int                    `json:"max_participants"`
	WaitingRoomEnabled bool                   `json:"waiting_room_enabled"`
	IsLocked           bool                   `json:"is_locked"`
	PasswordHash       *string                `json:"-"`
	Settings           map[string]interface{} `json:"settings"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type RoomInvite struct {
	ID            uuid.UUID  `json:"id"`
	RoomID        uuid.UUID  `json:"room_id"`
	CreatedByUserID uuid.UUID  `json:"created_by_user_id"`
	LinkToken     string     `json:"link_token"`
	Label         *string    `json:"label,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	MaxUses       *int       `json:"max_uses,omitempty"`
	UsedCount     int        `json:"used_count"`
	CreatedAt     time.Time  `json:"created_at"`
}

type RoomParticipant struct {
	ID            uuid.UUID  `json:"id"`
	RoomID        uuid.UUID  `json:"room_id"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	Role          string     `json:"role"`
	DisplayName   string     `json:"display_name"`
	LiveKitSID    *string    `json:"livekit_sid,omitempty"`
	JoinedAt      time.Time  `json:"joined_at"`
	LeftAt        *time.Time `json:"left_at,omitempty"`
	LeaveReason   *string    `json:"leave_reason,omitempty"`
	IsKicked      bool       `json:"is_kicked"`
	InitialMuted  bool       `json:"initial_muted"`
	ClientIP      *string    `json:"client_ip,omitempty"`
	UserAgent     *string    `json:"user_agent,omitempty"`
}

type WaitingRoomEntry struct {
	ID              uuid.UUID  `json:"id"`
	RoomID          uuid.UUID  `json:"room_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	DisplayName     string     `json:"display_name"`
	Status          string     `json:"status"`
	RequestedAt     time.Time  `json:"requested_at"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
	DecidedByUserID *uuid.UUID `json:"decided_by_user_id,omitempty"`
	Reason          *string    `json:"reason,omitempty"`
}

const (
	RoomStatusScheduled = "scheduled"
	RoomStatusActive    = "active"
	RoomStatusEnded     = "ended"
	RoomStatusCancelled = "cancelled"
)

const (
	ParticipantRoleHost       = "host"
	ParticipantRoleCoHost     = "co_host"
	ParticipantRoleParticipant = "participant"
)

const (
	WaitingRoomStatusPending  = "pending"
	WaitingRoomStatusApproved = "approved"
	WaitingRoomStatusRejected = "rejected"
	WaitingRoomStatusExpired  = "expired"
)

