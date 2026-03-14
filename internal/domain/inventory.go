package domain

import (
	"time"

	"github.com/google/uuid"
)

type SupplyCategory struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Supply struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty"`
	CategoryName string    `json:"category_name,omitempty"`
	OutletID     *uuid.UUID `json:"outlet_id,omitempty"`
	OutletName   string    `json:"outlet_name,omitempty"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	CurrentStock float64   `json:"current_stock"`
	MinStock     float64   `json:"min_stock"`
	CostPerUnit  int64     `json:"cost_per_unit"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StockMovement struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	SupplyID     uuid.UUID `json:"supply_id"`
	SupplyName   string    `json:"supply_name,omitempty"`
	MovementType string    `json:"movement_type"`
	Quantity     float64   `json:"quantity"`
	Notes        *string   `json:"notes,omitempty"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedByName string   `json:"created_by_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateSupplyCategoryRequest struct {
	Name string `json:"name" validate:"required,max=100"`
	Icon string `json:"icon"`
}

type CreateSupplyRequest struct {
	CategoryID  *string `json:"category_id"`
	OutletID    *string `json:"outlet_id"`
	Name        string  `json:"name" validate:"required,max=200"`
	Unit        string  `json:"unit" validate:"required,max=50"`
	CurrentStock float64 `json:"current_stock"`
	MinStock    float64 `json:"min_stock"`
	CostPerUnit int64   `json:"cost_per_unit"`
}

type UpdateSupplyRequest struct {
	Name        string  `json:"name" validate:"required,max=200"`
	Unit        string  `json:"unit" validate:"required,max=50"`
	MinStock    float64 `json:"min_stock"`
	CostPerUnit int64   `json:"cost_per_unit"`
	IsActive    *bool   `json:"is_active"`
}

type StockMovementRequest struct {
	SupplyID     string  `json:"supply_id" validate:"required,uuid"`
	MovementType string  `json:"movement_type" validate:"required,oneof=in out adjustment"`
	Quantity     float64 `json:"quantity" validate:"required,gt=0"`
	Notes        string  `json:"notes"`
}

type LowStockAlert struct {
	SupplyID     uuid.UUID `json:"supply_id"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	CurrentStock float64   `json:"current_stock"`
	MinStock     float64   `json:"min_stock"`
	OutletName   string    `json:"outlet_name,omitempty"`
}

// --- Service-Supply Mappings ---

type ServiceSupplyMapping struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	ServiceTemplateID uuid.UUID `json:"service_template_id"`
	SupplyID          uuid.UUID `json:"supply_id"`
	QuantityPerUnit   float64   `json:"quantity_per_unit"`
	Unit              string    `json:"unit"`
	ServiceName       string    `json:"service_name"`
	SupplyName        string    `json:"supply_name"`
	CreatedAt         time.Time `json:"created_at"`
}

type CreateServiceSupplyMappingRequest struct {
	ServiceTemplateID uuid.UUID `json:"service_template_id" validate:"required"`
	SupplyID          uuid.UUID `json:"supply_id" validate:"required"`
	QuantityPerUnit   float64   `json:"quantity_per_unit" validate:"required,gt=0"`
	Unit              string    `json:"unit" validate:"required"`
}

type UpdateServiceSupplyMappingRequest struct {
	QuantityPerUnit float64 `json:"quantity_per_unit" validate:"required,gt=0"`
	Unit            string  `json:"unit" validate:"required"`
}

type ServiceCostResponse struct {
	ServiceTemplateID uuid.UUID `json:"service_template_id"`
	ServiceName       string    `json:"service_name"`
	TotalCostPerUnit  int64     `json:"total_cost_per_unit"`
	MappingCount      int       `json:"mapping_count"`
}
