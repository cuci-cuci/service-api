package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type SyncMonitorService struct {
	db *pgxpool.Pool
}

func NewSyncMonitorService(db *pgxpool.Pool) *SyncMonitorService {
	return &SyncMonitorService{db: db}
}

type GlobalSyncHealth struct {
	TotalSessions    int        `json:"total_sessions"`
	TotalDevices     int        `json:"total_devices"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty"`
	CompletedCount   int        `json:"completed_count"`
	FailedCount      int        `json:"failed_count"`
	AvgTransactions  float64    `json:"avg_transactions_per_session"`
}

func (s *SyncMonitorService) GetGlobalHealth(ctx context.Context) (*GlobalSyncHealth, error) {
	q := middleware.GetQuerier(ctx, s.db)

	health := &GlobalSyncHealth{}

	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM sync_sessions").Scan(&health.TotalSessions)
	if err != nil {
		slog.Error("failed to count sync sessions", "error", err)
		return nil, apperror.Internal("failed to count sync sessions", err)
	}

	err = q.QueryRow(ctx,
		"SELECT COUNT(DISTINCT outlet_id) FROM sync_sessions").Scan(&health.TotalDevices)
	if err != nil {
		slog.Error("failed to count sync devices", "error", err)
		return nil, apperror.Internal("failed to count sync devices", err)
	}

	err = q.QueryRow(ctx,
		"SELECT MAX(completed_at) FROM sync_sessions WHERE status = 'completed'").Scan(&health.LastSyncAt)
	if err != nil {
		// Ignore — no rows is fine
		slog.Debug("no completed sync sessions found", "error", err)
	}

	err = q.QueryRow(ctx,
		"SELECT COUNT(*) FROM sync_sessions WHERE status = 'completed'").Scan(&health.CompletedCount)
	if err != nil {
		slog.Error("failed to count completed sessions", "error", err)
		return nil, apperror.Internal("failed to count completed sessions", err)
	}

	err = q.QueryRow(ctx,
		"SELECT COUNT(*) FROM sync_sessions WHERE status = 'failed'").Scan(&health.FailedCount)
	if err != nil {
		slog.Error("failed to count failed sessions", "error", err)
		return nil, apperror.Internal("failed to count failed sessions", err)
	}

	err = q.QueryRow(ctx,
		"SELECT COALESCE(AVG(transaction_count), 0) FROM sync_sessions").Scan(&health.AvgTransactions)
	if err != nil {
		slog.Error("failed to get avg transactions", "error", err)
		return nil, apperror.Internal("failed to get avg transactions", err)
	}

	return health, nil
}

type OutletSyncHealth struct {
	OutletID       uuid.UUID  `json:"outlet_id"`
	OutletName     string     `json:"outlet_name"`
	TotalSessions  int        `json:"total_sessions"`
	FailedSessions int        `json:"failed_sessions"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
}

func (s *SyncMonitorService) GetOutletHealth(ctx context.Context) ([]OutletSyncHealth, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT o.id, o.name,
			COUNT(ss.id) as total_sessions,
			COUNT(ss.id) FILTER (WHERE ss.status = 'failed') as failed_sessions,
			MAX(ss.completed_at) as last_sync_at
		FROM outlets o
		LEFT JOIN sync_sessions ss ON o.id = ss.outlet_id
		WHERE o.is_active = true
		GROUP BY o.id, o.name
		ORDER BY o.name`)
	if err != nil {
		slog.Error("failed to query outlet health", "error", err)
		return nil, apperror.Internal("failed to query outlet health", err)
	}
	defer rows.Close()

	var results []OutletSyncHealth
	for rows.Next() {
		var h OutletSyncHealth
		if err := rows.Scan(&h.OutletID, &h.OutletName, &h.TotalSessions, &h.FailedSessions, &h.LastSyncAt); err != nil {
			return nil, apperror.Internal("failed to scan outlet health", err)
		}
		results = append(results, h)
	}

	if results == nil {
		results = []OutletSyncHealth{}
	}

	return results, nil
}

func (s *SyncMonitorService) ListSessions(ctx context.Context, params pagination.Params) ([]domain.SyncSession, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM sync_sessions").Scan(&total)
	if err != nil {
		slog.Error("failed to count sync sessions", "error", err)
		return nil, 0, apperror.Internal("failed to count sync sessions", err)
	}

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, outlet_id, direction, status, transaction_count, error_message, started_at, completed_at
		 FROM sync_sessions
		 ORDER BY started_at DESC
		 LIMIT $1 OFFSET $2`,
		params.PerPage, params.Offset())
	if err != nil {
		slog.Error("failed to list sync sessions", "error", err)
		return nil, 0, apperror.Internal("failed to list sync sessions", err)
	}
	defer rows.Close()

	var sessions []domain.SyncSession
	for rows.Next() {
		var ss domain.SyncSession
		if err := rows.Scan(&ss.ID, &ss.TenantID, &ss.OutletID, &ss.Direction, &ss.Status,
			&ss.TransactionCount, &ss.ErrorMessage, &ss.StartedAt, &ss.CompletedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan sync session", err)
		}
		sessions = append(sessions, ss)
	}

	if sessions == nil {
		sessions = []domain.SyncSession{}
	}

	return sessions, total, nil
}
