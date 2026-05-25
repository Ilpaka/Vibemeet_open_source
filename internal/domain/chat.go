package domain

import (
	"time"

	"github.com/google/uuid"
)

type ChatMessage struct {
	ID                    int64      `json:"id"`
	RoomID                uuid.UUID  `json:"room_id"`
	SenderParticipantID   *uuid.UUID `json:"sender_participant_id,omitempty"`
	MessageType           string     `json:"message_type"`
	Content               string     `json:"content"`
	CreatedAt             time.Time  `json:"created_at"`
	EditedAt              *time.Time `json:"edited_at,omitempty"`
	DeletedAt             *time.Time `json:"deleted_at,omitempty"`
	DeletedByParticipantID *uuid.UUID `json:"deleted_by_participant_id,omitempty"`
}

const (
	MessageTypeUser   = "user"
	MessageTypeSystem = "system"
)

