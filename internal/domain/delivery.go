package domain

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryZone struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	OutletID         uuid.UUID `json:"outlet_id"`
	Name             string    `json:"name"`
	District         *string   `json:"district,omitempty"`
	Fee              int64     `json:"fee"`
	EstimatedMinutes int       `json:"estimated_minutes"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateDeliveryZoneRequest struct {
	OutletID         string  `json:"outlet_id" validate:"required,uuid"`
	Name             string  `json:"name" validate:"required,max=100"`
	District         *string `json:"district" validate:"omitempty,max=200"`
	Fee              int64   `json:"fee" validate:"gte=0"`
	EstimatedMinutes int     `json:"estimated_minutes" validate:"gte=0"`
}

type UpdateDeliveryZoneRequest struct {
	Name             *string `json:"name" validate:"omitempty,max=100"`
	District         *string `json:"district" validate:"omitempty,max=200"`
	Fee              *int64  `json:"fee" validate:"omitempty,gte=0"`
	EstimatedMinutes *int    `json:"estimated_minutes" validate:"omitempty,gte=0"`
	IsActive         *bool   `json:"is_active"`
}

type PickupRequest struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	OutletID      uuid.UUID  `json:"outlet_id"`
	ZoneID        *uuid.UUID `json:"zone_id,omitempty"`
	OrderID       *uuid.UUID `json:"order_id,omitempty"`
	CustomerName  string     `json:"customer_name"`
	CustomerPhone string     `json:"customer_phone"`
	Address       string     `json:"address"`
	PickupType    string     `json:"pickup_type"`
	Status        string     `json:"status"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	DeliveryFee   int64      `json:"delivery_fee"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	// Joined fields
	ZoneName   string `json:"zone_name,omitempty"`
	OutletName string `json:"outlet_name,omitempty"`
}

type CreatePickupRequestPayload struct {
	OutletID      string  `json:"outlet_id" validate:"required,uuid"`
	ZoneID        *string `json:"zone_id" validate:"omitempty,uuid"`
	CustomerName  string  `json:"customer_name" validate:"required,max=200"`
	CustomerPhone string  `json:"customer_phone" validate:"required,max=20"`
	Address       string  `json:"address" validate:"required"`
	PickupType    string  `json:"pickup_type" validate:"required,oneof=pickup delivery"`
	ScheduledAt   *string `json:"scheduled_at"`
	Notes         string  `json:"notes"`
}

type UpdatePickupRequestStatus struct {
	Status string `json:"status" validate:"required,oneof=assigned in_transit completed cancelled"`
}
