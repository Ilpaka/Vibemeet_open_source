package domain

import (
	"time"

	"github.com/google/uuid"
)

// AnonymousRoom is a simplified room model that is not tied to a user account
type AnonymousRoom struct {
	ID              uuid.UUID  `json:"id"`
	LiveKitRoomName string     `json:"livekit_room_name"`
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	Status          string     `json:"status"` // active, ended
	MaxParticipants int        `json:"max_participants"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"` // Automatic cleanup of inactive rooms
}

// AnonymousParticipant represents an anonymous participant of a room
type AnonymousParticipant struct {
	ID            uuid.UUID  `json:"id"`
	RoomID        uuid.UUID  `json:"room_id"`
	ParticipantID string     `json:"participant_id"` // Temporary client-side ID (UUID string)
	DisplayName   string     `json:"display_name"`
	Role          string     `json:"role"` // host, participant
	LiveKitSID    *string    `json:"livekit_sid,omitempty"`
	JoinedAt      time.Time  `json:"joined_at"`
	LeftAt        *time.Time `json:"left_at,omitempty"`
	ClientIP      *string    `json:"client_ip,omitempty"`
	UserAgent     *string    `json:"user_agent,omitempty"`
}

// AnonymousChatMessage is a chat message stored in Redis
type AnonymousChatMessage struct {
	ID            string    `json:"id"`             // UUID string
	RoomID        uuid.UUID `json:"room_id"`        // Room UUID
	ParticipantID string    `json:"participant_id"` // Temporary participant ID
	DisplayName   string    `json:"display_name"`   // Sender display name
	MessageType   string    `json:"message_type"`   // user, system
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
}

// Message type constants
const (
	AnonymousMessageTypeUser   = "user"
	AnonymousMessageTypeSystem = "system"
)

// Note: RoomStatusActive, RoomStatusEnded, ParticipantRoleHost, and
// ParticipantRoleParticipant are defined in room.go and reused here.
