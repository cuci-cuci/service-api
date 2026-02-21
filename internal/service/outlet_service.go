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

type OutletService struct {
	db *pgxpool.Pool
}

func NewOutletService(db *pgxpool.Pool) *OutletService {
	return &OutletService{db: db}
}

func (s *OutletService) ListByTenant(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.Outlet, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM outlets WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		slog.Error("failed to count outlets", "error", err)
		return nil, 0, apperror.Internal("failed to count outlets", err)
	}

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, name, address, phone, is_active, created_at, updated_at
		 FROM outlets WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, params.PerPage, params.Offset())
	if err != nil {
		slog.Error("failed to list outlets", "error", err)
		return nil, 0, apperror.Internal("failed to list outlets", err)
	}
	defer rows.Close()

	var outlets []domain.Outlet
	for rows.Next() {
		var o domain.Outlet
		if err := rows.Scan(&o.ID, &o.TenantID, &o.Name, &o.Address, &o.Phone, &o.IsActive, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan outlet", err)
		}
		outlets = append(outlets, o)
	}

	if outlets == nil {
		outlets = []domain.Outlet{}
	}

	return outlets, total, nil
}

func (s *OutletService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Outlet, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var o domain.Outlet
	err := q.QueryRow(ctx,
		`SELECT id, tenant_id, name, address, phone, is_active, created_at, updated_at
		 FROM outlets WHERE id = $1`, id).
		Scan(&o.ID, &o.TenantID, &o.Name, &o.Address, &o.Phone, &o.IsActive, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("outlet not found")
		}
		return nil, apperror.Internal("failed to get outlet", err)
	}
	return &o, nil
}

func (s *OutletService) Create(ctx context.Context, tenantID uuid.UUID, req domain.CreateOutletRequest) (*domain.Outlet, error) {
	q := middleware.GetQuerier(ctx, s.db)

	o := domain.Outlet{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      req.Name,
		Address:   req.Address,
		Phone:     req.Phone,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := q.Exec(ctx,
		`INSERT INTO outlets (id, tenant_id, name, address, phone, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		o.ID, o.TenantID, o.Name, o.Address, o.Phone, o.IsActive, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create outlet", err)
	}

	return &o, nil
}

func (s *OutletService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateOutletRequest) (*domain.Outlet, error) {
	q := middleware.GetQuerier(ctx, s.db)

	o, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		o.Name = req.Name
	}
	if req.Address != "" {
		o.Address = req.Address
	}
	if req.Phone != "" {
		o.Phone = req.Phone
	}
	if req.IsActive != nil {
		o.IsActive = *req.IsActive
	}
	o.UpdatedAt = time.Now()

	_, err = q.Exec(ctx,
		`UPDATE outlets SET name = $1, address = $2, phone = $3, is_active = $4, updated_at = $5 WHERE id = $6`,
		o.Name, o.Address, o.Phone, o.IsActive, o.UpdatedAt, o.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update outlet", err)
	}

	return o, nil
}
