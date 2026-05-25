package domain

import (
	"time"

	"github.com/google/uuid"
)

type ParticipantStats struct {
	ID                  int64     `json:"id"`
	RoomParticipantID   uuid.UUID `json:"room_participant_id"`
	AvgRTTMs            *float64  `json:"avg_rtt_ms,omitempty"`
	MaxRTTMs            *float64  `json:"max_rtt_ms,omitempty"`
	AvgJitterMs         *float64  `json:"avg_jitter_ms,omitempty"`
	PacketLossUpPercent *float64  `json:"packet_loss_up_percent,omitempty"`
	PacketLossDownPercent *float64  `json:"packet_loss_down_percent,omitempty"`
	AvgBitrateKbps      *float64  `json:"avg_bitrate_kbps,omitempty"`
	NetworkScore        *int      `json:"network_score,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

