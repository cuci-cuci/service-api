package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type AdminInventoryService struct {
	db *pgxpool.Pool
}

func NewAdminInventoryService(db *pgxpool.Pool) *AdminInventoryService {
	return &AdminInventoryService{db: db}
}

type AdminSupplyItem struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	TenantName   string     `json:"tenant_name"`
	CategoryName string     `json:"category_name"`
	OutletName   string     `json:"outlet_name"`
	Name         string     `json:"name"`
	Unit         string     `json:"unit"`
	CurrentStock float64    `json:"current_stock"`
	MinStock     float64    `json:"min_stock"`
	CostPerUnit  int64      `json:"cost_per_unit"`
	IsActive     bool       `json:"is_active"`
}

type AdminLowStockAlert struct {
	SupplyID     uuid.UUID `json:"supply_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	TenantName   string    `json:"tenant_name"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	CurrentStock float64   `json:"current_stock"`
	MinStock     float64   `json:"min_stock"`
	OutletName   string    `json:"outlet_name"`
}

type AdminInventorySummary struct {
	TotalSupplies  int `json:"total_supplies"`
	LowStockCount  int `json:"low_stock_count"`
	TotalTenants   int `json:"total_tenants"`
}

func (s *AdminInventoryService) ListSupplies(ctx context.Context, tenantID uuid.UUID) ([]AdminSupplyItem, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT s.id, s.tenant_id, t.name, COALESCE(sc.name, ''), COALESCE(o.name, ''),
			s.name, s.unit, s.current_stock, s.min_stock, s.cost_per_unit, s.is_active
		FROM supplies s
		JOIN tenants t ON s.tenant_id = t.id
		LEFT JOIN supply_categories sc ON s.category_id = sc.id
		LEFT JOIN outlets o ON s.outlet_id = o.id
		WHERE s.tenant_id = $1
		ORDER BY s.name
	`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list supplies", err)
	}
	defer rows.Close()

	var result []AdminSupplyItem
	for rows.Next() {
		var item AdminSupplyItem
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.CategoryName,
			&item.OutletName, &item.Name, &item.Unit, &item.CurrentStock,
			&item.MinStock, &item.CostPerUnit, &item.IsActive); err != nil {
			return nil, apperror.Internal("failed to scan supply", err)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []AdminSupplyItem{}
	}
	return result, nil
}

func (s *AdminInventoryService) LowStockAlertsGlobal(ctx context.Context) ([]AdminLowStockAlert, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT s.id, s.tenant_id, t.name, s.name, s.unit, s.current_stock, s.min_stock, COALESCE(o.name, '')
		FROM supplies s
		JOIN tenants t ON s.tenant_id = t.id
		LEFT JOIN outlets o ON s.outlet_id = o.id
		WHERE s.is_active = true AND s.min_stock > 0 AND s.current_stock <= s.min_stock
		ORDER BY (s.current_stock / NULLIF(s.min_stock, 0)) ASC
		LIMIT 100
	`)
	if err != nil {
		return nil, apperror.Internal("failed to get low stock alerts", err)
	}
	defer rows.Close()

	var result []AdminLowStockAlert
	for rows.Next() {
		var a AdminLowStockAlert
		if err := rows.Scan(&a.SupplyID, &a.TenantID, &a.TenantName, &a.Name, &a.Unit,
			&a.CurrentStock, &a.MinStock, &a.OutletName); err != nil {
			return nil, apperror.Internal("failed to scan alert", err)
		}
		result = append(result, a)
	}
	if result == nil {
		result = []AdminLowStockAlert{}
	}
	return result, nil
}

func (s *AdminInventoryService) Summary(ctx context.Context) (*AdminInventorySummary, error) {
	q := middleware.GetQuerier(ctx, s.db)

	summary := &AdminInventorySummary{}

	_ = q.QueryRow(ctx, `SELECT COUNT(*) FROM supplies WHERE is_active = true`).Scan(&summary.TotalSupplies)
	_ = q.QueryRow(ctx, `
		SELECT COUNT(*) FROM supplies
		WHERE is_active = true AND min_stock > 0 AND current_stock <= min_stock
	`).Scan(&summary.LowStockCount)
	_ = q.QueryRow(ctx, `SELECT COUNT(DISTINCT tenant_id) FROM supplies`).Scan(&summary.TotalTenants)

	return summary, nil
}
