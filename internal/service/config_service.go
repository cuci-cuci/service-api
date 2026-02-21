package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type ConfigService struct {
	db              *pgxpool.Pool
	templateService *ServiceTemplateService
}

func NewConfigService(db *pgxpool.Pool, templateService *ServiceTemplateService) *ConfigService {
	return &ConfigService{db: db, templateService: templateService}
}

func (s *ConfigService) GetCurrentConfig(ctx context.Context, tenantID uuid.UUID) (*domain.ConfigVersion, error) {
	var cv domain.ConfigVersion
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, version, data, created_by, created_at
		 FROM config_versions WHERE tenant_id = $1 ORDER BY version DESC LIMIT 1`, tenantID).
		Scan(&cv.ID, &cv.TenantID, &cv.Version, &cv.Data, &cv.CreatedBy, &cv.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("no config found for this tenant")
		}
		return nil, apperror.Internal("failed to get config", err)
	}
	return &cv, nil
}

func (s *ConfigService) PushConfig(ctx context.Context, tenantID uuid.UUID, createdBy uuid.UUID) (*domain.ConfigVersion, error) {
	// Build the config snapshot
	services, err := s.templateService.GetServicesWithPrices(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	categories, _, err := s.templateService.ListCategories(ctx, pagination.Params{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, err
	}

	// Get members for this tenant
	var members []domain.Member
	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, name, phone, email, tier, discount_percent, total_points, created_at
		 FROM members WHERE tenant_id = $1 OR tenant_id IS NULL`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to get members", err)
	}
	defer rows.Close()

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

	// Get tenant feature flags
	flagRows, err := s.db.Query(ctx,
		`SELECT ff.key, COALESCE(tff.enabled, ff.default_enabled) as enabled
		 FROM feature_flags ff
		 LEFT JOIN tenant_feature_flags tff ON ff.id = tff.feature_flag_id AND tff.tenant_id = $1`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to get feature flags", err)
	}
	defer flagRows.Close()

	flags := make(map[string]bool)
	for flagRows.Next() {
		var key string
		var enabled bool
		if err := flagRows.Scan(&key, &enabled); err != nil {
			return nil, apperror.Internal("failed to scan flag", err)
		}
		flags[key] = enabled
	}

	snapshot := map[string]any{
		"services":      services,
		"categories":    categories,
		"members":       members,
		"feature_flags": flags,
		"generated_at":  time.Now(),
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, apperror.Internal("failed to marshal config", err)
	}

	// Get current max version
	var currentVersion int
	err = s.db.QueryRow(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM config_versions WHERE tenant_id = $1", tenantID).
		Scan(&currentVersion)
	if err != nil {
		return nil, apperror.Internal("failed to get current version", err)
	}

	cv := domain.ConfigVersion{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Version:   currentVersion + 1,
		Data:      data,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO config_versions (id, tenant_id, version, data, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		cv.ID, cv.TenantID, cv.Version, cv.Data, cv.CreatedBy, cv.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to insert config version", err)
	}

	slog.Info("config pushed", "tenant_id", tenantID, "version", cv.Version)
	return &cv, nil
}

func (s *ConfigService) BroadcastConfig(ctx context.Context, req domain.PushConfigRequest, createdBy uuid.UUID) ([]domain.ConfigVersion, error) {
	var tenantIDs []uuid.UUID

	if req.Broadcast {
		rows, err := s.db.Query(ctx, "SELECT id FROM tenants WHERE is_active = true")
		if err != nil {
			return nil, apperror.Internal("failed to list active tenants", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return nil, apperror.Internal("failed to scan tenant id", err)
			}
			tenantIDs = append(tenantIDs, id)
		}
	} else {
		tenantIDs = req.TenantIDs
	}

	var results []domain.ConfigVersion
	for _, tid := range tenantIDs {
		cv, err := s.PushConfig(ctx, tid, createdBy)
		if err != nil {
			slog.Error("failed to push config", "tenant_id", tid, "error", err)
			continue
		}
		results = append(results, *cv)
	}

	if results == nil {
		results = []domain.ConfigVersion{}
	}

	return results, nil
}

func (s *ConfigService) GetConfigHistory(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.ConfigVersion, int, error) {
	var total int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM config_versions WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count configs", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, version, data, created_by, created_at
		 FROM config_versions WHERE tenant_id = $1 ORDER BY version DESC LIMIT $2 OFFSET $3`,
		tenantID, params.PerPage, params.Offset())
	if err != nil {
		return nil, 0, apperror.Internal("failed to list configs", err)
	}
	defer rows.Close()

	var configs []domain.ConfigVersion
	for rows.Next() {
		var cv domain.ConfigVersion
		if err := rows.Scan(&cv.ID, &cv.TenantID, &cv.Version, &cv.Data, &cv.CreatedBy, &cv.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan config", err)
		}
		configs = append(configs, cv)
	}

	if configs == nil {
		configs = []domain.ConfigVersion{}
	}

	return configs, total, nil
}
