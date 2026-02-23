package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

func generateTrackingToken() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// validTransitions defines the allowed status transitions for orders.
var validTransitions = map[string][]string{
	"received": {"washing", "cancelled"},
	"washing":  {"drying", "ironing", "cancelled"},
	"drying":   {"ironing", "done", "cancelled"},
	"ironing":  {"done", "cancelled"},
	"done":     {"picked_up", "cancelled"},
}

type OrderService struct {
	db *pgxpool.Pool
}

func NewOrderService(db *pgxpool.Pool) *OrderService {
	return &OrderService{db: db}
}

func (s *OrderService) List(ctx context.Context, tenantID uuid.UUID, params pagination.Params, status string, outletID string) ([]domain.OrderDetailResponse, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	countQuery := `SELECT COUNT(*) FROM orders WHERE tenant_id = $1`
	dataQuery := `
		SELECT o.id, o.transaction_id, o.tenant_id, o.outlet_id, o.status,
		       o.estimated_completion_at, o.completed_at, o.picked_up_at, o.notes,
		       o.customer_phone, o.tracking_token,
		       o.created_by, o.updated_by, o.created_at, o.updated_at,
		       t.customer_name, COALESCE(t.total_amount, 0), COALESCE(t.local_order_number, '')
		FROM orders o
		LEFT JOIN transactions t ON o.transaction_id = t.id
		WHERE o.tenant_id = $1
	`

	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		filter := fmt.Sprintf(" AND o.status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		dataQuery += filter
		args = append(args, status)
		argIdx++
	}
	if outletID != "" {
		oid, err := uuid.Parse(outletID)
		if err != nil {
			return nil, 0, apperror.Validation("invalid outlet_id")
		}
		filter := fmt.Sprintf(" AND o.outlet_id = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND outlet_id = $%d", argIdx)
		dataQuery += filter
		args = append(args, oid)
		argIdx++
	}

	var total int
	err := q.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count orders", err)
	}

	dataQuery += " ORDER BY o.created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := q.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to query orders", err)
	}
	defer rows.Close()

	var results []domain.OrderDetailResponse
	for rows.Next() {
		var r domain.OrderDetailResponse
		if err := rows.Scan(
			&r.ID, &r.TransactionID, &r.TenantID, &r.OutletID, &r.Status,
			&r.EstimatedCompletionAt, &r.CompletedAt, &r.PickedUpAt, &r.Notes,
			&r.CustomerPhone, &r.TrackingToken,
			&r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt,
			&r.CustomerName, &r.TotalAmount, &r.OrderNumber,
		); err != nil {
			return nil, 0, apperror.Internal("failed to scan order", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []domain.OrderDetailResponse{}
	}

	return results, total, nil
}

func (s *OrderService) GetByID(ctx context.Context, id uuid.UUID) (*domain.OrderDetailResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var r domain.OrderDetailResponse
	err := q.QueryRow(ctx, `
		SELECT o.id, o.transaction_id, o.tenant_id, o.outlet_id, o.status,
		       o.estimated_completion_at, o.completed_at, o.picked_up_at, o.notes,
		       o.customer_phone, o.tracking_token,
		       o.created_by, o.updated_by, o.created_at, o.updated_at,
		       t.customer_name, COALESCE(t.total_amount, 0), COALESCE(t.local_order_number, '')
		FROM orders o
		LEFT JOIN transactions t ON o.transaction_id = t.id
		WHERE o.id = $1
	`, id).Scan(
		&r.ID, &r.TransactionID, &r.TenantID, &r.OutletID, &r.Status,
		&r.EstimatedCompletionAt, &r.CompletedAt, &r.PickedUpAt, &r.Notes,
		&r.CustomerPhone, &r.TrackingToken,
		&r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt,
		&r.CustomerName, &r.TotalAmount, &r.OrderNumber,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("order not found")
		}
		return nil, apperror.Internal("failed to get order", err)
	}

	// Get transaction if linked
	if r.TransactionID != nil {
		var tx domain.Transaction
		err := q.QueryRow(ctx, `
			SELECT id, tenant_id, outlet_id, local_order_number, customer_name,
			       items, subtotal, discount_amount, tax_amount, total_amount,
			       payment_status, payments, status, config_version_id, notes,
			       created_by, created_at, synced_at
			FROM transactions WHERE id = $1
		`, *r.TransactionID).Scan(
			&tx.ID, &tx.TenantID, &tx.OutletID, &tx.LocalOrderNumber, &tx.CustomerName,
			&tx.Items, &tx.Subtotal, &tx.DiscountAmount, &tx.TaxAmount, &tx.TotalAmount,
			&tx.PaymentStatus, &tx.Payments, &tx.Status, &tx.ConfigVersionID, &tx.Notes,
			&tx.CreatedBy, &tx.CreatedAt, &tx.SyncedAt,
		)
		if err == nil {
			r.Transaction = &tx
		}
	}

	// Get status logs
	logRows, err := q.Query(ctx, `
		SELECT id, order_id, from_status, to_status, changed_by, notes, created_at
		FROM order_status_logs
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return nil, apperror.Internal("failed to get status logs", err)
	}
	defer logRows.Close()

	var logs []domain.OrderStatusLog
	for logRows.Next() {
		var l domain.OrderStatusLog
		if err := logRows.Scan(&l.ID, &l.OrderID, &l.FromStatus, &l.ToStatus, &l.ChangedBy, &l.Notes, &l.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan status log", err)
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []domain.OrderStatusLog{}
	}
	r.StatusLogs = logs

	return &r, nil
}

func (s *OrderService) Create(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, req domain.CreateOrderRequest) (*domain.Order, error) {
	q := middleware.GetQuerier(ctx, s.db)

	outletID, err := uuid.Parse(req.OutletID)
	if err != nil {
		return nil, apperror.Validation("invalid outlet_id")
	}

	var txID *uuid.UUID
	if req.TransactionID != nil && *req.TransactionID != "" {
		parsed, err := uuid.Parse(*req.TransactionID)
		if err != nil {
			return nil, apperror.Validation("invalid transaction_id")
		}
		txID = &parsed
	}

	var estimatedAt *time.Time
	if req.EstimatedDurationHours > 0 {
		t := time.Now().Add(time.Duration(req.EstimatedDurationHours) * time.Hour)
		estimatedAt = &t
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	var custPhone *string
	if req.CustomerPhone != "" {
		custPhone = &req.CustomerPhone
	}

	trackingToken := generateTrackingToken()

	orderID := uuid.New()
	now := time.Now()

	_, err = q.Exec(ctx, `
		INSERT INTO orders (id, transaction_id, tenant_id, outlet_id, status,
		                    estimated_completion_at, notes, customer_phone, tracking_token,
		                    created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'received', $5, $6, $7, $8, $9, $9, $10, $10)
	`, orderID, txID, tenantID, outletID, estimatedAt, notes, custPhone, trackingToken, userID, now)
	if err != nil {
		return nil, apperror.Internal("failed to create order", err)
	}

	// Create initial status log
	_, err = q.Exec(ctx, `
		INSERT INTO order_status_logs (id, order_id, from_status, to_status, changed_by, notes, created_at)
		VALUES ($1, $2, NULL, 'received', $3, $4, $5)
	`, uuid.New(), orderID, userID, notes, now)
	if err != nil {
		return nil, apperror.Internal("failed to create status log", err)
	}

	order := &domain.Order{
		ID:                    orderID,
		TransactionID:         txID,
		TenantID:              tenantID,
		OutletID:              outletID,
		Status:                "received",
		EstimatedCompletionAt: estimatedAt,
		Notes:                 notes,
		CustomerPhone:         custPhone,
		TrackingToken:         &trackingToken,
		CreatedBy:             userID,
		UpdatedBy:             userID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	return order, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, req domain.UpdateOrderStatusRequest) (*domain.Order, error) {
	q := middleware.GetQuerier(ctx, s.db)

	// Get current order
	var order domain.Order
	err := q.QueryRow(ctx, `
		SELECT id, transaction_id, tenant_id, outlet_id, status,
		       estimated_completion_at, completed_at, picked_up_at, notes,
		       customer_phone, tracking_token,
		       created_by, updated_by, created_at, updated_at
		FROM orders WHERE id = $1
	`, id).Scan(
		&order.ID, &order.TransactionID, &order.TenantID, &order.OutletID, &order.Status,
		&order.EstimatedCompletionAt, &order.CompletedAt, &order.PickedUpAt, &order.Notes,
		&order.CustomerPhone, &order.TrackingToken,
		&order.CreatedBy, &order.UpdatedBy, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("order not found")
		}
		return nil, apperror.Internal("failed to get order", err)
	}

	// Validate transition
	allowed, ok := validTransitions[order.Status]
	if !ok {
		return nil, apperror.Validation(fmt.Sprintf("cannot transition from status '%s'", order.Status))
	}
	isValid := false
	for _, s := range allowed {
		if s == req.Status {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, apperror.Validation(fmt.Sprintf("invalid transition from '%s' to '%s'", order.Status, req.Status))
	}

	now := time.Now()
	fromStatus := order.Status

	// Update order
	updateQuery := `UPDATE orders SET status = $1, updated_by = $2, updated_at = $3`
	args := []any{req.Status, userID, now}
	argIdx := 4

	if req.Status == "done" {
		updateQuery += fmt.Sprintf(", completed_at = $%d", argIdx)
		args = append(args, now)
		argIdx++
	}
	if req.Status == "picked_up" {
		updateQuery += fmt.Sprintf(", picked_up_at = $%d", argIdx)
		args = append(args, now)
		argIdx++
	}

	updateQuery += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = q.Exec(ctx, updateQuery, args...)
	if err != nil {
		return nil, apperror.Internal("failed to update order status", err)
	}

	// Create status log
	var logNotes *string
	if req.Notes != "" {
		logNotes = &req.Notes
	}
	_, err = q.Exec(ctx, `
		INSERT INTO order_status_logs (id, order_id, from_status, to_status, changed_by, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New(), id, fromStatus, req.Status, userID, logNotes, now)
	if err != nil {
		return nil, apperror.Internal("failed to create status log", err)
	}

	// Return updated order
	order.Status = req.Status
	order.UpdatedBy = userID
	order.UpdatedAt = now
	if req.Status == "done" {
		order.CompletedAt = &now
	}
	if req.Status == "picked_up" {
		order.PickedUpAt = &now
	}

	return &order, nil
}

func (s *OrderService) ListActive(ctx context.Context, tenantID uuid.UUID, outletID string) ([]domain.OrderDetailResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT o.id, o.transaction_id, o.tenant_id, o.outlet_id, o.status,
		       o.estimated_completion_at, o.completed_at, o.picked_up_at, o.notes,
		       o.customer_phone, o.tracking_token,
		       o.created_by, o.updated_by, o.created_at, o.updated_at,
		       t.customer_name, COALESCE(t.total_amount, 0), COALESCE(t.local_order_number, '')
		FROM orders o
		LEFT JOIN transactions t ON o.transaction_id = t.id
		WHERE o.tenant_id = $1 AND o.status NOT IN ('picked_up', 'cancelled')
	`
	args := []any{tenantID}

	if outletID != "" {
		oid, err := uuid.Parse(outletID)
		if err != nil {
			return nil, apperror.Validation("invalid outlet_id")
		}
		query += " AND o.outlet_id = $2"
		args = append(args, oid)
	}

	query += " ORDER BY o.created_at DESC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to query active orders", err)
	}
	defer rows.Close()

	var results []domain.OrderDetailResponse
	for rows.Next() {
		var r domain.OrderDetailResponse
		if err := rows.Scan(
			&r.ID, &r.TransactionID, &r.TenantID, &r.OutletID, &r.Status,
			&r.EstimatedCompletionAt, &r.CompletedAt, &r.PickedUpAt, &r.Notes,
			&r.CustomerPhone, &r.TrackingToken,
			&r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt,
			&r.CustomerName, &r.TotalAmount, &r.OrderNumber,
		); err != nil {
			return nil, apperror.Internal("failed to scan active order", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []domain.OrderDetailResponse{}
	}

	return results, nil
}

func (s *OrderService) CreateFromTransaction(ctx context.Context, tenantID uuid.UUID, tx domain.Transaction, userID uuid.UUID) error {
	q := middleware.GetQuerier(ctx, s.db)

	orderID := uuid.New()
	now := time.Now()

	trackingToken := generateTrackingToken()

	_, err := q.Exec(ctx, `
		INSERT INTO orders (id, transaction_id, tenant_id, outlet_id, status,
		                    notes, tracking_token, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'received', $5, $6, $7, $7, $8, $8)
		ON CONFLICT (transaction_id) DO NOTHING
	`, orderID, tx.ID, tenantID, tx.OutletID, tx.Notes, trackingToken, userID, now)
	if err != nil {
		return apperror.Internal("failed to create order from transaction", err)
	}

	// Create initial status log (only if order was actually inserted)
	_, err = q.Exec(ctx, `
		INSERT INTO order_status_logs (id, order_id, from_status, to_status, changed_by, created_at)
		SELECT $1, $2, NULL, 'received', $3, $4
		WHERE EXISTS (SELECT 1 FROM orders WHERE id = $2 AND created_at = $4)
	`, uuid.New(), orderID, userID, now)
	if err != nil {
		return apperror.Internal("failed to create initial status log", err)
	}

	return nil
}

// GetByTrackingToken returns an order by its public tracking token (no auth required).
func (s *OrderService) GetByTrackingToken(ctx context.Context, token string) (*domain.OrderTrackingResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var r domain.OrderTrackingResponse
	err := q.QueryRow(ctx, `
		SELECT o.id, o.status, o.estimated_completion_at, o.completed_at, o.picked_up_at,
		       o.tracking_token, o.created_at,
		       t.customer_name, COALESCE(t.local_order_number, ''),
		       tn.name
		FROM orders o
		LEFT JOIN transactions t ON o.transaction_id = t.id
		JOIN tenants tn ON o.tenant_id = tn.id
		WHERE o.tracking_token = $1
	`, token).Scan(
		&r.ID, &r.Status, &r.EstimatedCompletionAt, &r.CompletedAt, &r.PickedUpAt,
		&r.TrackingToken, &r.CreatedAt,
		&r.CustomerName, &r.OrderNumber,
		&r.BusinessName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("pesanan tidak ditemukan")
		}
		return nil, apperror.Internal("failed to get order by tracking token", err)
	}

	// Get status logs
	logRows, err := q.Query(ctx, `
		SELECT to_status, created_at
		FROM order_status_logs
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, r.ID)
	if err != nil {
		return nil, apperror.Internal("failed to get status logs", err)
	}
	defer logRows.Close()

	for logRows.Next() {
		var entry domain.TrackingStatusEntry
		if err := logRows.Scan(&entry.Status, &entry.Timestamp); err != nil {
			return nil, apperror.Internal("failed to scan status log", err)
		}
		r.StatusHistory = append(r.StatusHistory, entry)
	}
	if r.StatusHistory == nil {
		r.StatusHistory = []domain.TrackingStatusEntry{}
	}

	return &r, nil
}
