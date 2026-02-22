package service

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type DashboardService struct {
	db *pgxpool.Pool
}

func NewDashboardService(db *pgxpool.Pool) *DashboardService {
	return &DashboardService{db: db}
}

type DashboardStats struct {
	TotalTenants   int   `json:"total_tenants"`
	TotalOutlets   int   `json:"total_outlets"`
	MonthlyRevenue int64 `json:"monthly_revenue"`
	ActiveMembers  int   `json:"active_members"`
}

type MonthlyRevenue struct {
	Month   string `json:"month"`
	Revenue int64  `json:"revenue"`
}

type AdminTransaction struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	TenantName       string  `json:"tenant_name"`
	OutletID         string  `json:"outlet_id"`
	OutletName       string  `json:"outlet_name"`
	LocalOrderNumber string  `json:"local_order_number"`
	CustomerName     *string `json:"customer_name,omitempty"`
	TotalAmount      int64   `json:"total_amount"`
	PaymentStatus    string  `json:"payment_status"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
}

func (s *DashboardService) GetStats(ctx context.Context) (*DashboardStats, error) {
	q := middleware.GetQuerier(ctx, s.db)

	stats := &DashboardStats{}

	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&stats.TotalTenants)
	if err != nil {
		slog.Error("failed to count tenants", "error", err)
		return nil, apperror.Internal("failed to count tenants", err)
	}

	err = q.QueryRow(ctx, "SELECT COUNT(*) FROM outlets").Scan(&stats.TotalOutlets)
	if err != nil {
		slog.Error("failed to count outlets", "error", err)
		return nil, apperror.Internal("failed to count outlets", err)
	}

	err = q.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_amount), 0) FROM transactions WHERE created_at >= date_trunc('month', NOW())`).
		Scan(&stats.MonthlyRevenue)
	if err != nil {
		slog.Error("failed to get monthly revenue", "error", err)
		return nil, apperror.Internal("failed to get monthly revenue", err)
	}

	err = q.QueryRow(ctx, "SELECT COUNT(*) FROM members WHERE tenant_id IS NOT NULL").Scan(&stats.ActiveMembers)
	if err != nil {
		slog.Error("failed to count active members", "error", err)
		return nil, apperror.Internal("failed to count active members", err)
	}

	return stats, nil
}

func (s *DashboardService) GetRevenue(ctx context.Context) ([]MonthlyRevenue, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx,
		`SELECT to_char(date_trunc('month', created_at), 'YYYY-MM') as month,
		        COALESCE(SUM(total_amount), 0) as revenue
		 FROM transactions
		 WHERE created_at >= NOW() - INTERVAL '12 months'
		 GROUP BY date_trunc('month', created_at)
		 ORDER BY month`)
	if err != nil {
		slog.Error("failed to get revenue data", "error", err)
		return nil, apperror.Internal("failed to get revenue data", err)
	}
	defer rows.Close()

	var result []MonthlyRevenue
	for rows.Next() {
		var mr MonthlyRevenue
		if err := rows.Scan(&mr.Month, &mr.Revenue); err != nil {
			return nil, apperror.Internal("failed to scan revenue data", err)
		}
		result = append(result, mr)
	}

	if result == nil {
		result = []MonthlyRevenue{}
	}

	return result, nil
}

func (s *DashboardService) ListTransactions(ctx context.Context, params pagination.Params) ([]AdminTransaction, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&total)
	if err != nil {
		slog.Error("failed to count transactions", "error", err)
		return nil, 0, apperror.Internal("failed to count transactions", err)
	}

	rows, err := q.Query(ctx,
		`SELECT t.id::text, t.tenant_id::text, tn.name, t.outlet_id::text, o.name,
		        t.local_order_number, t.customer_name, t.total_amount,
		        t.payment_status, t.status, t.created_at::text
		 FROM transactions t
		 JOIN tenants tn ON t.tenant_id = tn.id
		 JOIN outlets o ON t.outlet_id = o.id
		 ORDER BY t.created_at DESC
		 LIMIT $1 OFFSET $2`,
		params.PerPage, params.Offset())
	if err != nil {
		slog.Error("failed to list transactions", "error", err)
		return nil, 0, apperror.Internal("failed to list transactions", err)
	}
	defer rows.Close()

	var transactions []AdminTransaction
	for rows.Next() {
		var tx AdminTransaction
		if err := rows.Scan(&tx.ID, &tx.TenantID, &tx.TenantName, &tx.OutletID, &tx.OutletName,
			&tx.LocalOrderNumber, &tx.CustomerName, &tx.TotalAmount,
			&tx.PaymentStatus, &tx.Status, &tx.CreatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan transaction", err)
		}
		transactions = append(transactions, tx)
	}

	if transactions == nil {
		transactions = []AdminTransaction{}
	}

	return transactions, total, nil
}
