package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vibemeet/internal/domain"
	"vibemeet/pkg/logger"
)

type AuditRepository interface {
	CreateLog(ctx context.Context, log *domain.AuditLog) error
}

type auditRepository struct {
	db  *pgxpool.Pool
	log logger.Logger
}

func NewAuditRepository(db *pgxpool.Pool, log logger.Logger) AuditRepository {
	return &auditRepository{db: db, log: log}
}

func (r *auditRepository) CreateLog(ctx context.Context, auditLog *domain.AuditLog) error {
	query := `
		INSERT INTO audit_log (event_time, actor_user_id, actor_role, room_id, event_type, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		auditLog.EventTime, auditLog.ActorUserID, auditLog.ActorRole,
		auditLog.RoomID, auditLog.EventType, auditLog.Payload,
	).Scan(&auditLog.ID)

	if err != nil {
		r.log.Error("Failed to create audit log", "error", err)
		return err
	}

	return nil
}
