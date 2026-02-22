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
	"github.com/bangun-ekosistem/service-api/internal/middleware"
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
	q := middleware.GetQuerier(ctx, s.db)

	var cv domain.ConfigVersion
	err := q.QueryRow(ctx,
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
	q := middleware.GetQuerier(ctx, s.db)

	// Build the config snapshot
	services, err := s.templateService.GetServicesWithPrices(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	categories, _, err := s.templateService.ListCategories(ctx, pagination.Params{Page: 1, PerPage: pagination.MaxPerPage})
	if err != nil {
		return nil, err
	}

	// Get members for this tenant
	var members []domain.Member
	rows, err := q.Query(ctx,
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
	flagRows, err := q.Query(ctx,
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

	// FIX 8: Atomic version increment using a single INSERT...SELECT to
	// prevent race conditions when multiple PushConfig calls happen concurrently.
	cv := domain.ConfigVersion{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Data:      data,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	err = q.QueryRow(ctx,
		`INSERT INTO config_versions (id, tenant_id, version, data, created_by, created_at)
		 SELECT $1, $2, COALESCE(MAX(version), 0) + 1, $3, $4, $5
		 FROM config_versions WHERE tenant_id = $2
		 RETURNING version`,
		cv.ID, cv.TenantID, cv.Data, cv.CreatedBy, cv.CreatedAt).Scan(&cv.Version)
	if err != nil {
		return nil, apperror.Internal("failed to insert config version", err)
	}

	slog.Info("config pushed", "tenant_id", tenantID, "version", cv.Version)
	return &cv, nil
}

func (s *ConfigService) BroadcastConfig(ctx context.Context, req domain.PushConfigRequest, createdBy uuid.UUID) ([]domain.ConfigVersion, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var tenantIDs []uuid.UUID

	if req.Broadcast {
		rows, err := q.Query(ctx, "SELECT id FROM tenants WHERE is_active = true")
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

type TenantConfigListItem struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	TenantName string          `json:"tenant_name"`
	Version    int             `json:"version"`
	Data       json.RawMessage `json:"data"`
	CreatedBy  uuid.UUID       `json:"created_by"`
	CreatedAt  time.Time       `json:"created_at"`
}

func (s *ConfigService) ListAllConfigs(ctx context.Context) ([]TenantConfigListItem, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx,
		`SELECT cv.id, cv.tenant_id, t.name, cv.version, cv.data, cv.created_by, cv.created_at
		 FROM config_versions cv
		 JOIN tenants t ON cv.tenant_id = t.id
		 WHERE cv.version = (
		     SELECT MAX(cv2.version) FROM config_versions cv2 WHERE cv2.tenant_id = cv.tenant_id
		 )
		 ORDER BY cv.created_at DESC`)
	if err != nil {
		slog.Error("failed to list all configs", "error", err)
		return nil, apperror.Internal("failed to list configs", err)
	}
	defer rows.Close()

	var configs []TenantConfigListItem
	for rows.Next() {
		var c TenantConfigListItem
		if err := rows.Scan(&c.ID, &c.TenantID, &c.TenantName, &c.Version, &c.Data, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan config", err)
		}
		configs = append(configs, c)
	}

	if configs == nil {
		configs = []TenantConfigListItem{}
	}

	return configs, nil
}

func (s *ConfigService) GetConfigHistory(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.ConfigVersion, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM config_versions WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count configs", err)
	}

	rows, err := q.Query(ctx,
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
