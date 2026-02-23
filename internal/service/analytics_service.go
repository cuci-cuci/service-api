package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type DailyRevenuePoint struct {
	Date         string `json:"date"`
	Revenue      int64  `json:"revenue"`
	Transactions int    `json:"transactions"`
}

type AnalyticsService struct {
	db *pgxpool.Pool
}

func NewAnalyticsService(db *pgxpool.Pool) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func (s *AnalyticsService) RevenueByTenant(ctx context.Context, startDate, endDate string) ([]domain.RevenueByTenantResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT t.tenant_id, te.name as tenant_name,
		       COALESCE(SUM(t.total_amount), 0) as revenue,
		       COUNT(t.id) as transaction_count
		FROM transactions t
		JOIN tenants te ON t.tenant_id = te.id
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if startDate != "" {
		query += " AND t.created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += " AND t.created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, endDate)
		argIdx++
	}

	query += " GROUP BY t.tenant_id, te.name ORDER BY revenue DESC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to query revenue", err)
	}
	defer rows.Close()

	var results []domain.RevenueByTenantResponse
	for rows.Next() {
		var r domain.RevenueByTenantResponse
		if err := rows.Scan(&r.TenantID, &r.TenantName, &r.Revenue, &r.TxCount); err != nil {
			return nil, apperror.Internal("failed to scan revenue", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []domain.RevenueByTenantResponse{}
	}

	return results, nil
}

func (s *AnalyticsService) RevenueByOutlet(ctx context.Context, tenantID uuid.UUID, startDate, endDate string) ([]domain.RevenueByOutletResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT t.outlet_id, o.name as outlet_name,
		       COALESCE(SUM(t.total_amount), 0) as revenue,
		       COUNT(t.id) as transaction_count
		FROM transactions t
		JOIN outlets o ON t.outlet_id = o.id
		WHERE t.tenant_id = $1
	`
	args := []any{tenantID}
	argIdx := 2

	if startDate != "" {
		query += " AND t.created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += " AND t.created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, endDate)
		argIdx++
	}

	query += " GROUP BY t.outlet_id, o.name ORDER BY revenue DESC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to query revenue by outlet", err)
	}
	defer rows.Close()

	var results []domain.RevenueByOutletResponse
	for rows.Next() {
		var r domain.RevenueByOutletResponse
		if err := rows.Scan(&r.OutletID, &r.OutletName, &r.Revenue, &r.TxCount); err != nil {
			return nil, apperror.Internal("failed to scan revenue by outlet", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []domain.RevenueByOutletResponse{}
	}

	return results, nil
}

func (s *AnalyticsService) TransactionStats(ctx context.Context, startDate, endDate string) (*domain.TransactionStatsResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT COUNT(id) as total_transactions,
		       COALESCE(SUM(total_amount), 0) as total_revenue,
		       COALESCE(AVG(total_amount), 0) as avg_transaction
		FROM transactions
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if startDate != "" {
		query += " AND created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += " AND created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, endDate)
		argIdx++
	}

	var stats domain.TransactionStatsResponse
	err := q.QueryRow(ctx, query, args...).
		Scan(&stats.TotalTransactions, &stats.TotalRevenue, &stats.AvgTransaction)
	if err != nil {
		return nil, apperror.Internal("failed to query transaction stats", err)
	}

	return &stats, nil
}

func (s *AnalyticsService) RevenueByService(ctx context.Context, tenantID uuid.UUID, startDate, endDate string) ([]domain.RevenueByServiceResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT item->>'serviceName' as service_name,
		       COALESCE(SUM((item->>'subtotal')::bigint), 0) as revenue,
		       COALESCE(SUM((item->>'quantity')::int), 0) as quantity
		FROM transactions t,
		     jsonb_array_elements(t.items) as item
		WHERE t.tenant_id = $1 AND t.status = 'completed'
	`
	args := []any{tenantID}
	argIdx := 2

	if startDate != "" {
		query += " AND t.created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += " AND t.created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, endDate)
		argIdx++
	}

	query += " GROUP BY item->>'serviceName' ORDER BY revenue DESC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to query revenue by service", err)
	}
	defer rows.Close()

	var results []domain.RevenueByServiceResponse
	for rows.Next() {
		var r domain.RevenueByServiceResponse
		if err := rows.Scan(&r.ServiceName, &r.Revenue, &r.Quantity); err != nil {
			return nil, apperror.Internal("failed to scan revenue by service", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []domain.RevenueByServiceResponse{}
	}

	return results, nil
}

func (s *AnalyticsService) RevenueByPaymentMethod(ctx context.Context, tenantID uuid.UUID, startDate, endDate string) ([]domain.RevenueByPaymentMethodResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT pay->>'methodName' as method_name,
		       COALESCE(pay->>'methodType', 'cash') as method_type,
		       COALESCE(SUM((pay->>'amount')::bigint), 0) as revenue,
		       COUNT(DISTINCT t.id) as transaction_count
		FROM transactions t,
		     jsonb_array_elements(t.payments) as pay
		WHERE t.tenant_id = $1 AND t.status = 'completed'
	`
	args := []any{tenantID}
	argIdx := 2

	if startDate != "" {
		query += " AND t.created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += " AND t.created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, endDate)
		argIdx++
	}

	query += " GROUP BY pay->>'methodName', pay->>'methodType' ORDER BY revenue DESC"

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal("failed to query revenue by payment method", err)
	}
	defer rows.Close()

	var results []domain.RevenueByPaymentMethodResponse
	for rows.Next() {
		var r domain.RevenueByPaymentMethodResponse
		if err := rows.Scan(&r.MethodName, &r.MethodType, &r.Revenue, &r.TxCount); err != nil {
			return nil, apperror.Internal("failed to scan revenue by payment method", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []domain.RevenueByPaymentMethodResponse{}
	}

	return results, nil
}

func (s *AnalyticsService) DailyRevenue(ctx context.Context, tenantID uuid.UUID, days int) ([]DailyRevenuePoint, error) {
	q := middleware.GetQuerier(ctx, s.db)

	query := `
		SELECT to_char(date_trunc('day', t.created_at), 'YYYY-MM-DD') as date,
		       COALESCE(SUM(t.total_amount), 0) as revenue,
		       COUNT(*) as transactions
		FROM transactions t
		WHERE t.tenant_id = $1 AND t.created_at >= NOW() - ($2 || ' days')::INTERVAL
		GROUP BY date_trunc('day', t.created_at)
		ORDER BY date
	`

	rows, err := q.Query(ctx, query, tenantID, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, apperror.Internal("failed to query daily revenue", err)
	}
	defer rows.Close()

	var results []DailyRevenuePoint
	for rows.Next() {
		var p DailyRevenuePoint
		if err := rows.Scan(&p.Date, &p.Revenue, &p.Transactions); err != nil {
			return nil, apperror.Internal("failed to scan daily revenue", err)
		}
		results = append(results, p)
	}

	if results == nil {
		results = []DailyRevenuePoint{}
	}

	return results, nil
}
