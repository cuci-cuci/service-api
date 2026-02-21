package service

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type AnalyticsService struct {
	db *pgxpool.Pool
}

func NewAnalyticsService(db *pgxpool.Pool) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func (s *AnalyticsService) RevenueByTenant(ctx context.Context, startDate, endDate string) ([]domain.RevenueByTenantResponse, error) {
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

	rows, err := s.db.Query(ctx, query, args...)
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

func (s *AnalyticsService) TransactionStats(ctx context.Context, startDate, endDate string) (*domain.TransactionStatsResponse, error) {
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
	err := s.db.QueryRow(ctx, query, args...).
		Scan(&stats.TotalTransactions, &stats.TotalRevenue, &stats.AvgTransaction)
	if err != nil {
		return nil, apperror.Internal("failed to query transaction stats", err)
	}

	return &stats, nil
}
