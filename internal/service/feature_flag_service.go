package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type FeatureFlagService struct {
	db *pgxpool.Pool
}

func NewFeatureFlagService(db *pgxpool.Pool) *FeatureFlagService {
	return &FeatureFlagService{db: db}
}

func (s *FeatureFlagService) ListFlags(ctx context.Context) ([]domain.FeatureFlag, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, key, description, default_enabled FROM feature_flags ORDER BY key ASC`)
	if err != nil {
		slog.Error("failed to list feature flags", "error", err)
		return nil, apperror.Internal("failed to list flags", err)
	}
	defer rows.Close()

	var flags []domain.FeatureFlag
	for rows.Next() {
		var f domain.FeatureFlag
		if err := rows.Scan(&f.ID, &f.Key, &f.Description, &f.DefaultEnabled); err != nil {
			return nil, apperror.Internal("failed to scan flag", err)
		}
		flags = append(flags, f)
	}

	if flags == nil {
		flags = []domain.FeatureFlag{}
	}

	return flags, nil
}

func (s *FeatureFlagService) CreateFlag(ctx context.Context, req domain.CreateFeatureFlagRequest) (*domain.FeatureFlag, error) {
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM feature_flags WHERE key = $1)", req.Key).Scan(&exists)
	if err != nil {
		return nil, apperror.Internal("failed to check flag key", err)
	}
	if exists {
		return nil, apperror.Conflict("feature flag with this key already exists")
	}

	f := domain.FeatureFlag{
		ID:             uuid.New(),
		Key:            req.Key,
		Description:    req.Description,
		DefaultEnabled: req.DefaultEnabled,
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO feature_flags (id, key, description, default_enabled) VALUES ($1, $2, $3, $4)`,
		f.ID, f.Key, f.Description, f.DefaultEnabled)
	if err != nil {
		return nil, apperror.Internal("failed to create flag", err)
	}

	return &f, nil
}

func (s *FeatureFlagService) UpdateFlag(ctx context.Context, id uuid.UUID, req domain.UpdateFeatureFlagRequest) (*domain.FeatureFlag, error) {
	var f domain.FeatureFlag
	err := s.db.QueryRow(ctx,
		`SELECT id, key, description, default_enabled FROM feature_flags WHERE id = $1`, id).
		Scan(&f.ID, &f.Key, &f.Description, &f.DefaultEnabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("feature flag not found")
		}
		return nil, apperror.Internal("failed to get flag", err)
	}

	if req.Description != nil {
		f.Description = *req.Description
	}
	if req.DefaultEnabled != nil {
		f.DefaultEnabled = *req.DefaultEnabled
	}

	_, err = s.db.Exec(ctx,
		`UPDATE feature_flags SET description = $1, default_enabled = $2 WHERE id = $3`,
		f.Description, f.DefaultEnabled, f.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update flag", err)
	}

	return &f, nil
}

type TenantFeatureFlagResponse struct {
	domain.FeatureFlag
	Enabled bool `json:"enabled"`
}

func (s *FeatureFlagService) GetTenantFlags(ctx context.Context, tenantID uuid.UUID) ([]TenantFeatureFlagResponse, error) {
	rows, err := s.db.Query(ctx,
		`SELECT ff.id, ff.key, ff.description, ff.default_enabled,
		        COALESCE(tff.enabled, ff.default_enabled) as enabled
		 FROM feature_flags ff
		 LEFT JOIN tenant_feature_flags tff ON ff.id = tff.feature_flag_id AND tff.tenant_id = $1
		 ORDER BY ff.key ASC`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to get tenant flags", err)
	}
	defer rows.Close()

	var flags []TenantFeatureFlagResponse
	for rows.Next() {
		var f TenantFeatureFlagResponse
		if err := rows.Scan(&f.ID, &f.Key, &f.Description, &f.DefaultEnabled, &f.Enabled); err != nil {
			return nil, apperror.Internal("failed to scan tenant flag", err)
		}
		flags = append(flags, f)
	}

	if flags == nil {
		flags = []TenantFeatureFlagResponse{}
	}

	return flags, nil
}

func (s *FeatureFlagService) SetTenantFlag(ctx context.Context, tenantID uuid.UUID, flagID uuid.UUID, enabled bool) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO tenant_feature_flags (id, tenant_id, feature_flag_id, enabled)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (tenant_id, feature_flag_id) DO UPDATE SET enabled = $4`,
		uuid.New(), tenantID, flagID, enabled)
	if err != nil {
		return apperror.Internal("failed to set tenant flag", err)
	}
	return nil
}
