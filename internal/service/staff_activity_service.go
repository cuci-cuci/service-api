package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type StaffActivityService struct {
	db *pgxpool.Pool
}

func NewStaffActivityService(db *pgxpool.Pool) *StaffActivityService {
	return &StaffActivityService{db: db}
}

type StaffActivity struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	UserID        uuid.UUID  `json:"user_id"`
	UserName      string     `json:"user_name"`
	ActivityType  string     `json:"activity_type"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty"`
	ReferenceType *string    `json:"reference_type,omitempty"`
	Amount        int64      `json:"amount"`
	Notes         *string    `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type StaffSummary struct {
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	TxCount      int       `json:"tx_count"`
	Revenue      int64     `json:"revenue"`
	CancelCount  int       `json:"cancel_count"`
	RefundCount  int       `json:"refund_count"`
	CancelAmount int64     `json:"cancel_amount"`
	RefundAmount int64     `json:"refund_amount"`
}

func (s *StaffActivityService) LogActivity(ctx context.Context, tenantID, userID uuid.UUID, activityType string, refID *uuid.UUID, refType *string, amount int64, notes *string) error {
	q := middleware.GetQuerier(ctx, s.db)
	_, err := q.Exec(ctx, `
		INSERT INTO staff_activity_logs (tenant_id, user_id, activity_type, reference_id, reference_type, amount, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, userID, activityType, refID, refType, amount, notes)
	if err != nil {
		return apperror.Internal("failed to log staff activity", err)
	}
	return nil
}

func (s *StaffActivityService) ListActivities(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit int) ([]StaffActivity, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT sal.id, sal.tenant_id, sal.user_id, u.name, sal.activity_type,
			sal.reference_id, sal.reference_type, sal.amount, sal.notes, sal.created_at
		FROM staff_activity_logs sal
		JOIN users u ON sal.user_id = u.id
		WHERE sal.tenant_id = $1
	`
	args := []any{tenantID}
	if userID != nil {
		query += " AND sal.user_id = $2"
		args = append(args, *userID)
	}
	query += " ORDER BY sal.created_at DESC LIMIT $" + string(rune('0'+len(args)+1))

	// Fix: use proper arg indexing
	if userID != nil {
		query = `
			SELECT sal.id, sal.tenant_id, sal.user_id, u.name, sal.activity_type,
				sal.reference_id, sal.reference_type, sal.amount, sal.notes, sal.created_at
			FROM staff_activity_logs sal
			JOIN users u ON sal.user_id = u.id
			WHERE sal.tenant_id = $1 AND sal.user_id = $2
			ORDER BY sal.created_at DESC LIMIT $3`
		args = []any{tenantID, *userID, limit}
	} else {
		query = `
			SELECT sal.id, sal.tenant_id, sal.user_id, u.name, sal.activity_type,
				sal.reference_id, sal.reference_type, sal.amount, sal.notes, sal.created_at
			FROM staff_activity_logs sal
			JOIN users u ON sal.user_id = u.id
			WHERE sal.tenant_id = $1
			ORDER BY sal.created_at DESC LIMIT $2`
		args = []any{tenantID, limit}
	}

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to list activities", err)
	}
	defer rows.Close()

	var result []StaffActivity
	for rows.Next() {
		var a StaffActivity
		if err := rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.UserName, &a.ActivityType,
			&a.ReferenceID, &a.ReferenceType, &a.Amount, &a.Notes, &a.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan activity", err)
		}
		result = append(result, a)
	}
	if result == nil {
		result = []StaffActivity{}
	}
	return result, nil
}

func (s *StaffActivityService) GetStaffSummaries(ctx context.Context, tenantID uuid.UUID, days int) ([]StaffSummary, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT u.id, u.name,
			COALESCE(COUNT(CASE WHEN sal.activity_type = 'transaction' THEN 1 END), 0) as tx_count,
			COALESCE(SUM(CASE WHEN sal.activity_type = 'transaction' THEN sal.amount END), 0) as revenue,
			COALESCE(COUNT(CASE WHEN sal.activity_type = 'cancel' THEN 1 END), 0) as cancel_count,
			COALESCE(COUNT(CASE WHEN sal.activity_type = 'refund' THEN 1 END), 0) as refund_count,
			COALESCE(SUM(CASE WHEN sal.activity_type = 'cancel' THEN sal.amount END), 0) as cancel_amount,
			COALESCE(SUM(CASE WHEN sal.activity_type = 'refund' THEN sal.amount END), 0) as refund_amount
		FROM users u
		LEFT JOIN staff_activity_logs sal ON u.id = sal.user_id
			AND sal.tenant_id = $1
			AND sal.created_at >= NOW() - make_interval(days => $2)
		WHERE u.tenant_id = $1 AND u.role = 'cashier'
		GROUP BY u.id, u.name
		ORDER BY revenue DESC
	`, tenantID, days)
	if err != nil {
		return nil, apperror.Internal("failed to get staff summaries", err)
	}
	defer rows.Close()

	var result []StaffSummary
	for rows.Next() {
		var s StaffSummary
		if err := rows.Scan(&s.UserID, &s.Name, &s.TxCount, &s.Revenue,
			&s.CancelCount, &s.RefundCount, &s.CancelAmount, &s.RefundAmount); err != nil {
			return nil, apperror.Internal("failed to scan staff summary", err)
		}
		result = append(result, s)
	}
	if result == nil {
		result = []StaffSummary{}
	}
	return result, nil
}
