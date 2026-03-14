package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type AdminOrderService struct {
	db *pgxpool.Pool
}

func NewAdminOrderService(db *pgxpool.Pool) *AdminOrderService {
	return &AdminOrderService{db: db}
}

type AdminOrder struct {
	ID                    uuid.UUID  `json:"id"`
	TenantID              uuid.UUID  `json:"tenant_id"`
	TenantName            string     `json:"tenant_name"`
	OutletName            string     `json:"outlet_name"`
	Status                string     `json:"status"`
	CustomerName          *string    `json:"customer_name,omitempty"`
	CustomerPhone         *string    `json:"customer_phone,omitempty"`
	TotalAmount           int64      `json:"total_amount"`
	OrderNumber           string     `json:"order_number"`
	DeliveryType          string     `json:"delivery_type"`
	DeliveryAddress       *string    `json:"delivery_address,omitempty"`
	DeliveryFee           int64      `json:"delivery_fee"`
	ScheduledPickupAt     *time.Time `json:"scheduled_pickup_at,omitempty"`
	EstimatedCompletionAt *time.Time `json:"estimated_completion_at,omitempty"`
	TrackingToken         *string    `json:"tracking_token,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

type AdminOrderSummary struct {
	TotalOrders   int `json:"total_orders"`
	ActiveOrders  int `json:"active_orders"`
	DeliveryCount int `json:"delivery_count"`
}

func (s *AdminOrderService) List(ctx context.Context, params pagination.Params, status string, deliveryOnly bool) ([]AdminOrder, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	countQuery := `SELECT COUNT(*) FROM orders o`
	dataQuery := `
		SELECT o.id, o.tenant_id, t.name, COALESCE(ol.name, ''),
			o.status, tr.customer_name, o.customer_phone,
			COALESCE(tr.total_amount, 0), COALESCE(tr.local_order_number, ''),
			COALESCE(o.delivery_type, 'pickup'), o.delivery_address, COALESCE(o.delivery_fee, 0),
			o.scheduled_pickup_at, o.estimated_completion_at, o.tracking_token,
			o.created_at
		FROM orders o
		JOIN tenants t ON o.tenant_id = t.id
		LEFT JOIN outlets ol ON o.outlet_id = ol.id
		LEFT JOIN transactions tr ON o.transaction_id = tr.id
	`

	where := " WHERE 1=1"
	args := []any{}
	argIdx := 1

	if status != "" {
		where += fmt.Sprintf(" AND o.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if deliveryOnly {
		where += " AND o.delivery_type = 'delivery'"
	}

	countQuery += where
	dataQuery += where + fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	args = append(args, params.PerPage, params.Offset())

	var total int
	if err := q.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, apperror.Internal("failed to count orders", err)
	}

	rows, err := q.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list orders", err)
	}
	defer rows.Close()

	var result []AdminOrder
	for rows.Next() {
		var o AdminOrder
		if err := rows.Scan(&o.ID, &o.TenantID, &o.TenantName, &o.OutletName,
			&o.Status, &o.CustomerName, &o.CustomerPhone,
			&o.TotalAmount, &o.OrderNumber,
			&o.DeliveryType, &o.DeliveryAddress, &o.DeliveryFee,
			&o.ScheduledPickupAt, &o.EstimatedCompletionAt, &o.TrackingToken,
			&o.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan order", err)
		}
		result = append(result, o)
	}
	if result == nil {
		result = []AdminOrder{}
	}
	return result, total, nil
}

func (s *AdminOrderService) Summary(ctx context.Context) (*AdminOrderSummary, error) {
	q := middleware.GetQuerier(ctx, s.db)

	summary := &AdminOrderSummary{}
	_ = q.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&summary.TotalOrders)
	_ = q.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders WHERE status NOT IN ('picked_up', 'cancelled')
	`).Scan(&summary.ActiveOrders)
	_ = q.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders WHERE delivery_type = 'delivery' AND status NOT IN ('picked_up', 'cancelled')
	`).Scan(&summary.DeliveryCount)

	return summary, nil
}
