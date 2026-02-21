package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type AuditService struct {
	db *pgxpool.Pool
}

func NewAuditService(db *pgxpool.Pool) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, entityType, entityID string, oldValue, newValue any, tenantID *uuid.UUID) {
	var oldJSON, newJSON json.RawMessage

	if oldValue != nil {
		data, err := json.Marshal(oldValue)
		if err == nil {
			oldJSON = data
		}
	}
	if newValue != nil {
		data, err := json.Marshal(newValue)
		if err == nil {
			newJSON = data
		}
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO audit_logs (id, actor_id, actor_name, action, entity_type, entity_id, old_value, new_value, tenant_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		uuid.New(), actorID, actorName, action, entityType, entityID, oldJSON, newJSON, tenantID, time.Now())
	if err != nil {
		slog.Error("failed to write audit log", "error", err, "action", action, "entity_type", entityType)
	}
}

func (s *AuditService) ListLogs(ctx context.Context, tenantID *uuid.UUID, params pagination.Params) ([]domain.AuditLog, int, error) {
	var total int
	var args []any
	countQuery := "SELECT COUNT(*) FROM audit_logs"
	listQuery := `SELECT id, actor_id, actor_name, action, entity_type, entity_id, old_value, new_value, tenant_id, created_at FROM audit_logs`

	if tenantID != nil {
		countQuery += " WHERE tenant_id = $1"
		listQuery += " WHERE tenant_id = $1"
		args = append(args, *tenantID)
	}

	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count audit logs", err)
	}

	listQuery += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, params.PerPage, params.Offset())

	rows, err := s.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list audit logs", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorName, &l.Action, &l.EntityType, &l.EntityID,
			&l.OldValue, &l.NewValue, &l.TenantID, &l.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan audit log", err)
		}
		logs = append(logs, l)
	}

	if logs == nil {
		logs = []domain.AuditLog{}
	}

	return logs, total, nil
}

