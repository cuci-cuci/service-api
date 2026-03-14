package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type DeliveryService struct {
	db *pgxpool.Pool
}

func NewDeliveryService(db *pgxpool.Pool) *DeliveryService {
	return &DeliveryService{db: db}
}

// --- Delivery Zones ---

// ListZones returns active zones, optionally filtered by outlet.
func (s *DeliveryService) ListZones(ctx context.Context, tenantID uuid.UUID, outletID *uuid.UUID) ([]domain.DeliveryZone, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `SELECT id, tenant_id, outlet_id, name, district, fee, estimated_minutes, is_active, created_at, updated_at
		FROM delivery_zones WHERE tenant_id = $1 AND is_active = true`
	args := []any{tenantID}
	argIdx := 2

	if outletID != nil {
		query += fmt.Sprintf(" AND outlet_id = $%d", argIdx)
		args = append(args, *outletID)
		argIdx++
	}
	_ = argIdx
	query += " ORDER BY name ASC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to list delivery zones", err)
	}
	defer rows.Close()

	var zones []domain.DeliveryZone
	for rows.Next() {
		var z domain.DeliveryZone
		if err := rows.Scan(&z.ID, &z.TenantID, &z.OutletID, &z.Name, &z.District, &z.Fee, &z.EstimatedMinutes, &z.IsActive, &z.CreatedAt, &z.UpdatedAt); err != nil {
			return nil, apperror.Internal("failed to scan delivery zone", err)
		}
		zones = append(zones, z)
	}
	if zones == nil {
		zones = []domain.DeliveryZone{}
	}
	return zones, nil
}

// ListAllZones returns all zones (including inactive) for owner management, optionally filtered by outlet.
func (s *DeliveryService) ListAllZones(ctx context.Context, tenantID uuid.UUID, outletID *uuid.UUID) ([]domain.DeliveryZone, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `SELECT id, tenant_id, outlet_id, name, district, fee, estimated_minutes, is_active, created_at, updated_at
		FROM delivery_zones WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if outletID != nil {
		query += fmt.Sprintf(" AND outlet_id = $%d", argIdx)
		args = append(args, *outletID)
		argIdx++
	}
	_ = argIdx
	query += " ORDER BY name ASC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to list delivery zones", err)
	}
	defer rows.Close()

	var zones []domain.DeliveryZone
	for rows.Next() {
		var z domain.DeliveryZone
		if err := rows.Scan(&z.ID, &z.TenantID, &z.OutletID, &z.Name, &z.District, &z.Fee, &z.EstimatedMinutes, &z.IsActive, &z.CreatedAt, &z.UpdatedAt); err != nil {
			return nil, apperror.Internal("failed to scan delivery zone", err)
		}
		zones = append(zones, z)
	}
	if zones == nil {
		zones = []domain.DeliveryZone{}
	}
	return zones, nil
}

func (s *DeliveryService) CreateZone(ctx context.Context, tenantID uuid.UUID, req domain.CreateDeliveryZoneRequest) (*domain.DeliveryZone, error) {
	q := middleware.GetQuerier(ctx, s.db)

	outletID, parseErr := uuid.Parse(req.OutletID)
	if parseErr != nil {
		return nil, apperror.Validation("invalid outlet_id")
	}

	var z domain.DeliveryZone
	err := q.QueryRow(ctx, `
		INSERT INTO delivery_zones (tenant_id, outlet_id, name, district, fee, estimated_minutes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, outlet_id, name, district, fee, estimated_minutes, is_active, created_at, updated_at
	`, tenantID, outletID, req.Name, req.District, req.Fee, req.EstimatedMinutes).Scan(
		&z.ID, &z.TenantID, &z.OutletID, &z.Name, &z.District, &z.Fee, &z.EstimatedMinutes, &z.IsActive, &z.CreatedAt, &z.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create delivery zone", err)
	}

	return &z, nil
}

func (s *DeliveryService) UpdateZone(ctx context.Context, tenantID, zoneID uuid.UUID, req domain.UpdateDeliveryZoneRequest) (*domain.DeliveryZone, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var z domain.DeliveryZone
	err := q.QueryRow(ctx,
		`SELECT id, tenant_id, outlet_id, name, district, fee, estimated_minutes, is_active, created_at, updated_at
		 FROM delivery_zones WHERE id = $1 AND tenant_id = $2`, zoneID, tenantID).
		Scan(&z.ID, &z.TenantID, &z.OutletID, &z.Name, &z.District, &z.Fee, &z.EstimatedMinutes, &z.IsActive, &z.CreatedAt, &z.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("delivery zone not found")
		}
		return nil, apperror.Internal("failed to get delivery zone", err)
	}

	if req.Name != nil {
		z.Name = *req.Name
	}
	if req.District != nil {
		z.District = req.District
	}
	if req.Fee != nil {
		z.Fee = *req.Fee
	}
	if req.EstimatedMinutes != nil {
		z.EstimatedMinutes = *req.EstimatedMinutes
	}
	if req.IsActive != nil {
		z.IsActive = *req.IsActive
	}
	z.UpdatedAt = time.Now()

	_, err = q.Exec(ctx,
		`UPDATE delivery_zones SET name=$1, district=$2, fee=$3, estimated_minutes=$4, is_active=$5, updated_at=$6
		 WHERE id=$7 AND tenant_id=$8`,
		z.Name, z.District, z.Fee, z.EstimatedMinutes, z.IsActive, z.UpdatedAt, z.ID, z.TenantID)
	if err != nil {
		return nil, apperror.Internal("failed to update delivery zone", err)
	}

	return &z, nil
}

func (s *DeliveryService) DeleteZone(ctx context.Context, tenantID, zoneID uuid.UUID) error {
	q := middleware.GetQuerier(ctx, s.db)

	result, err := q.Exec(ctx, `UPDATE delivery_zones SET is_active = false, updated_at = $1 WHERE id = $2 AND tenant_id = $3`, time.Now(), zoneID, tenantID)
	if err != nil {
		return apperror.Internal("failed to delete delivery zone", err)
	}
	if result.RowsAffected() == 0 {
		return apperror.NotFound("delivery zone not found")
	}
	return nil
}

// GetZoneFee returns just the fee for a zone (for auto-fee calculation).
func (s *DeliveryService) GetZoneFee(ctx context.Context, tenantID, zoneID uuid.UUID) (int64, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var fee int64
	err := q.QueryRow(ctx,
		`SELECT fee FROM delivery_zones WHERE id = $1 AND tenant_id = $2 AND is_active = true`,
		zoneID, tenantID).Scan(&fee)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, apperror.NotFound("delivery zone not found")
		}
		return 0, apperror.Internal("failed to get zone fee", err)
	}
	return fee, nil
}

// --- Pickup Requests ---

func (s *DeliveryService) ListPickupRequests(ctx context.Context, tenantID uuid.UUID, outletID *uuid.UUID, status string) ([]domain.PickupRequest, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT pr.id, pr.tenant_id, pr.outlet_id, pr.zone_id, pr.order_id,
			pr.customer_name, pr.customer_phone, pr.address, pr.pickup_type,
			pr.status, pr.scheduled_at, pr.completed_at, pr.notes, pr.delivery_fee,
			pr.created_at, pr.updated_at,
			COALESCE(dz.name, ''), COALESCE(o.name, '')
		FROM pickup_requests pr
		LEFT JOIN delivery_zones dz ON pr.zone_id = dz.id
		LEFT JOIN outlets o ON pr.outlet_id = o.id
		WHERE pr.tenant_id = $1
	`
	args := []any{tenantID}
	argIdx := 2

	if outletID != nil {
		query += fmt.Sprintf(" AND pr.outlet_id = $%d", argIdx)
		args = append(args, *outletID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND pr.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	_ = argIdx
	query += " ORDER BY pr.created_at DESC LIMIT 100"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to list pickup requests", err)
	}
	defer rows.Close()

	var result []domain.PickupRequest
	for rows.Next() {
		var pr domain.PickupRequest
		if err := rows.Scan(
			&pr.ID, &pr.TenantID, &pr.OutletID, &pr.ZoneID, &pr.OrderID,
			&pr.CustomerName, &pr.CustomerPhone, &pr.Address, &pr.PickupType,
			&pr.Status, &pr.ScheduledAt, &pr.CompletedAt, &pr.Notes, &pr.DeliveryFee,
			&pr.CreatedAt, &pr.UpdatedAt,
			&pr.ZoneName, &pr.OutletName,
		); err != nil {
			return nil, apperror.Internal("failed to scan pickup request", err)
		}
		result = append(result, pr)
	}
	if result == nil {
		result = []domain.PickupRequest{}
	}
	return result, nil
}

func (s *DeliveryService) CreatePickupRequest(ctx context.Context, tenantID uuid.UUID, req domain.CreatePickupRequestPayload) (*domain.PickupRequest, error) {
	q := middleware.GetQuerier(ctx, s.db)

	outletID, err := uuid.Parse(req.OutletID)
	if err != nil {
		return nil, apperror.Validation("invalid outlet_id")
	}

	var zoneID *uuid.UUID
	var deliveryFee int64
	if req.ZoneID != nil && *req.ZoneID != "" {
		id, parseErr := uuid.Parse(*req.ZoneID)
		if parseErr != nil {
			return nil, apperror.Validation("invalid zone_id")
		}
		zoneID = &id
		// Auto-fetch zone fee
		fee, feeErr := s.GetZoneFee(ctx, tenantID, id)
		if feeErr != nil {
			return nil, feeErr
		}
		deliveryFee = fee
	}

	var scheduledAt *time.Time
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		t, parseErr := time.Parse(time.RFC3339, *req.ScheduledAt)
		if parseErr != nil {
			return nil, apperror.Validation("invalid scheduled_at format, use RFC3339")
		}
		scheduledAt = &t
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	var pr domain.PickupRequest
	err = q.QueryRow(ctx, `
		INSERT INTO pickup_requests (tenant_id, outlet_id, zone_id, customer_name, customer_phone, address, pickup_type, scheduled_at, notes, delivery_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, tenant_id, outlet_id, zone_id, order_id, customer_name, customer_phone, address, pickup_type, status, scheduled_at, completed_at, notes, delivery_fee, created_at, updated_at
	`, tenantID, outletID, zoneID, req.CustomerName, req.CustomerPhone, req.Address, req.PickupType, scheduledAt, notes, deliveryFee).Scan(
		&pr.ID, &pr.TenantID, &pr.OutletID, &pr.ZoneID, &pr.OrderID,
		&pr.CustomerName, &pr.CustomerPhone, &pr.Address, &pr.PickupType,
		&pr.Status, &pr.ScheduledAt, &pr.CompletedAt, &pr.Notes, &pr.DeliveryFee,
		&pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create pickup request", err)
	}

	return &pr, nil
}

func (s *DeliveryService) UpdatePickupRequestStatus(ctx context.Context, tenantID, requestID uuid.UUID, req domain.UpdatePickupRequestStatus) error {
	q := middleware.GetQuerier(ctx, s.db)

	var completedAt *time.Time
	if req.Status == "completed" {
		now := time.Now()
		completedAt = &now
	}

	tag, err := q.Exec(ctx, `
		UPDATE pickup_requests SET status = $1, completed_at = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`, req.Status, completedAt, requestID, tenantID)
	if err != nil {
		return apperror.Internal("failed to update pickup request status", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("pickup request not found")
	}
	return nil
}
