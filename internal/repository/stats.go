package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"
)

type StatsRepository interface {
	CreateParticipantStats(ctx context.Context, stats *domain.ParticipantStats) error
	GetParticipantStats(ctx context.Context, participantID uuid.UUID) (*domain.ParticipantStats, error)
	GetRoomStats(ctx context.Context, roomID uuid.UUID) ([]*domain.ParticipantStats, error)
}

type statsRepository struct {
	db  *pgxpool.Pool
	log logger.Logger
}

func NewStatsRepository(db *pgxpool.Pool, log logger.Logger) StatsRepository {
	return &statsRepository{db: db, log: log}
}

func (r *statsRepository) CreateParticipantStats(ctx context.Context, stats *domain.ParticipantStats) error {
	query := `
		INSERT INTO participant_stats (room_participant_id, avg_rtt_ms, max_rtt_ms, avg_jitter_ms,
		                              packet_loss_up_pct, packet_loss_down_pct, avg_bitrate_kbps,
		                              network_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	
	err := r.db.QueryRow(ctx, query,
		stats.RoomParticipantID, stats.AvgRTTMs, stats.MaxRTTMs, stats.AvgJitterMs,
		stats.PacketLossUpPercent, stats.PacketLossDownPercent, stats.AvgBitrateKbps,
		stats.NetworkScore, stats.CreatedAt,
	).Scan(&stats.ID, &stats.CreatedAt)
	
	if err != nil {
		r.log.Error("Failed to create participant stats", "error", err)
		return err
	}
	
	return nil
}

func (r *statsRepository) GetParticipantStats(ctx context.Context, participantID uuid.UUID) (*domain.ParticipantStats, error) {
	query := `
		SELECT id, room_participant_id, avg_rtt_ms, max_rtt_ms, avg_jitter_ms,
		       packet_loss_up_pct, packet_loss_down_pct, avg_bitrate_kbps, network_score, created_at
		FROM participant_stats
		WHERE room_participant_id = $1
	`
	
	stats := &domain.ParticipantStats{}
	err := r.db.QueryRow(ctx, query, participantID).Scan(
		&stats.ID, &stats.RoomParticipantID, &stats.AvgRTTMs, &stats.MaxRTTMs, &stats.AvgJitterMs,
		&stats.PacketLossUpPercent, &stats.PacketLossDownPercent, &stats.AvgBitrateKbps,
		&stats.NetworkScore, &stats.CreatedAt,
	)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("stats not found")
		}
		r.log.Error("Failed to get participant stats", "error", err)
		return nil, err
	}
	
	return stats, nil
}

func (r *statsRepository) GetRoomStats(ctx context.Context, roomID uuid.UUID) ([]*domain.ParticipantStats, error) {
	query := `
		SELECT ps.id, ps.room_participant_id, ps.avg_rtt_ms, ps.max_rtt_ms, ps.avg_jitter_ms,
		       ps.packet_loss_up_pct, ps.packet_loss_down_pct, ps.avg_bitrate_kbps, ps.network_score, ps.created_at
		FROM participant_stats ps
		JOIN room_participants rp ON ps.room_participant_id = rp.id
		WHERE rp.room_id = $1
		ORDER BY ps.created_at DESC
	`
	
	rows, err := r.db.Query(ctx, query, roomID)
	if err != nil {
		r.log.Error("Failed to get room stats", "error", err)
		return nil, err
	}
	defer rows.Close()
	
	var stats []*domain.ParticipantStats
	for rows.Next() {
		s := &domain.ParticipantStats{}
		err := rows.Scan(
			&s.ID, &s.RoomParticipantID, &s.AvgRTTMs, &s.MaxRTTMs, &s.AvgJitterMs,
			&s.PacketLossUpPercent, &s.PacketLossDownPercent, &s.AvgBitrateKbps,
			&s.NetworkScore, &s.CreatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan stats", "error", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	
	return stats, nil
}

