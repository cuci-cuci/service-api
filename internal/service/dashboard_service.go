package service

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
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
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	TenantName       string    `json:"tenant_name"`
	OutletID         uuid.UUID `json:"outlet_id"`
	OutletName       string    `json:"outlet_name"`
	LocalOrderNumber string    `json:"local_order_number"`
	CustomerName     *string   `json:"customer_name,omitempty"`
	TotalAmount      int64     `json:"total_amount"`
	PaymentStatus    string    `json:"payment_status"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
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
		`SELECT t.id, t.tenant_id, tn.name, t.outlet_id, o.name,
		        t.local_order_number, t.customer_name, t.total_amount,
		        t.payment_status, t.status, t.created_at
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

type TenantHealth struct {
	TenantID         uuid.UUID  `json:"tenant_id"`
	TenantName       string     `json:"tenant_name"`
	TotalOutlets     int        `json:"total_outlets"`
	TotalUsers       int        `json:"total_users"`
	MonthRevenue     int64      `json:"month_revenue"`
	PrevMonthRevenue int64      `json:"prev_month_revenue"`
	GrowthPct        float64    `json:"growth_pct"`
	TodayTx          int        `json:"today_tx"`
	MonthTx          int        `json:"month_tx"`
	LastTxAt         *time.Time `json:"last_tx_at"`
	IsActive         bool       `json:"is_active"`
}

func (s *DashboardService) GetTenantHealth(ctx context.Context) ([]TenantHealth, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT
			t.id,
			t.name,
			COALESCE(oc.cnt, 0) AS total_outlets,
			COALESCE(uc.cnt, 0) AS total_users,
			COALESCE(cur.revenue, 0) AS month_revenue,
			COALESCE(prev.revenue, 0) AS prev_month_revenue,
			COALESCE(cur.today_tx, 0) AS today_tx,
			COALESCE(cur.month_tx, 0) AS month_tx,
			cur.last_tx_at
		FROM tenants t
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) AS cnt FROM outlets GROUP BY tenant_id
		) oc ON oc.tenant_id = t.id
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) AS cnt FROM users WHERE tenant_id IS NOT NULL GROUP BY tenant_id
		) uc ON uc.tenant_id = t.id
		LEFT JOIN (
			SELECT
				tenant_id,
				COALESCE(SUM(total_amount), 0) AS revenue,
				COUNT(*) AS month_tx,
				COUNT(*) FILTER (WHERE created_at >= date_trunc('day', NOW())) AS today_tx,
				MAX(created_at) AS last_tx_at
			FROM transactions
			WHERE created_at >= date_trunc('month', NOW())
			GROUP BY tenant_id
		) cur ON cur.tenant_id = t.id
		LEFT JOIN (
			SELECT
				tenant_id,
				COALESCE(SUM(total_amount), 0) AS revenue
			FROM transactions
			WHERE created_at >= date_trunc('month', NOW()) - INTERVAL '1 month'
			  AND created_at < date_trunc('month', NOW())
			GROUP BY tenant_id
		) prev ON prev.tenant_id = t.id
		ORDER BY month_revenue DESC
	`)
	if err != nil {
		slog.Error("failed to get tenant health", "error", err)
		return nil, apperror.Internal("failed to get tenant health", err)
	}
	defer rows.Close()

	var result []TenantHealth
	for rows.Next() {
		var th TenantHealth
		if err := rows.Scan(
			&th.TenantID, &th.TenantName,
			&th.TotalOutlets, &th.TotalUsers,
			&th.MonthRevenue, &th.PrevMonthRevenue,
			&th.TodayTx, &th.MonthTx,
			&th.LastTxAt,
		); err != nil {
			return nil, apperror.Internal("failed to scan tenant health", err)
		}

		// Calculate growth percentage
		if th.PrevMonthRevenue > 0 {
			th.GrowthPct = math.Round((float64(th.MonthRevenue-th.PrevMonthRevenue)/float64(th.PrevMonthRevenue)*100)*100) / 100
		}

		// Active = has transactions in last 7 days
		if th.LastTxAt != nil {
			th.IsActive = th.LastTxAt.After(time.Now().AddDate(0, 0, -7))
		}

		result = append(result, th)
	}

	if result == nil {
		result = []TenantHealth{}
	}

	return result, nil
}

func (s *DashboardService) TenantHealthOverview(ctx context.Context) ([]domain.TenantHealth, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT
			t.id,
			t.name,
			ss.last_sync_at,
			COALESCE(tx.week_tx_count, 0),
			COALESCE(o.active_orders, 0)
		FROM tenants t
		LEFT JOIN (
			SELECT tenant_id, MAX(completed_at) AS last_sync_at
			FROM sync_sessions
			WHERE status = 'completed'
			GROUP BY tenant_id
		) ss ON ss.tenant_id = t.id
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) AS week_tx_count
			FROM transactions
			WHERE created_at >= NOW() - INTERVAL '7 days'
			  AND status = 'completed'
			GROUP BY tenant_id
		) tx ON tx.tenant_id = t.id
		LEFT JOIN (
			SELECT tenant_id, COUNT(*) AS active_orders
			FROM orders
			WHERE status NOT IN ('completed', 'cancelled', 'picked_up')
			GROUP BY tenant_id
		) o ON o.tenant_id = t.id
		ORDER BY t.name
	`)
	if err != nil {
		slog.Error("failed to get tenant health overview", "error", err)
		return nil, apperror.Internal("failed to get tenant health overview", err)
	}
	defer rows.Close()

	now := time.Now()
	var result []domain.TenantHealth
	for rows.Next() {
		var th domain.TenantHealth
		if err := rows.Scan(
			&th.TenantID, &th.TenantName,
			&th.LastSyncAt,
			&th.WeekTxCount, &th.ActiveOrders,
		); err != nil {
			return nil, apperror.Internal("failed to scan tenant health overview", err)
		}

		// Determine health status
		if th.LastSyncAt == nil {
			th.Status = "critical"
		} else {
			hoursSinceSync := now.Sub(*th.LastSyncAt).Hours()
			if hoursSinceSync <= 2 && th.WeekTxCount > 0 {
				th.Status = "healthy"
			} else if hoursSinceSync <= 24 {
				th.Status = "warning"
			} else {
				th.Status = "critical"
			}
		}
		if th.WeekTxCount == 0 && th.Status != "critical" {
			th.Status = "warning"
		}

		result = append(result, th)
	}

	if result == nil {
		result = []domain.TenantHealth{}
	}

	return result, nil
}
