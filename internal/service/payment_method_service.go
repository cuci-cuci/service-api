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

type PaymentMethodService struct {
	db *pgxpool.Pool
}

func NewPaymentMethodService(db *pgxpool.Pool) *PaymentMethodService {
	return &PaymentMethodService{db: db}
}

func (s *PaymentMethodService) List(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.PaymentMethod, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM payment_methods WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		slog.Error("failed to count payment methods", "error", err)
		return nil, 0, apperror.Internal("failed to count payment methods", err)
	}

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, name, type, is_active, sort_order, created_at
		 FROM payment_methods WHERE tenant_id = $1 ORDER BY sort_order ASC, name ASC LIMIT $2 OFFSET $3`,
		tenantID, params.PerPage, params.Offset())
	if err != nil {
		slog.Error("failed to list payment methods", "error", err)
		return nil, 0, apperror.Internal("failed to list payment methods", err)
	}
	defer rows.Close()

	var methods []domain.PaymentMethod
	for rows.Next() {
		var m domain.PaymentMethod
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Type, &m.IsActive, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan payment method", err)
		}
		methods = append(methods, m)
	}

	if methods == nil {
		methods = []domain.PaymentMethod{}
	}

	return methods, total, nil
}

func (s *PaymentMethodService) Create(ctx context.Context, tenantID uuid.UUID, req domain.CreatePaymentMethodRequest) (*domain.PaymentMethod, error) {
	q := middleware.GetQuerier(ctx, s.db)

	m := domain.PaymentMethod{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      req.Name,
		Type:      req.Type,
		IsActive:  true,
		SortOrder: req.SortOrder,
		CreatedAt: time.Now(),
	}

	_, err := q.Exec(ctx,
		`INSERT INTO payment_methods (id, tenant_id, name, type, is_active, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.TenantID, m.Name, m.Type, m.IsActive, m.SortOrder, m.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create payment method", err)
	}

	return &m, nil
}

func (s *PaymentMethodService) Update(ctx context.Context, id uuid.UUID, req domain.UpdatePaymentMethodRequest) (*domain.PaymentMethod, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var m domain.PaymentMethod
	err := q.QueryRow(ctx,
		`SELECT id, tenant_id, name, type, is_active, sort_order, created_at
		 FROM payment_methods WHERE id = $1`, id).
		Scan(&m.ID, &m.TenantID, &m.Name, &m.Type, &m.IsActive, &m.SortOrder, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("payment method not found")
		}
		return nil, apperror.Internal("failed to get payment method", err)
	}

	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Type != "" {
		m.Type = req.Type
	}
	if req.SortOrder != nil {
		m.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		m.IsActive = *req.IsActive
	}

	_, err = q.Exec(ctx,
		`UPDATE payment_methods SET name = $1, type = $2, sort_order = $3, is_active = $4 WHERE id = $5`,
		m.Name, m.Type, m.SortOrder, m.IsActive, m.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update payment method", err)
	}

	return &m, nil
}

func (s *PaymentMethodService) Delete(ctx context.Context, id uuid.UUID) error {
	q := middleware.GetQuerier(ctx, s.db)

	result, err := q.Exec(ctx, `UPDATE payment_methods SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return apperror.Internal("failed to delete payment method", err)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound("payment method not found")
	}

	return nil
}
