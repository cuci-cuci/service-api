package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type SyncService struct {
	db            *pgxpool.Pool
	configService *ConfigService
	templateSvc   *ServiceTemplateService
}

func NewSyncService(db *pgxpool.Pool, configService *ConfigService, templateSvc *ServiceTemplateService) *SyncService {
	return &SyncService{db: db, configService: configService, templateSvc: templateSvc}
}

func (s *SyncService) Upload(ctx context.Context, tenantID uuid.UUID, outletID uuid.UUID, transactions []domain.Transaction) (*domain.SyncUploadResult, error) {
	q := middleware.GetQuerier(ctx, s.db)
	startedAt := time.Now()

	// FIX 7: Use pgx.Batch for bulk inserts instead of individual Exec calls in a loop.
	batch := &pgx.Batch{}
	now := time.Now()
	for _, tx := range transactions {
		batch.Queue(
			`INSERT INTO transactions (id, tenant_id, outlet_id, local_order_number, customer_name,
			 items, subtotal, discount_amount, tax_amount, total_amount, payment_status, payments,
			 status, config_version_id, notes, created_by, created_at, synced_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
			 ON CONFLICT (id) DO NOTHING`,
			tx.ID, tenantID, outletID, tx.LocalOrderNumber, tx.CustomerName,
			tx.Items, tx.Subtotal, tx.DiscountAmount, tx.TaxAmount, tx.TotalAmount,
			tx.PaymentStatus, tx.Payments, tx.Status, tx.ConfigVersionID, tx.Notes,
			tx.CreatedBy, tx.CreatedAt, now)
	}

	br := q.SendBatch(ctx, batch)
	inserted := 0
	for range transactions {
		ct, err := br.Exec()
		if err != nil {
			slog.Error("failed to insert transaction in batch", "error", err)
			continue
		}
		if ct.RowsAffected() > 0 {
			inserted++
		}
	}
	if err := br.Close(); err != nil {
		slog.Error("failed to close batch", "error", err)
	}

	// Record sync session
	completedAt := time.Now()
	_, err := q.Exec(ctx,
		`INSERT INTO sync_sessions (id, tenant_id, outlet_id, direction, status, transaction_count, started_at, completed_at)
		 VALUES ($1, $2, $3, 'upload', 'completed', $4, $5, $6)`,
		uuid.New(), tenantID, outletID, len(transactions), startedAt, completedAt)
	if err != nil {
		slog.Error("failed to record sync session", "error", err)
	}

	return &domain.SyncUploadResult{
		Received: len(transactions),
		Inserted: inserted,
		Skipped:  len(transactions) - inserted,
	}, nil
}

func (s *SyncService) Download(ctx context.Context, tenantID uuid.UUID, currentVersion int) (*domain.SyncDownloadResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)
	resp := &domain.SyncDownloadResponse{}

	// Get latest config version
	cv, err := s.configService.GetCurrentConfig(ctx, tenantID)
	if err != nil {
		if appErr, ok := apperror.IsAppError(err); ok && appErr.Code == 404 {
			resp.ConfigVersion = 0
			resp.Services = []domain.ServiceWithPrice{}
			resp.Categories = []domain.ServiceCategory{}
			resp.Members = []domain.Member{}
			return resp, nil
		}
		return nil, err
	}

	resp.ConfigVersion = cv.Version

	if cv.Version > currentVersion {
		resp.Config = cv
	}

	// Always return fresh data
	services, err := s.templateSvc.GetServicesWithPrices(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp.Services = services

	categories, _, err := s.templateSvc.ListCategories(ctx, pagination.Params{Page: 1, PerPage: pagination.MaxPerPage})
	if err != nil {
		return nil, err
	}
	resp.Categories = categories

	// Get members
	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, name, phone, email, tier, discount_percent, total_points, created_at
		 FROM members WHERE tenant_id = $1 OR tenant_id IS NULL`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to get members", err)
	}
	defer rows.Close()

	var members []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan member", err)
		}
		members = append(members, m)
	}
	if members == nil {
		members = []domain.Member{}
	}
	resp.Members = members

	return resp, nil
}

func (s *SyncService) GetSyncHealth(ctx context.Context, tenantID uuid.UUID) (*domain.SyncHealthResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	resp := &domain.SyncHealthResponse{
		TenantID: tenantID,
	}

	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM sync_sessions WHERE tenant_id = $1", tenantID).
		Scan(&resp.TotalSessions)
	if err != nil {
		return nil, apperror.Internal("failed to count sync sessions", err)
	}

	var lastSync *time.Time
	var lastStatus *string
	err = q.QueryRow(ctx,
		`SELECT completed_at, status FROM sync_sessions WHERE tenant_id = $1 ORDER BY started_at DESC LIMIT 1`, tenantID).
		Scan(&lastSync, &lastStatus)
	if err != nil {
		// No sessions yet - that's ok
		resp.LastSyncStatus = "none"
	} else {
		resp.LastSync = lastSync
		if lastStatus != nil {
			resp.LastSyncStatus = *lastStatus
		}
	}

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, outlet_id, direction, status, transaction_count, error_message, started_at, completed_at
		 FROM sync_sessions WHERE tenant_id = $1 ORDER BY started_at DESC LIMIT 10`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list sync sessions", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ss domain.SyncSession
		if err := rows.Scan(&ss.ID, &ss.TenantID, &ss.OutletID, &ss.Direction, &ss.Status,
			&ss.TransactionCount, &ss.ErrorMessage, &ss.StartedAt, &ss.CompletedAt); err != nil {
			return nil, apperror.Internal("failed to scan sync session", err)
		}
		resp.RecentSessions = append(resp.RecentSessions, ss)
	}
	if resp.RecentSessions == nil {
		resp.RecentSessions = []domain.SyncSession{}
	}

	return resp, nil
}
