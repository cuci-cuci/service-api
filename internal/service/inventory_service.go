package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type InventoryService struct {
	db *pgxpool.Pool
}

func NewInventoryService(db *pgxpool.Pool) *InventoryService {
	return &InventoryService{db: db}
}

// --- Categories ---

func (s *InventoryService) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]domain.SupplyCategory, error) {
	q := middleware.GetQuerier(ctx, s.db)
	rows, err := q.Query(ctx, `
		SELECT id, tenant_id, name, icon, is_active, created_at, updated_at
		FROM supply_categories WHERE tenant_id = $1 ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list supply categories", err)
	}
	defer rows.Close()

	var result []domain.SupplyCategory
	for rows.Next() {
		var c domain.SupplyCategory
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Icon, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, apperror.Internal("failed to scan supply category", err)
		}
		result = append(result, c)
	}
	if result == nil {
		result = []domain.SupplyCategory{}
	}
	return result, nil
}

func (s *InventoryService) CreateCategory(ctx context.Context, tenantID uuid.UUID, req domain.CreateSupplyCategoryRequest) (*domain.SupplyCategory, error) {
	q := middleware.GetQuerier(ctx, s.db)
	icon := req.Icon
	if icon == "" {
		icon = "Package"
	}
	var c domain.SupplyCategory
	err := q.QueryRow(ctx, `
		INSERT INTO supply_categories (tenant_id, name, icon) VALUES ($1, $2, $3)
		RETURNING id, tenant_id, name, icon, is_active, created_at, updated_at
	`, tenantID, req.Name, icon).Scan(&c.ID, &c.TenantID, &c.Name, &c.Icon, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create supply category", err)
	}
	return &c, nil
}

// --- Supplies ---

func (s *InventoryService) ListSupplies(ctx context.Context, tenantID uuid.UUID, categoryID *uuid.UUID, lowStockOnly bool) ([]domain.Supply, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT s.id, s.tenant_id, s.category_id, COALESCE(sc.name, ''), s.outlet_id, COALESCE(o.name, ''),
			s.name, s.unit, s.current_stock, s.min_stock, s.cost_per_unit, s.is_active, s.created_at, s.updated_at
		FROM supplies s
		LEFT JOIN supply_categories sc ON s.category_id = sc.id
		LEFT JOIN outlets o ON s.outlet_id = o.id
		WHERE s.tenant_id = $1 AND s.is_active = true
	`
	args := []any{tenantID}
	argIdx := 2

	if categoryID != nil {
		query += fmt.Sprintf(" AND s.category_id = $%d", argIdx)
		args = append(args, *categoryID)
		argIdx++
	}
	if lowStockOnly {
		query += " AND s.current_stock <= s.min_stock AND s.min_stock > 0"
	}
	query += " ORDER BY s.name"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to list supplies", err)
	}
	defer rows.Close()

	var result []domain.Supply
	for rows.Next() {
		var sup domain.Supply
		if err := rows.Scan(
			&sup.ID, &sup.TenantID, &sup.CategoryID, &sup.CategoryName,
			&sup.OutletID, &sup.OutletName,
			&sup.Name, &sup.Unit, &sup.CurrentStock, &sup.MinStock,
			&sup.CostPerUnit, &sup.IsActive, &sup.CreatedAt, &sup.UpdatedAt,
		); err != nil {
			return nil, apperror.Internal("failed to scan supply", err)
		}
		result = append(result, sup)
	}
	if result == nil {
		result = []domain.Supply{}
	}
	return result, nil
}

func (s *InventoryService) CreateSupply(ctx context.Context, tenantID uuid.UUID, req domain.CreateSupplyRequest) (*domain.Supply, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var categoryID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		id, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return nil, apperror.Validation("invalid category_id")
		}
		categoryID = &id
	}
	var outletID *uuid.UUID
	if req.OutletID != nil && *req.OutletID != "" {
		id, err := uuid.Parse(*req.OutletID)
		if err != nil {
			return nil, apperror.Validation("invalid outlet_id")
		}
		outletID = &id
	}

	var sup domain.Supply
	err := q.QueryRow(ctx, `
		INSERT INTO supplies (tenant_id, category_id, outlet_id, name, unit, current_stock, min_stock, cost_per_unit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, category_id, outlet_id, name, unit, current_stock, min_stock, cost_per_unit, is_active, created_at, updated_at
	`, tenantID, categoryID, outletID, req.Name, req.Unit, req.CurrentStock, req.MinStock, req.CostPerUnit,
	).Scan(&sup.ID, &sup.TenantID, &sup.CategoryID, &sup.OutletID, &sup.Name, &sup.Unit,
		&sup.CurrentStock, &sup.MinStock, &sup.CostPerUnit, &sup.IsActive, &sup.CreatedAt, &sup.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create supply", err)
	}
	return &sup, nil
}

func (s *InventoryService) UpdateSupply(ctx context.Context, tenantID, supplyID uuid.UUID, req domain.UpdateSupplyRequest) error {
	q := middleware.GetQuerier(ctx, s.db)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	tag, err := q.Exec(ctx, `
		UPDATE supplies SET name=$1, unit=$2, min_stock=$3, cost_per_unit=$4, is_active=$5, updated_at=NOW()
		WHERE id=$6 AND tenant_id=$7
	`, req.Name, req.Unit, req.MinStock, req.CostPerUnit, isActive, supplyID, tenantID)
	if err != nil {
		return apperror.Internal("failed to update supply", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("supply not found")
	}
	return nil
}

// --- Stock Movements ---

func (s *InventoryService) RecordMovement(ctx context.Context, tenantID, userID uuid.UUID, req domain.StockMovementRequest) error {
	q := middleware.GetQuerier(ctx, s.db)

	supplyID, err := uuid.Parse(req.SupplyID)
	if err != nil {
		return apperror.Validation("invalid supply_id")
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	// Insert movement
	_, err = q.Exec(ctx, `
		INSERT INTO stock_movements (tenant_id, supply_id, movement_type, quantity, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, supplyID, req.MovementType, req.Quantity, notes, userID)
	if err != nil {
		return apperror.Internal("failed to record stock movement", err)
	}

	// Update current stock
	var delta float64
	switch req.MovementType {
	case "in":
		delta = req.Quantity
	case "out":
		delta = -req.Quantity
	case "adjustment":
		// Adjustment sets absolute value
		_, err = q.Exec(ctx, `
			UPDATE supplies SET current_stock = $1, updated_at = NOW()
			WHERE id = $2 AND tenant_id = $3
		`, req.Quantity, supplyID, tenantID)
		if err != nil {
			return apperror.Internal("failed to adjust stock", err)
		}
		return nil
	}

	tag, err := q.Exec(ctx, `
		UPDATE supplies SET current_stock = current_stock + $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, delta, supplyID, tenantID)
	if err != nil {
		return apperror.Internal("failed to update stock", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("supply not found")
	}
	return nil
}

func (s *InventoryService) ListMovements(ctx context.Context, tenantID uuid.UUID, supplyID *uuid.UUID, limit int) ([]domain.StockMovement, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT sm.id, sm.tenant_id, sm.supply_id, s.name, sm.movement_type, sm.quantity,
			sm.notes, sm.created_by, u.name, sm.created_at
		FROM stock_movements sm
		JOIN supplies s ON sm.supply_id = s.id
		JOIN users u ON sm.created_by = u.id
		WHERE sm.tenant_id = $1
	`
	args := []any{tenantID}
	argIdx := 2

	if supplyID != nil {
		query += fmt.Sprintf(" AND sm.supply_id = $%d", argIdx)
		args = append(args, *supplyID)
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY sm.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to list movements", err)
	}
	defer rows.Close()

	var result []domain.StockMovement
	for rows.Next() {
		var m domain.StockMovement
		if err := rows.Scan(&m.ID, &m.TenantID, &m.SupplyID, &m.SupplyName, &m.MovementType,
			&m.Quantity, &m.Notes, &m.CreatedBy, &m.CreatedByName, &m.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan movement", err)
		}
		result = append(result, m)
	}
	if result == nil {
		result = []domain.StockMovement{}
	}
	return result, nil
}

func (s *InventoryService) GetLowStockAlerts(ctx context.Context, tenantID uuid.UUID) ([]domain.LowStockAlert, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT s.id, s.name, s.unit, s.current_stock, s.min_stock, COALESCE(o.name, '')
		FROM supplies s
		LEFT JOIN outlets o ON s.outlet_id = o.id
		WHERE s.tenant_id = $1 AND s.is_active = true
			AND s.min_stock > 0 AND s.current_stock <= s.min_stock
		ORDER BY (s.current_stock / NULLIF(s.min_stock, 0)) ASC
	`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to get low stock alerts", err)
	}
	defer rows.Close()

	var result []domain.LowStockAlert
	for rows.Next() {
		var a domain.LowStockAlert
		if err := rows.Scan(&a.SupplyID, &a.Name, &a.Unit, &a.CurrentStock, &a.MinStock, &a.OutletName); err != nil {
			return nil, apperror.Internal("failed to scan low stock alert", err)
		}
		result = append(result, a)
	}
	if result == nil {
		result = []domain.LowStockAlert{}
	}
	return result, nil
}
