package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type TenantService struct {
	db *pgxpool.Pool
}

func NewTenantService(db *pgxpool.Pool) *TenantService {
	return &TenantService{db: db}
}

func (s *TenantService) List(ctx context.Context, params pagination.Params) ([]domain.Tenant, int, error) {
	var total int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&total)
	if err != nil {
		slog.Error("failed to count tenants", "error", err)
		return nil, 0, apperror.Internal("failed to count tenants", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, name, slug, is_active, created_at, updated_at
		 FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		params.PerPage, params.Offset())
	if err != nil {
		slog.Error("failed to list tenants", "error", err)
		return nil, 0, apperror.Internal("failed to list tenants", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan tenant", err)
		}
		tenants = append(tenants, t)
	}

	if tenants == nil {
		tenants = []domain.Tenant{}
	}

	return tenants, total, nil
}

func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	var t domain.Tenant
	err := s.db.QueryRow(ctx,
		`SELECT id, name, slug, is_active, created_at, updated_at
		 FROM tenants WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Slug, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("tenant not found")
		}
		return nil, apperror.Internal("failed to get tenant", err)
	}
	return &t, nil
}

func (s *TenantService) Create(ctx context.Context, req domain.CreateTenantRequest) (*domain.Tenant, error) {
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)", req.Slug).Scan(&exists)
	if err != nil {
		return nil, apperror.Internal("failed to check slug", err)
	}
	if exists {
		return nil, apperror.Conflict("tenant with this slug already exists")
	}

	t := domain.Tenant{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.Name, t.Slug, t.IsActive, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create tenant", err)
	}

	return &t, nil
}

func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateTenantRequest) (*domain.Tenant, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		t.Name = req.Name
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}
	t.UpdatedAt = time.Now()

	_, err = s.db.Exec(ctx,
		`UPDATE tenants SET name = $1, is_active = $2, updated_at = $3 WHERE id = $4`,
		t.Name, t.IsActive, t.UpdatedAt, t.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update tenant", err)
	}

	return t, nil
}
