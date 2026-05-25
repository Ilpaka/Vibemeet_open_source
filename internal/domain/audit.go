package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID          int64                  `json:"id"`
	EventTime   time.Time              `json:"event_time"`
	ActorUserID *uuid.UUID             `json:"actor_user_id,omitempty"`
	ActorRole   string                 `json:"actor_role"`
	RoomID      *uuid.UUID             `json:"room_id,omitempty"`
	EventType   string                 `json:"event_type"`
	Payload     map[string]interface{} `json:"payload"`
}

const (
	ActorRoleUser          = "user"
	ActorRoleHost          = "host"
	ActorRoleTechnicalAdmin = "technical_admin"
	ActorRoleSystem        = "system"
)

const (
	EventTypeRoomCreated     = "ROOM_CREATED"
	EventTypeRoomUpdated     = "ROOM_UPDATED"
	EventTypeRoomDeleted     = "ROOM_DELETED"
	EventTypeRoomJoined      = "ROOM_JOINED"
	EventTypeRoomLeft        = "ROOM_LEFT"
	EventTypeUserKicked      = "USER_KICKED"
	EventTypeRoomLocked      = "ROOM_LOCKED"
	EventTypeRoomUnlocked    = "ROOM_UNLOCKED"
	EventTypeWaitingRoomApproved = "WAITING_ROOM_APPROVED"
	EventTypeWaitingRoomRejected = "WAITING_ROOM_REJECTED"
)

