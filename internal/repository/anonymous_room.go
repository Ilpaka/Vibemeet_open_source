package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnonymousRoomRepository interface {
	Create(ctx context.Context, room *domain.AnonymousRoom) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AnonymousRoom, error)
	GetByLiveKitRoomName(ctx context.Context, name string) (*domain.AnonymousRoom, error)
	Update(ctx context.Context, room *domain.AnonymousRoom) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateParticipant(ctx context.Context, participant *domain.AnonymousParticipant) error
	GetParticipant(ctx context.Context, roomID uuid.UUID, participantID string) (*domain.AnonymousParticipant, error)
	GetParticipantByID(ctx context.Context, participantID uuid.UUID) (*domain.AnonymousParticipant, error)
	GetParticipantsByRoom(ctx context.Context, roomID uuid.UUID) ([]*domain.AnonymousParticipant, error)
	UpdateParticipant(ctx context.Context, participant *domain.AnonymousParticipant) error
	// Methods for leave functionality
	MarkParticipantLeft(ctx context.Context, roomID uuid.UUID, participantID string) error
	GetActiveParticipantCount(ctx context.Context, roomID uuid.UUID) (int, error)
	SetRoomStatus(ctx context.Context, roomID uuid.UUID, status string) error
	// Cleanup of inactive rooms
	CleanupInactiveRooms(ctx context.Context, inactiveDuration time.Duration) (int, error)
}

type anonymousRoomRepository struct {
	db  *pgxpool.Pool
	log logger.Logger
}

func NewAnonymousRoomRepository(db *pgxpool.Pool, log logger.Logger) AnonymousRoomRepository {
	return &anonymousRoomRepository{
		db:  db,
		log: log,
	}
}

func (r *anonymousRoomRepository) Create(ctx context.Context, room *domain.AnonymousRoom) error {
	query := `
		INSERT INTO anonymous_rooms (
			id, livekit_room_name, title, description, status, 
			max_participants, created_at, updated_at, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		room.ID, room.LiveKitRoomName, room.Title, room.Description, room.Status,
		room.MaxParticipants, room.CreatedAt, room.UpdatedAt, room.ExpiresAt,
	).Scan(&room.CreatedAt, &room.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to create anonymous room", "error", err)
		return err
	}

	return nil
}

func (r *anonymousRoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AnonymousRoom, error) {
	query := `
		SELECT id, livekit_room_name, title, description, status,
		       max_participants, created_at, updated_at, expires_at
		FROM anonymous_rooms
		WHERE id = $1
	`

	room := &domain.AnonymousRoom{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.LiveKitRoomName, &room.Title, &room.Description, &room.Status,
		&room.MaxParticipants, &room.CreatedAt, &room.UpdatedAt, &room.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("room not found")
		}
		r.log.Error("Failed to get anonymous room by ID", "error", err)
		return nil, err
	}

	return room, nil
}

func (r *anonymousRoomRepository) GetByLiveKitRoomName(ctx context.Context, name string) (*domain.AnonymousRoom, error) {
	query := `
		SELECT id, livekit_room_name, title, description, status,
		       max_participants, created_at, updated_at, expires_at
		FROM anonymous_rooms
		WHERE livekit_room_name = $1
	`

	room := &domain.AnonymousRoom{}
	err := r.db.QueryRow(ctx, query, name).Scan(
		&room.ID, &room.LiveKitRoomName, &room.Title, &room.Description, &room.Status,
		&room.MaxParticipants, &room.CreatedAt, &room.UpdatedAt, &room.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("room not found")
		}
		r.log.Error("Failed to get anonymous room by LiveKit name", "error", err)
		return nil, err
	}

	return room, nil
}

func (r *anonymousRoomRepository) Update(ctx context.Context, room *domain.AnonymousRoom) error {
	query := `
		UPDATE anonymous_rooms
		SET title = $2, description = $3, status = $4,
		    max_participants = $5, expires_at = $6, updated_at = $7
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		room.ID, room.Title, room.Description, room.Status,
		room.MaxParticipants, room.ExpiresAt, time.Now(),
	).Scan(&room.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to update anonymous room", "error", err)
		return err
	}

	return nil
}

func (r *anonymousRoomRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM anonymous_rooms WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.log.Error("Failed to delete anonymous room", "error", err)
		return err
	}
	return nil
}

func (r *anonymousRoomRepository) CreateParticipant(ctx context.Context, participant *domain.AnonymousParticipant) error {
	query := `
		INSERT INTO anonymous_participants (
			id, room_id, participant_id, display_name, role,
			livekit_sid, joined_at, client_ip, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		participant.ID, participant.RoomID, participant.ParticipantID, participant.DisplayName,
		participant.Role, participant.LiveKitSID, participant.JoinedAt,
		participant.ClientIP, participant.UserAgent,
	)

	if err != nil {
		r.log.Error("Failed to create anonymous participant", "error", err)
		return err
	}

	return nil
}

func (r *anonymousRoomRepository) GetParticipant(ctx context.Context, roomID uuid.UUID, participantID string) (*domain.AnonymousParticipant, error) {
	query := `
		SELECT id, room_id, participant_id, display_name, role,
		       livekit_sid, joined_at, left_at, client_ip, user_agent
		FROM anonymous_participants
		WHERE room_id = $1 AND participant_id = $2 AND left_at IS NULL
		ORDER BY joined_at DESC
		LIMIT 1
	`

	participant := &domain.AnonymousParticipant{}
	var leftAt sql.NullTime
	err := r.db.QueryRow(ctx, query, roomID, participantID).Scan(
		&participant.ID, &participant.RoomID, &participant.ParticipantID, &participant.DisplayName,
		&participant.Role, &participant.LiveKitSID, &participant.JoinedAt, &leftAt,
		&participant.ClientIP, &participant.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("participant not found")
		}
		r.log.Error("Failed to get anonymous participant", "error", err)
		return nil, err
	}

	if leftAt.Valid {
		participant.LeftAt = &leftAt.Time
	}

	return participant, nil
}

func (r *anonymousRoomRepository) GetParticipantByID(ctx context.Context, participantID uuid.UUID) (*domain.AnonymousParticipant, error) {
	query := `
		SELECT id, room_id, participant_id, display_name, role,
		       livekit_sid, joined_at, left_at, client_ip, user_agent
		FROM anonymous_participants
		WHERE id = $1
	`

	participant := &domain.AnonymousParticipant{}
	var leftAt sql.NullTime
	err := r.db.QueryRow(ctx, query, participantID).Scan(
		&participant.ID, &participant.RoomID, &participant.ParticipantID, &participant.DisplayName,
		&participant.Role, &participant.LiveKitSID, &participant.JoinedAt, &leftAt,
		&participant.ClientIP, &participant.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("participant not found")
		}
		r.log.Error("Failed to get anonymous participant by ID", "error", err)
		return nil, err
	}

	if leftAt.Valid {
		participant.LeftAt = &leftAt.Time
	}

	return participant, nil
}

func (r *anonymousRoomRepository) GetParticipantsByRoom(ctx context.Context, roomID uuid.UUID) ([]*domain.AnonymousParticipant, error) {
	query := `
		SELECT id, room_id, participant_id, display_name, role,
		       livekit_sid, joined_at, left_at, client_ip, user_agent
		FROM anonymous_participants
		WHERE room_id = $1 AND left_at IS NULL
		ORDER BY joined_at ASC
	`

	rows, err := r.db.Query(ctx, query, roomID)
	if err != nil {
		r.log.Error("Failed to get anonymous participants", "error", err)
		return nil, err
	}
	defer rows.Close()

	var participants []*domain.AnonymousParticipant
	for rows.Next() {
		participant := &domain.AnonymousParticipant{}
		var leftAt sql.NullTime
		err := rows.Scan(
			&participant.ID, &participant.RoomID, &participant.ParticipantID, &participant.DisplayName,
			&participant.Role, &participant.LiveKitSID, &participant.JoinedAt, &leftAt,
			&participant.ClientIP, &participant.UserAgent,
		)
		if err != nil {
			r.log.Error("Failed to scan anonymous participant", "error", err)
			return nil, err
		}
		if leftAt.Valid {
			participant.LeftAt = &leftAt.Time
		}
		participants = append(participants, participant)
	}

	return participants, nil
}

func (r *anonymousRoomRepository) UpdateParticipant(ctx context.Context, participant *domain.AnonymousParticipant) error {
	query := `
		UPDATE anonymous_participants
		SET left_at = $3
		WHERE id = $1 AND room_id = $2
	`

	_, err := r.db.Exec(ctx, query,
		participant.ID, participant.RoomID, participant.LeftAt,
	)

	if err != nil {
		r.log.Error("Failed to update anonymous participant", "error", err)
		return err
	}

	return nil
}

func (r *anonymousRoomRepository) CleanupInactiveRooms(ctx context.Context, inactiveDuration time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-inactiveDuration)

	query := `
		DELETE FROM anonymous_rooms
		WHERE (expires_at IS NOT NULL AND expires_at < NOW())
		   OR (status = 'ended' AND updated_at < $1)
	`

	result, err := r.db.Exec(ctx, query, cutoffTime)
	if err != nil {
		r.log.Error("Failed to cleanup inactive rooms", "error", err)
		return 0, err
	}

	deletedCount := int(result.RowsAffected())
	r.log.Info("Cleaned up inactive rooms", "count", deletedCount)

	return deletedCount, nil
}

func (r *anonymousRoomRepository) MarkParticipantLeft(ctx context.Context, roomID uuid.UUID, participantID string) error {
	query := `
		UPDATE anonymous_participants
		SET left_at = $3
		WHERE room_id = $1 AND participant_id = $2 AND left_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, roomID, participantID, time.Now())
	if err != nil {
		r.log.Error("Failed to mark participant as left", "error", err, "room_id", roomID, "participant_id", participantID)
		return err
	}

	return nil
}

func (r *anonymousRoomRepository) GetActiveParticipantCount(ctx context.Context, roomID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM anonymous_participants
		WHERE room_id = $1 AND left_at IS NULL
	`

	var count int
	err := r.db.QueryRow(ctx, query, roomID).Scan(&count)
	if err != nil {
		r.log.Error("Failed to get active participant count", "error", err, "room_id", roomID)
		return 0, err
	}

	return count, nil
}

func (r *anonymousRoomRepository) SetRoomStatus(ctx context.Context, roomID uuid.UUID, status string) error {
	query := `
		UPDATE anonymous_rooms
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, roomID, status, time.Now())
	if err != nil {
		r.log.Error("Failed to set room status", "error", err, "room_id", roomID, "status", status)
		return err
	}

	r.log.Info("Room status updated", "room_id", roomID, "status", status)
	return nil
}
