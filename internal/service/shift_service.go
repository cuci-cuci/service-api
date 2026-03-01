package service

import (
	"context"
	"encoding/json"
	"log/slog"
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

type ShiftService struct {
	db *pgxpool.Pool
}

func NewShiftService(db *pgxpool.Pool) *ShiftService {
	return &ShiftService{db: db}
}

func (s *ShiftService) OpenShift(ctx context.Context, tenantID uuid.UUID, cashierID uuid.UUID, req domain.OpenShiftRequest) (*domain.Shift, error) {
	q := middleware.GetQuerier(ctx, s.db)

	outletID, err := uuid.Parse(req.OutletID)
	if err != nil {
		return nil, apperror.Validation("invalid outlet_id")
	}

	// Check no open shift for this cashier
	var existing int
	err = q.QueryRow(ctx, `SELECT COUNT(*) FROM shifts WHERE cashier_id = $1 AND status = 'open'`, cashierID).Scan(&existing)
	if err != nil {
		return nil, apperror.Internal("failed to check existing shift", err)
	}
	if existing > 0 {
		return nil, apperror.Conflict("cashier already has an open shift")
	}

	shiftID := uuid.New()
	now := time.Now()

	_, err = q.Exec(ctx, `
		INSERT INTO shifts (id, tenant_id, outlet_id, cashier_id, opening_cash, status, opened_at, created_at)
		VALUES ($1, $2, $3, $4, $5, 'open', $6, $6)
	`, shiftID, tenantID, outletID, cashierID, req.OpeningCash, now)
	if err != nil {
		// Handle unique constraint from idx_shifts_one_open_per_cashier
		if strings.Contains(err.Error(), "idx_shifts_one_open_per_cashier") || strings.Contains(err.Error(), "duplicate key") {
			return nil, apperror.Conflict("cashier already has an open shift")
		}
		return nil, apperror.Internal("failed to create shift", err)
	}

	shift := &domain.Shift{
		ID:          shiftID,
		TenantID:    tenantID,
		OutletID:    outletID,
		CashierID:   cashierID,
		OpeningCash: req.OpeningCash,
		Status:      "open",
		OpenedAt:    now,
		CreatedAt:   now,
	}

	return shift, nil
}

func (s *ShiftService) CloseShift(ctx context.Context, shiftID uuid.UUID, cashierID uuid.UUID, req domain.CloseShiftRequest) (*domain.Shift, error) {
	q := middleware.GetQuerier(ctx, s.db)

	// Get the shift
	var shift domain.Shift
	err := q.QueryRow(ctx, `
		SELECT id, tenant_id, outlet_id, cashier_id, opening_cash, status, opened_at, created_at
		FROM shifts WHERE id = $1 AND status = 'open'
	`, shiftID).Scan(
		&shift.ID, &shift.TenantID, &shift.OutletID, &shift.CashierID,
		&shift.OpeningCash, &shift.Status, &shift.OpenedAt, &shift.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("open shift not found")
		}
		return nil, apperror.Internal("failed to get shift", err)
	}

	// Verify the cashier owns this shift
	if shift.CashierID != cashierID {
		return nil, apperror.Forbidden("you can only close your own shift")
	}

	// Calculate expected cash from cash transactions during this shift
	var expectedCash int64
	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(t.total_amount), 0)
		FROM shift_transactions st
		JOIN transactions t ON st.transaction_id = t.id
		WHERE st.shift_id = $1
		AND t.status = 'completed'
		AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(t.payments) p
			WHERE LOWER(COALESCE(p->>'methodType', p->>'type', p->>'method')) = 'cash'
		)
	`, shiftID).Scan(&expectedCash)
	if err != nil {
		slog.Warn("failed to calculate cash-only expected amount, falling back to total",
			"shift_id", shiftID, "error", err)
		expectedCash = 0
		if fallbackErr := q.QueryRow(ctx, `
			SELECT COALESCE(SUM(t.total_amount), 0)
			FROM shift_transactions st
			JOIN transactions t ON st.transaction_id = t.id
			WHERE st.shift_id = $1 AND t.status = 'completed'
		`, shiftID).Scan(&expectedCash); fallbackErr != nil {
			slog.Error("failed to calculate expected cash fallback",
				"shift_id", shiftID, "error", fallbackErr)
		}
	}

	expectedCashTotal := shift.OpeningCash + expectedCash
	cashDifference := req.ClosingCash - expectedCashTotal
	now := time.Now()

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	_, err = q.Exec(ctx, `
		UPDATE shifts
		SET closing_cash = $1, expected_cash = $2, cash_difference = $3,
		    status = 'closed', closed_at = $4, notes = $5
		WHERE id = $6
	`, req.ClosingCash, expectedCashTotal, cashDifference, now, notes, shiftID)
	if err != nil {
		return nil, apperror.Internal("failed to close shift", err)
	}

	shift.ClosingCash = &req.ClosingCash
	shift.ExpectedCash = &expectedCashTotal
	shift.CashDifference = &cashDifference
	shift.Status = "closed"
	shift.ClosedAt = &now
	shift.Notes = notes

	return &shift, nil
}

func (s *ShiftService) GetCurrentShift(ctx context.Context, cashierID uuid.UUID) (*domain.Shift, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var shift domain.Shift
	err := q.QueryRow(ctx, `
		SELECT id, tenant_id, outlet_id, cashier_id, opening_cash,
		       closing_cash, expected_cash, cash_difference, status,
		       opened_at, closed_at, notes, created_at
		FROM shifts
		WHERE cashier_id = $1 AND status = 'open'
		ORDER BY opened_at DESC
		LIMIT 1
	`, cashierID).Scan(
		&shift.ID, &shift.TenantID, &shift.OutletID, &shift.CashierID,
		&shift.OpeningCash, &shift.ClosingCash, &shift.ExpectedCash,
		&shift.CashDifference, &shift.Status, &shift.OpenedAt,
		&shift.ClosedAt, &shift.Notes, &shift.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Internal("failed to get current shift", err)
	}

	return &shift, nil
}

func (s *ShiftService) ListShifts(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.Shift, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM shifts WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count shifts", err)
	}

	query := `
		SELECT id, tenant_id, outlet_id, cashier_id, opening_cash,
		       closing_cash, expected_cash, cash_difference, status,
		       opened_at, closed_at, notes, created_at
		FROM shifts
		WHERE tenant_id = $1
		ORDER BY opened_at DESC
		LIMIT $` + strconv.Itoa(2) + ` OFFSET $` + strconv.Itoa(3)

	rows, err := q.Query(ctx, query, tenantID, params.PerPage, params.Offset())
	if err != nil {
		return nil, 0, apperror.Internal("failed to query shifts", err)
	}
	defer rows.Close()

	var results []domain.Shift
	for rows.Next() {
		var shift domain.Shift
		if err := rows.Scan(
			&shift.ID, &shift.TenantID, &shift.OutletID, &shift.CashierID,
			&shift.OpeningCash, &shift.ClosingCash, &shift.ExpectedCash,
			&shift.CashDifference, &shift.Status, &shift.OpenedAt,
			&shift.ClosedAt, &shift.Notes, &shift.CreatedAt,
		); err != nil {
			return nil, 0, apperror.Internal("failed to scan shift", err)
		}
		results = append(results, shift)
	}

	if results == nil {
		results = []domain.Shift{}
	}

	return results, total, nil
}

func (s *ShiftService) GetShiftSummary(ctx context.Context, shiftID uuid.UUID) (*domain.ShiftSummaryResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var resp domain.ShiftSummaryResponse
	err := q.QueryRow(ctx, `
		SELECT s.id, s.tenant_id, s.outlet_id, s.cashier_id, s.opening_cash,
		       s.closing_cash, s.expected_cash, s.cash_difference, s.status,
		       s.opened_at, s.closed_at, s.notes, s.created_at,
		       u.name
		FROM shifts s
		JOIN users u ON s.cashier_id = u.id
		WHERE s.id = $1
	`, shiftID).Scan(
		&resp.ID, &resp.TenantID, &resp.OutletID, &resp.CashierID,
		&resp.OpeningCash, &resp.ClosingCash, &resp.ExpectedCash,
		&resp.CashDifference, &resp.Status, &resp.OpenedAt,
		&resp.ClosedAt, &resp.Notes, &resp.CreatedAt,
		&resp.CashierName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("shift not found")
		}
		return nil, apperror.Internal("failed to get shift summary", err)
	}

	// Get transaction count and total revenue
	err = q.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(t.total_amount), 0)
		FROM shift_transactions st
		JOIN transactions t ON st.transaction_id = t.id
		WHERE st.shift_id = $1 AND t.status = 'completed'
	`, shiftID).Scan(&resp.TransactionCount, &resp.TotalRevenue)
	if err != nil {
		return nil, apperror.Internal("failed to get shift transaction stats", err)
	}

	// Get payment breakdown
	rows, err := q.Query(ctx, `
		SELECT COALESCE(p.value->>'methodType', p.value->>'type', p.value->>'method') AS payment_type,
		       COUNT(*) AS count,
		       COALESCE(SUM((p.value->>'amount')::bigint), 0) AS amount
		FROM shift_transactions st
		JOIN transactions t ON st.transaction_id = t.id,
		     jsonb_array_elements(t.payments) AS p(value)
		WHERE st.shift_id = $1 AND t.status = 'completed'
		GROUP BY COALESCE(p.value->>'methodType', p.value->>'type', p.value->>'method')
		ORDER BY amount DESC
	`, shiftID)
	if err != nil {
		// If payment breakdown query fails, return empty breakdown
		resp.PaymentBreakdown = []domain.ShiftPaymentBreakdown{}
		return &resp, nil
	}
	defer rows.Close()

	var breakdown []domain.ShiftPaymentBreakdown
	for rows.Next() {
		var pb domain.ShiftPaymentBreakdown
		var rawAmount json.Number
		if err := rows.Scan(&pb.PaymentType, &pb.Count, &rawAmount); err != nil {
			continue
		}
		pb.Amount, _ = rawAmount.Int64()
		breakdown = append(breakdown, pb)
	}
	if breakdown == nil {
		breakdown = []domain.ShiftPaymentBreakdown{}
	}
	resp.PaymentBreakdown = breakdown

	return &resp, nil
}
