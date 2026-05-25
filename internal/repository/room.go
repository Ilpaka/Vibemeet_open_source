package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"
)

type RoomRepository interface {
	Create(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error)
	GetByLiveKitRoomName(ctx context.Context, name string) (*domain.Room, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error)
	Update(ctx context.Context, room *domain.Room) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateInvite(ctx context.Context, invite *domain.RoomInvite) error
	GetInviteByToken(ctx context.Context, token string) (*domain.RoomInvite, error)
	IncrementInviteUsage(ctx context.Context, inviteID uuid.UUID) error
	CreateParticipant(ctx context.Context, participant *domain.RoomParticipant) error
	GetParticipant(ctx context.Context, roomID, userID uuid.UUID) (*domain.RoomParticipant, error)
	GetParticipantByID(ctx context.Context, participantID uuid.UUID) (*domain.RoomParticipant, error)
	GetParticipantsByRoom(ctx context.Context, roomID uuid.UUID) ([]*domain.RoomParticipant, error)
	UpdateParticipant(ctx context.Context, participant *domain.RoomParticipant) error
	CreateWaitingRoomEntry(ctx context.Context, entry *domain.WaitingRoomEntry) error
	GetWaitingRoomEntries(ctx context.Context, roomID uuid.UUID, status string) ([]*domain.WaitingRoomEntry, error)
	UpdateWaitingRoomEntry(ctx context.Context, entry *domain.WaitingRoomEntry) error
}

type roomRepository struct {
	db  *pgxpool.Pool
	log logger.Logger
}

func NewRoomRepository(db *pgxpool.Pool, log logger.Logger) RoomRepository {
	return &roomRepository{db: db, log: log}
}

func (r *roomRepository) Create(ctx context.Context, room *domain.Room) error {
	query := `
		INSERT INTO rooms (id, livekit_room_name, host_user_id, title, description, status, 
		                  scheduled_start_at, scheduled_end_at, max_participants, waiting_room_enabled,
		                  is_locked, password_hash, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		room.ID, room.LiveKitRoomName, room.HostUserID, room.Title, room.Description, room.Status,
		room.ScheduledStartAt, room.ScheduledEndAt, room.MaxParticipants, room.WaitingRoomEnabled,
		room.IsLocked, room.PasswordHash, room.Settings, room.CreatedAt, room.UpdatedAt,
	).Scan(&room.CreatedAt, &room.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to create room", "error", err)
		return err
	}

	return nil
}

func (r *roomRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	query := `
		SELECT id, livekit_room_name, host_user_id, title, description, status,
		       scheduled_start_at, scheduled_end_at, actual_start_at, actual_end_at,
		       max_participants, waiting_room_enabled, is_locked, password_hash, settings,
		       created_at, updated_at
		FROM rooms
		WHERE id = $1
	`

	room := &domain.Room{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.LiveKitRoomName, &room.HostUserID, &room.Title, &room.Description, &room.Status,
		&room.ScheduledStartAt, &room.ScheduledEndAt, &room.ActualStartAt, &room.ActualEndAt,
		&room.MaxParticipants, &room.WaitingRoomEnabled, &room.IsLocked, &room.PasswordHash, &room.Settings,
		&room.CreatedAt, &room.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("room not found")
		}
		r.log.Error("Failed to get room by ID", "error", err)
		return nil, err
	}

	return room, nil
}

func (r *roomRepository) GetByLiveKitRoomName(ctx context.Context, name string) (*domain.Room, error) {
	query := `
		SELECT id, livekit_room_name, host_user_id, title, description, status,
		       scheduled_start_at, scheduled_end_at, actual_start_at, actual_end_at,
		       max_participants, waiting_room_enabled, is_locked, password_hash, settings,
		       created_at, updated_at
		FROM rooms
		WHERE livekit_room_name = $1
	`

	room := &domain.Room{}
	err := r.db.QueryRow(ctx, query, name).Scan(
		&room.ID, &room.LiveKitRoomName, &room.HostUserID, &room.Title, &room.Description, &room.Status,
		&room.ScheduledStartAt, &room.ScheduledEndAt, &room.ActualStartAt, &room.ActualEndAt,
		&room.MaxParticipants, &room.WaitingRoomEnabled, &room.IsLocked, &room.PasswordHash, &room.Settings,
		&room.CreatedAt, &room.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("room not found")
		}
		r.log.Error("Failed to get room by LiveKit name", "error", err)
		return nil, err
	}

	return room, nil
}

func (r *roomRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error) {
	query := `
		SELECT id, livekit_room_name, host_user_id, title, description, status,
		       scheduled_start_at, scheduled_end_at, actual_start_at, actual_end_at,
		       max_participants, waiting_room_enabled, is_locked, password_hash, settings,
		       created_at, updated_at
		FROM rooms
		WHERE host_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.log.Error("Failed to list rooms", "error", err)
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.Room
	for rows.Next() {
		room := &domain.Room{}
		err := rows.Scan(
			&room.ID, &room.LiveKitRoomName, &room.HostUserID, &room.Title, &room.Description, &room.Status,
			&room.ScheduledStartAt, &room.ScheduledEndAt, &room.ActualStartAt, &room.ActualEndAt,
			&room.MaxParticipants, &room.WaitingRoomEnabled, &room.IsLocked, &room.PasswordHash, &room.Settings,
			&room.CreatedAt, &room.UpdatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan room", "error", err)
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

func (r *roomRepository) Update(ctx context.Context, room *domain.Room) error {
	query := `
		UPDATE rooms
		SET title = $2, description = $3, status = $4, scheduled_start_at = $5,
		    scheduled_end_at = $6, actual_start_at = $7, actual_end_at = $8,
		    max_participants = $9, waiting_room_enabled = $10, is_locked = $11,
		    password_hash = $12, settings = $13, updated_at = $14
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		room.ID, room.Title, room.Description, room.Status,
		room.ScheduledStartAt, room.ScheduledEndAt, room.ActualStartAt, room.ActualEndAt,
		room.MaxParticipants, room.WaitingRoomEnabled, room.IsLocked,
		room.PasswordHash, room.Settings, time.Now(),
	).Scan(&room.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to update room", "error", err)
		return err
	}

	return nil
}

func (r *roomRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM rooms WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.log.Error("Failed to delete room", "error", err)
		return err
	}
	return nil
}

func (r *roomRepository) CreateInvite(ctx context.Context, invite *domain.RoomInvite) error {
	query := `
		INSERT INTO room_invites (id, room_id, created_by_user_id, link_token, label, expires_at, max_uses, used_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		invite.ID, invite.RoomID, invite.CreatedByUserID, invite.LinkToken,
		invite.Label, invite.ExpiresAt, invite.MaxUses, invite.UsedCount, invite.CreatedAt,
	)

	if err != nil {
		r.log.Error("Failed to create invite", "error", err)
		return err
	}

	return nil
}

func (r *roomRepository) GetInviteByToken(ctx context.Context, token string) (*domain.RoomInvite, error) {
	query := `
		SELECT id, room_id, created_by_user_id, link_token, label, expires_at, max_uses, used_count, created_at
		FROM room_invites
		WHERE link_token = $1
	`

	invite := &domain.RoomInvite{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&invite.ID, &invite.RoomID, &invite.CreatedByUserID, &invite.LinkToken,
		&invite.Label, &invite.ExpiresAt, &invite.MaxUses, &invite.UsedCount, &invite.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invite not found")
		}
		r.log.Error("Failed to get invite", "error", err)
		return nil, err
	}

	return invite, nil
}

func (r *roomRepository) IncrementInviteUsage(ctx context.Context, inviteID uuid.UUID) error {
	query := `UPDATE room_invites SET used_count = used_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, inviteID)
	if err != nil {
		r.log.Error("Failed to increment invite usage", "error", err)
		return err
	}
	return nil
}

func (r *roomRepository) CreateParticipant(ctx context.Context, participant *domain.RoomParticipant) error {
	query := `
		INSERT INTO room_participants (id, room_id, user_id, role, display_name, livekit_sid,
		                              joined_at, initial_muted, client_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		participant.ID, participant.RoomID, participant.UserID, participant.Role,
		participant.DisplayName, participant.LiveKitSID, participant.JoinedAt,
		participant.InitialMuted, participant.ClientIP, participant.UserAgent,
	)

	if err != nil {
		r.log.Error("Failed to create participant", "error", err)
		return err
	}

	return nil
}

func (r *roomRepository) GetParticipant(ctx context.Context, roomID, userID uuid.UUID) (*domain.RoomParticipant, error) {
	query := `
		SELECT id, room_id, user_id, role, display_name, livekit_sid, joined_at, left_at,
		       leave_reason, is_kicked, initial_muted, client_ip, user_agent
		FROM room_participants
		WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL
		ORDER BY joined_at DESC
		LIMIT 1
	`

	participant := &domain.RoomParticipant{}
	var leftAt sql.NullTime
	err := r.db.QueryRow(ctx, query, roomID, userID).Scan(
		&participant.ID, &participant.RoomID, &participant.UserID, &participant.Role,
		&participant.DisplayName, &participant.LiveKitSID, &participant.JoinedAt, &leftAt,
		&participant.LeaveReason, &participant.IsKicked, &participant.InitialMuted,
		&participant.ClientIP, &participant.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("participant not found")
		}
		r.log.Error("Failed to get participant", "error", err)
		return nil, err
	}

	if leftAt.Valid {
		participant.LeftAt = &leftAt.Time
	}

	return participant, nil
}

func (r *roomRepository) GetParticipantByID(ctx context.Context, participantID uuid.UUID) (*domain.RoomParticipant, error) {
	query := `
		SELECT id, room_id, user_id, role, display_name, livekit_sid, joined_at, left_at,
		       leave_reason, is_kicked, initial_muted, client_ip, user_agent
		FROM room_participants
		WHERE id = $1
	`

	participant := &domain.RoomParticipant{}
	var leftAt sql.NullTime
	err := r.db.QueryRow(ctx, query, participantID).Scan(
		&participant.ID, &participant.RoomID, &participant.UserID, &participant.Role,
		&participant.DisplayName, &participant.LiveKitSID, &participant.JoinedAt, &leftAt,
		&participant.LeaveReason, &participant.IsKicked, &participant.InitialMuted,
		&participant.ClientIP, &participant.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("participant not found")
		}
		r.log.Error("Failed to get participant by ID", "error", err)
		return nil, err
	}

	if leftAt.Valid {
		participant.LeftAt = &leftAt.Time
	}

	return participant, nil
}

func (r *roomRepository) GetParticipantsByRoom(ctx context.Context, roomID uuid.UUID) ([]*domain.RoomParticipant, error) {
	query := `
		SELECT id, room_id, user_id, role, display_name, livekit_sid, joined_at, left_at,
		       leave_reason, is_kicked, initial_muted, client_ip, user_agent
		FROM room_participants
		WHERE room_id = $1 AND left_at IS NULL
		ORDER BY joined_at ASC
	`

	rows, err := r.db.Query(ctx, query, roomID)
	if err != nil {
		r.log.Error("Failed to get participants", "error", err)
		return nil, err
	}
	defer rows.Close()

	var participants []*domain.RoomParticipant
	for rows.Next() {
		participant := &domain.RoomParticipant{}
		var leftAt sql.NullTime
		err := rows.Scan(
			&participant.ID, &participant.RoomID, &participant.UserID, &participant.Role,
			&participant.DisplayName, &participant.LiveKitSID, &participant.JoinedAt, &leftAt,
			&participant.LeaveReason, &participant.IsKicked, &participant.InitialMuted,
			&participant.ClientIP, &participant.UserAgent,
		)
		if err != nil {
			r.log.Error("Failed to scan participant", "error", err)
			return nil, err
		}
		if leftAt.Valid {
			participant.LeftAt = &leftAt.Time
		}
		participants = append(participants, participant)
	}

	return participants, nil
}

func (r *roomRepository) UpdateParticipant(ctx context.Context, participant *domain.RoomParticipant) error {
	query := `
		UPDATE room_participants
		SET left_at = $3, leave_reason = $4, is_kicked = $5
		WHERE id = $1 AND room_id = $2
	`

	_, err := r.db.Exec(ctx, query,
		participant.ID, participant.RoomID, participant.LeftAt,
		participant.LeaveReason, participant.IsKicked,
	)

	if err != nil {
		r.log.Error("Failed to update participant", "error", err)
		return err
	}

	return nil
}

func (r *roomRepository) CreateWaitingRoomEntry(ctx context.Context, entry *domain.WaitingRoomEntry) error {
	query := `
		INSERT INTO waiting_room_entries (id, room_id, user_id, display_name, status, requested_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		entry.ID, entry.RoomID, entry.UserID, entry.DisplayName, entry.Status, entry.RequestedAt,
	)

	if err != nil {
		r.log.Error("Failed to create waiting room entry", "error", err)
		return err
	}

	return nil
}

func (r *roomRepository) GetWaitingRoomEntries(ctx context.Context, roomID uuid.UUID, status string) ([]*domain.WaitingRoomEntry, error) {
	query := `
		SELECT id, room_id, user_id, display_name, status, requested_at, decided_at, decided_by_user_id, reason
		FROM waiting_room_entries
		WHERE room_id = $1 AND status = $2
		ORDER BY requested_at ASC
	`

	rows, err := r.db.Query(ctx, query, roomID, status)
	if err != nil {
		r.log.Error("Failed to get waiting room entries", "error", err)
		return nil, err
	}
	defer rows.Close()

	var entries []*domain.WaitingRoomEntry
	for rows.Next() {
		entry := &domain.WaitingRoomEntry{}
		var decidedAt sql.NullTime
		err := rows.Scan(
			&entry.ID, &entry.RoomID, &entry.UserID, &entry.DisplayName, &entry.Status,
			&entry.RequestedAt, &decidedAt, &entry.DecidedByUserID, &entry.Reason,
		)
		if err != nil {
			r.log.Error("Failed to scan waiting room entry", "error", err)
			return nil, err
		}
		if decidedAt.Valid {
			entry.DecidedAt = &decidedAt.Time
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (r *roomRepository) UpdateWaitingRoomEntry(ctx context.Context, entry *domain.WaitingRoomEntry) error {
	query := `
		UPDATE waiting_room_entries
		SET status = $2, decided_at = $3, decided_by_user_id = $4, reason = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		entry.ID, entry.Status, entry.DecidedAt, entry.DecidedByUserID, entry.Reason,
	)

	if err != nil {
		r.log.Error("Failed to update waiting room entry", "error", err)
		return err
	}

	return nil
}
