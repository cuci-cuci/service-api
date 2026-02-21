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

type ServiceTemplateService struct {
	db *pgxpool.Pool
}

func NewServiceTemplateService(db *pgxpool.Pool) *ServiceTemplateService {
	return &ServiceTemplateService{db: db}
}

// --- Service Categories ---

func (s *ServiceTemplateService) ListCategories(ctx context.Context, params pagination.Params) ([]domain.ServiceCategory, int, error) {
	var total int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM service_categories").Scan(&total)
	if err != nil {
		slog.Error("failed to count categories", "error", err)
		return nil, 0, apperror.Internal("failed to count categories", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, name, icon, sort_order, is_active, created_at
		 FROM service_categories ORDER BY sort_order ASC, name ASC LIMIT $1 OFFSET $2`,
		params.PerPage, params.Offset())
	if err != nil {
		return nil, 0, apperror.Internal("failed to list categories", err)
	}
	defer rows.Close()

	var categories []domain.ServiceCategory
	for rows.Next() {
		var c domain.ServiceCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan category", err)
		}
		categories = append(categories, c)
	}

	if categories == nil {
		categories = []domain.ServiceCategory{}
	}

	return categories, total, nil
}

func (s *ServiceTemplateService) CreateCategory(ctx context.Context, req domain.CreateServiceCategoryRequest) (*domain.ServiceCategory, error) {
	c := domain.ServiceCategory{
		ID:        uuid.New(),
		Name:      req.Name,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO service_categories (id, name, icon, sort_order, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.Name, c.Icon, c.SortOrder, c.IsActive, c.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create category", err)
	}

	return &c, nil
}

func (s *ServiceTemplateService) UpdateCategory(ctx context.Context, id uuid.UUID, req domain.UpdateServiceCategoryRequest) (*domain.ServiceCategory, error) {
	var c domain.ServiceCategory
	err := s.db.QueryRow(ctx,
		`SELECT id, name, icon, sort_order, is_active, created_at FROM service_categories WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.IsActive, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("category not found")
		}
		return nil, apperror.Internal("failed to get category", err)
	}

	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Icon != "" {
		c.Icon = req.Icon
	}
	if req.SortOrder != nil {
		c.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}

	_, err = s.db.Exec(ctx,
		`UPDATE service_categories SET name = $1, icon = $2, sort_order = $3, is_active = $4 WHERE id = $5`,
		c.Name, c.Icon, c.SortOrder, c.IsActive, c.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update category", err)
	}

	return &c, nil
}

// --- Service Templates ---

func (s *ServiceTemplateService) ListTemplates(ctx context.Context, params pagination.Params) ([]domain.ServiceTemplate, int, error) {
	var total int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM service_templates").Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count templates", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, category_id, name, pricing_unit, base_price, estimated_duration_hours, is_active, sort_order, created_at
		 FROM service_templates ORDER BY sort_order ASC, name ASC LIMIT $1 OFFSET $2`,
		params.PerPage, params.Offset())
	if err != nil {
		return nil, 0, apperror.Internal("failed to list templates", err)
	}
	defer rows.Close()

	var templates []domain.ServiceTemplate
	for rows.Next() {
		var t domain.ServiceTemplate
		if err := rows.Scan(&t.ID, &t.CategoryID, &t.Name, &t.PricingUnit, &t.BasePrice,
			&t.EstimatedDurationHours, &t.IsActive, &t.SortOrder, &t.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan template", err)
		}
		templates = append(templates, t)
	}

	if templates == nil {
		templates = []domain.ServiceTemplate{}
	}

	return templates, total, nil
}

func (s *ServiceTemplateService) CreateTemplate(ctx context.Context, req domain.CreateServiceTemplateRequest) (*domain.ServiceTemplate, error) {
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, apperror.Validation("invalid category_id")
	}

	t := domain.ServiceTemplate{
		ID:                     uuid.New(),
		CategoryID:             categoryID,
		Name:                   req.Name,
		PricingUnit:            req.PricingUnit,
		BasePrice:              req.BasePrice,
		EstimatedDurationHours: req.EstimatedDurationHours,
		IsActive:               true,
		SortOrder:              req.SortOrder,
		CreatedAt:              time.Now(),
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, is_active, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.CategoryID, t.Name, t.PricingUnit, t.BasePrice, t.EstimatedDurationHours, t.IsActive, t.SortOrder, t.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create template", err)
	}

	return &t, nil
}

func (s *ServiceTemplateService) UpdateTemplate(ctx context.Context, id uuid.UUID, req domain.UpdateServiceTemplateRequest) (*domain.ServiceTemplate, error) {
	var t domain.ServiceTemplate
	err := s.db.QueryRow(ctx,
		`SELECT id, category_id, name, pricing_unit, base_price, estimated_duration_hours, is_active, sort_order, created_at
		 FROM service_templates WHERE id = $1`, id).
		Scan(&t.ID, &t.CategoryID, &t.Name, &t.PricingUnit, &t.BasePrice,
			&t.EstimatedDurationHours, &t.IsActive, &t.SortOrder, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("template not found")
		}
		return nil, apperror.Internal("failed to get template", err)
	}

	if req.Name != "" {
		t.Name = req.Name
	}
	if req.PricingUnit != "" {
		t.PricingUnit = req.PricingUnit
	}
	if req.BasePrice != nil {
		t.BasePrice = *req.BasePrice
	}
	if req.EstimatedDurationHours != nil {
		t.EstimatedDurationHours = *req.EstimatedDurationHours
	}
	if req.SortOrder != nil {
		t.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}

	_, err = s.db.Exec(ctx,
		`UPDATE service_templates SET name=$1, pricing_unit=$2, base_price=$3, estimated_duration_hours=$4, is_active=$5, sort_order=$6 WHERE id=$7`,
		t.Name, t.PricingUnit, t.BasePrice, t.EstimatedDurationHours, t.IsActive, t.SortOrder, t.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update template", err)
	}

	return &t, nil
}

// --- Tenant Service Prices ---

func (s *ServiceTemplateService) SetTenantPrice(ctx context.Context, tenantID uuid.UUID, req domain.SetTenantServicePriceRequest) (*domain.TenantServicePrice, error) {
	templateID, err := uuid.Parse(req.ServiceTemplateID)
	if err != nil {
		return nil, apperror.Validation("invalid service_template_id")
	}

	tsp := domain.TenantServicePrice{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ServiceTemplateID: templateID,
		Price:             req.Price,
		IsActive:          true,
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO tenant_service_prices (id, tenant_id, service_template_id, price, is_active)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id, service_template_id) DO UPDATE SET price = $4, is_active = $5`,
		tsp.ID, tsp.TenantID, tsp.ServiceTemplateID, tsp.Price, tsp.IsActive)
	if err != nil {
		return nil, apperror.Internal("failed to set tenant price", err)
	}

	return &tsp, nil
}

func (s *ServiceTemplateService) GetServicesWithPrices(ctx context.Context, tenantID uuid.UUID) ([]domain.ServiceWithPrice, error) {
	rows, err := s.db.Query(ctx,
		`SELECT st.id, st.category_id, st.name, st.pricing_unit, st.base_price,
		        st.estimated_duration_hours, st.is_active, st.sort_order, st.created_at,
		        tsp.price
		 FROM service_templates st
		 LEFT JOIN tenant_service_prices tsp ON st.id = tsp.service_template_id AND tsp.tenant_id = $1
		 WHERE st.is_active = true
		 ORDER BY st.sort_order ASC, st.name ASC`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list services with prices", err)
	}
	defer rows.Close()

	var services []domain.ServiceWithPrice
	for rows.Next() {
		var s domain.ServiceWithPrice
		var tenantPrice *int64
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.PricingUnit, &s.BasePrice,
			&s.EstimatedDurationHours, &s.IsActive, &s.SortOrder, &s.CreatedAt, &tenantPrice); err != nil {
			return nil, apperror.Internal("failed to scan service with price", err)
		}
		s.TenantPrice = tenantPrice
		services = append(services, s)
	}

	if services == nil {
		services = []domain.ServiceWithPrice{}
	}

	return services, nil
}
