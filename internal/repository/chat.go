package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"
)

type ChatRepository interface {
	CreateMessage(ctx context.Context, message *domain.ChatMessage) error
	GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*domain.ChatMessage, error)
	GetMessageByID(ctx context.Context, messageID int64) (*domain.ChatMessage, error)
	UpdateMessage(ctx context.Context, message *domain.ChatMessage) error
	DeleteMessage(ctx context.Context, messageID int64, deletedByParticipantID uuid.UUID) error
}

type chatRepository struct {
	db  *pgxpool.Pool
	log logger.Logger
}

func NewChatRepository(db *pgxpool.Pool, log logger.Logger) ChatRepository {
	return &chatRepository{db: db, log: log}
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *domain.ChatMessage) error {
	query := `
		INSERT INTO chat_messages (room_id, sender_participant_id, message_type, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		message.RoomID, message.SenderParticipantID, message.MessageType,
		message.Content, message.CreatedAt,
	).Scan(&message.ID, &message.CreatedAt)

	if err != nil {
		r.log.Error("Failed to create message", "error", err)
		return err
	}

	return nil
}

func (r *chatRepository) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*domain.ChatMessage, error) {
	query := `
		SELECT id, room_id, sender_participant_id, message_type, content, created_at, edited_at, deleted_at, deleted_by_participant_id
		FROM chat_messages
		WHERE room_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, roomID, limit, offset)
	if err != nil {
		r.log.Error("Failed to get messages", "error", err)
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.ChatMessage
	for rows.Next() {
		message := &domain.ChatMessage{}
		var editedAt, deletedAt sql.NullTime
		err := rows.Scan(
			&message.ID, &message.RoomID, &message.SenderParticipantID, &message.MessageType,
			&message.Content, &message.CreatedAt, &editedAt, &deletedAt, &message.DeletedByParticipantID,
		)
		if err != nil {
			r.log.Error("Failed to scan message", "error", err)
			return nil, err
		}
		if editedAt.Valid {
			message.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			message.DeletedAt = &deletedAt.Time
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func (r *chatRepository) GetMessageByID(ctx context.Context, messageID int64) (*domain.ChatMessage, error) {
	query := `
		SELECT id, room_id, sender_participant_id, message_type, content, created_at, edited_at, deleted_at, deleted_by_participant_id
		FROM chat_messages
		WHERE id = $1
	`

	message := &domain.ChatMessage{}
	var editedAt, deletedAt sql.NullTime
	err := r.db.QueryRow(ctx, query, messageID).Scan(
		&message.ID, &message.RoomID, &message.SenderParticipantID, &message.MessageType,
		&message.Content, &message.CreatedAt, &editedAt, &deletedAt, &message.DeletedByParticipantID,
	)

	if err != nil {
		r.log.Error("Failed to get message", "error", err)
		return nil, err
	}

	if editedAt.Valid {
		message.EditedAt = &editedAt.Time
	}
	if deletedAt.Valid {
		message.DeletedAt = &deletedAt.Time
	}

	return message, nil
}

func (r *chatRepository) UpdateMessage(ctx context.Context, message *domain.ChatMessage) error {
	query := `
		UPDATE chat_messages
		SET content = $2, edited_at = $3
		WHERE id = $1
		RETURNING edited_at
	`

	err := r.db.QueryRow(ctx, query, message.ID, message.Content, time.Now()).Scan(&message.EditedAt)
	if err != nil {
		r.log.Error("Failed to update message", "error", err)
		return err
	}

	return nil
}

func (r *chatRepository) DeleteMessage(ctx context.Context, messageID int64, deletedByParticipantID uuid.UUID) error {
	query := `
		UPDATE chat_messages
		SET deleted_at = $2, deleted_by_participant_id = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, messageID, time.Now(), deletedByParticipantID)
	if err != nil {
		r.log.Error("Failed to delete message", "error", err)
		return err
	}

	return nil
}
