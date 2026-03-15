package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type OwnerDashboardService struct {
	db *pgxpool.Pool
}

func NewOwnerDashboardService(db *pgxpool.Pool) *OwnerDashboardService {
	return &OwnerDashboardService{db: db}
}

type OwnerDashboardSummary struct {
	TodayRevenue      int64   `json:"today_revenue"`
	TodayTransactions int     `json:"today_transactions"`
	WeekRevenue       int64   `json:"week_revenue"`
	MonthRevenue      int64   `json:"month_revenue"`
	MonthExpenses     int64   `json:"month_expenses"`
	MonthProfit       int64   `json:"month_profit"`
	MarginPercent     float64 `json:"margin_percent"`
	PrevMonthRevenue  int64   `json:"prev_month_revenue"`
	RevenueGrowthPct  float64 `json:"revenue_growth_pct"`
}

type CashierPerformance struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	TxCount     int       `json:"tx_count"`
	Revenue     int64     `json:"revenue"`
	AvgRevenue  int64     `json:"avg_revenue"`
}

type CustomerInsights struct {
	NewCustomers       int `json:"new_customers"`
	ReturningCustomers int `json:"returning_customers"`
	TotalUnique        int `json:"total_unique"`
}

type DateRangeSummary struct {
	Revenue          int64   `json:"revenue"`
	Transactions     int     `json:"transactions"`
	Expenses         int64   `json:"expenses"`
	Profit           int64   `json:"profit"`
	PrevRevenue      int64   `json:"prev_revenue"`
	PrevTransactions int     `json:"prev_transactions"`
	PrevExpenses     int64   `json:"prev_expenses"`
	PrevProfit       int64   `json:"prev_profit"`
	RevenueGrowthPct float64 `json:"revenue_growth_pct"`
}

type DashboardGoal struct {
	ID          uuid.UUID `json:"id"`
	GoalType    string    `json:"goal_type"`
	TargetValue int64     `json:"target_value"`
	CurrentValue int64   `json:"current_value"`
	IsActive    bool      `json:"is_active"`
}

func (s *OwnerDashboardService) GetSummary(ctx context.Context, tenantID uuid.UUID, outletID *uuid.UUID) (*OwnerDashboardSummary, error) {
	q := middleware.GetQuerier(ctx, s.db)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Week start (Monday)
	weekday := now.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-int(weekday-1), 0, 0, 0, 0, now.Location())

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	prevMonthStart := monthStart.AddDate(0, -1, 0)
	prevMonthEnd := monthStart.Add(-time.Second)

	summary := &OwnerDashboardSummary{}

	outletFilter := ""
	args := []any{tenantID, todayStart, weekStart, monthStart}
	if outletID != nil {
		outletFilter = " AND outlet_id = $5"
		args = append(args, *outletID)
	}

	// Revenue aggregation in one query
	err := q.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= $2 THEN total_amount END), 0) as today_revenue,
			COALESCE(COUNT(CASE WHEN created_at >= $2 THEN 1 END), 0) as today_tx,
			COALESCE(SUM(CASE WHEN created_at >= $3 THEN total_amount END), 0) as week_revenue,
			COALESCE(SUM(CASE WHEN created_at >= $4 THEN total_amount END), 0) as month_revenue
		FROM transactions
		WHERE tenant_id = $1 AND status = 'completed'`+outletFilter+`
	`, args...).Scan(
		&summary.TodayRevenue,
		&summary.TodayTransactions,
		&summary.WeekRevenue,
		&summary.MonthRevenue,
	)
	if err != nil {
		return nil, apperror.Internal("failed to get revenue summary", err)
	}

	// Previous month revenue
	prevArgs := []any{tenantID, prevMonthStart, prevMonthEnd}
	prevFilter := ""
	if outletID != nil {
		prevFilter = " AND outlet_id = $4"
		prevArgs = append(prevArgs, *outletID)
	}
	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM transactions
		WHERE tenant_id = $1 AND status = 'completed'
			AND created_at >= $2 AND created_at <= $3`+prevFilter+`
	`, prevArgs...).Scan(&summary.PrevMonthRevenue)
	if err != nil {
		return nil, apperror.Internal("failed to get prev month revenue", err)
	}

	// Month expenses (outlet-scoped if available)
	expArgs := []any{tenantID, monthStart.Format("2006-01-02"), now.AddDate(0, 0, 1).Format("2006-01-02")}
	expFilter := ""
	if outletID != nil {
		expFilter = " AND outlet_id = $4"
		expArgs = append(expArgs, *outletID)
	}
	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE tenant_id = $1
			AND expense_date >= $2 AND expense_date < $3`+expFilter+`
	`, expArgs...).Scan(&summary.MonthExpenses)
	if err != nil {
		summary.MonthExpenses = 0
	}

	summary.MonthProfit = summary.MonthRevenue - summary.MonthExpenses
	if summary.MonthRevenue > 0 {
		summary.MarginPercent = float64(summary.MonthProfit) / float64(summary.MonthRevenue) * 100
	}
	if summary.PrevMonthRevenue > 0 {
		summary.RevenueGrowthPct = float64(summary.MonthRevenue-summary.PrevMonthRevenue) / float64(summary.PrevMonthRevenue) * 100
	}

	return summary, nil
}

func (s *OwnerDashboardService) GetCashierPerformance(ctx context.Context, tenantID uuid.UUID, days int, outletID *uuid.UUID) ([]CashierPerformance, error) {
	q := middleware.GetQuerier(ctx, s.db)

	outletFilter := ""
	args := []any{tenantID, strconv.Itoa(days)}
	if outletID != nil {
		outletFilter = fmt.Sprintf(" AND t.outlet_id = $%d", len(args)+1)
		args = append(args, *outletID)
	}

	rows, err := q.Query(ctx, `
		SELECT u.id, u.name, COUNT(t.id), COALESCE(SUM(t.total_amount), 0),
			COALESCE(AVG(t.total_amount)::bigint, 0)
		FROM transactions t
		JOIN users u ON t.created_by = u.id
		WHERE t.tenant_id = $1 AND t.status = 'completed'
			AND t.created_at >= NOW() - ($2::text || ' days')::INTERVAL`+outletFilter+`
		GROUP BY u.id, u.name
		ORDER BY SUM(t.total_amount) DESC
	`, args...)
	if err != nil {
		return nil, apperror.Internal("failed to get cashier performance", err)
	}
	defer rows.Close()

	var results []CashierPerformance
	for rows.Next() {
		var cp CashierPerformance
		if err := rows.Scan(&cp.UserID, &cp.Name, &cp.TxCount, &cp.Revenue, &cp.AvgRevenue); err != nil {
			return nil, apperror.Internal("failed to scan cashier performance", err)
		}
		results = append(results, cp)
	}

	if results == nil {
		results = []CashierPerformance{}
	}

	return results, nil
}

func (s *OwnerDashboardService) GetCustomerInsights(ctx context.Context, tenantID uuid.UUID, days int) (*CustomerInsights, error) {
	q := middleware.GetQuerier(ctx, s.db)

	insights := &CustomerInsights{}

	err := q.QueryRow(ctx, `
		WITH customer_first_tx AS (
			SELECT COALESCE(customer_name, '') as cname,
				MIN(created_at) as first_tx
			FROM transactions
			WHERE tenant_id = $1 AND status = 'completed'
				AND customer_name IS NOT NULL AND customer_name != ''
			GROUP BY customer_name
		)
		SELECT
			COALESCE(COUNT(CASE WHEN first_tx >= NOW() - ($2::text || ' days')::INTERVAL THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN first_tx < NOW() - ($2::text || ' days')::INTERVAL THEN 1 END), 0),
			COUNT(*)
		FROM customer_first_tx
	`, tenantID, days).Scan(&insights.NewCustomers, &insights.ReturningCustomers, &insights.TotalUnique)
	if err != nil {
		return nil, apperror.Internal("failed to get customer insights", err)
	}

	return insights, nil
}

func (s *OwnerDashboardService) GetGoals(ctx context.Context, tenantID uuid.UUID) ([]DashboardGoal, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx, `
		SELECT id, goal_type, target_value, is_active
		FROM dashboard_goals
		WHERE tenant_id = $1
		ORDER BY goal_type
	`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to get goals", err)
	}
	defer rows.Close()

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var goals []DashboardGoal
	for rows.Next() {
		var g DashboardGoal
		if err := rows.Scan(&g.ID, &g.GoalType, &g.TargetValue, &g.IsActive); err != nil {
			return nil, apperror.Internal("failed to scan goal", err)
		}
		goals = append(goals, g)
	}

	if goals == nil {
		goals = []DashboardGoal{}
	}

	// Compute current values
	for i := range goals {
		switch goals[i].GoalType {
		case "daily_revenue":
			_ = q.QueryRow(ctx, `
				SELECT COALESCE(SUM(total_amount), 0)
				FROM transactions
				WHERE tenant_id = $1 AND status = 'completed' AND created_at >= $2
			`, tenantID, todayStart).Scan(&goals[i].CurrentValue)
		case "monthly_revenue":
			_ = q.QueryRow(ctx, `
				SELECT COALESCE(SUM(total_amount), 0)
				FROM transactions
				WHERE tenant_id = $1 AND status = 'completed' AND created_at >= $2
			`, tenantID, monthStart).Scan(&goals[i].CurrentValue)
		case "daily_transactions":
			var count int
			_ = q.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM transactions
				WHERE tenant_id = $1 AND status = 'completed' AND created_at >= $2
			`, tenantID, todayStart).Scan(&count)
			goals[i].CurrentValue = int64(count)
		}
	}

	return goals, nil
}

func (s *OwnerDashboardService) UpsertGoal(ctx context.Context, tenantID uuid.UUID, goalType string, targetValue int64) error {
	q := middleware.GetQuerier(ctx, s.db)

	_, err := q.Exec(ctx, `
		INSERT INTO dashboard_goals (tenant_id, goal_type, target_value)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, goal_type)
		DO UPDATE SET target_value = $3, updated_at = NOW()
	`, tenantID, goalType, targetValue)
	if err != nil {
		return apperror.Internal("failed to upsert goal", err)
	}

	return nil
}

// GetSummaryByDateRange returns revenue, transactions, expenses and profit for
// a given date range plus the equivalent "previous period" of the same
// duration immediately before startDate, enabling period-over-period comparison.
func (s *OwnerDashboardService) GetSummaryByDateRange(ctx context.Context, tenantID uuid.UUID, startDate, endDate string, outletID *uuid.UUID) (*DateRangeSummary, error) {
	q := middleware.GetQuerier(ctx, s.db)

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, apperror.Validation("invalid start date format, use YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, apperror.Validation("invalid end date format, use YYYY-MM-DD")
	}
	if end.Before(start) {
		return nil, apperror.Validation("end date must be on or after start date")
	}

	// The "end" in the query is exclusive (< endDate + 1 day).
	endExclusive := end.AddDate(0, 0, 1).Format("2006-01-02")

	// Previous period: same duration, immediately before startDate.
	duration := end.Sub(start) + 24*time.Hour // inclusive range
	prevStart := start.Add(-duration).Format("2006-01-02")
	prevEnd := startDate // exclusive

	summary := &DateRangeSummary{}

	// --- Current period ---
	txArgs := []any{tenantID, startDate, endExclusive}
	txFilter := ""
	if outletID != nil {
		txFilter = " AND outlet_id = $4"
		txArgs = append(txArgs, *outletID)
	}

	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM transactions
		WHERE tenant_id = $1 AND status = 'completed'
			AND created_at >= $2::date AND created_at < $3::date`+txFilter+`
	`, txArgs...).Scan(&summary.Revenue, &summary.Transactions)
	if err != nil {
		return nil, apperror.Internal("failed to get range revenue", err)
	}

	expArgs := []any{tenantID, startDate, endExclusive}
	expFilter := ""
	if outletID != nil {
		expFilter = " AND outlet_id = $4"
		expArgs = append(expArgs, *outletID)
	}
	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE tenant_id = $1
			AND expense_date >= $2 AND expense_date < $3`+expFilter+`
	`, expArgs...).Scan(&summary.Expenses)
	if err != nil {
		summary.Expenses = 0
	}
	summary.Profit = summary.Revenue - summary.Expenses

	// --- Previous period ---
	prevTxArgs := []any{tenantID, prevStart, prevEnd}
	prevTxFilter := ""
	if outletID != nil {
		prevTxFilter = " AND outlet_id = $4"
		prevTxArgs = append(prevTxArgs, *outletID)
	}

	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM transactions
		WHERE tenant_id = $1 AND status = 'completed'
			AND created_at >= $2::date AND created_at < $3::date`+prevTxFilter+`
	`, prevTxArgs...).Scan(&summary.PrevRevenue, &summary.PrevTransactions)
	if err != nil {
		return nil, apperror.Internal("failed to get prev range revenue", err)
	}

	prevExpArgs := []any{tenantID, prevStart, prevEnd}
	prevExpFilter := ""
	if outletID != nil {
		prevExpFilter = " AND outlet_id = $4"
		prevExpArgs = append(prevExpArgs, *outletID)
	}
	err = q.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE tenant_id = $1
			AND expense_date >= $2 AND expense_date < $3`+prevExpFilter+`
	`, prevExpArgs...).Scan(&summary.PrevExpenses)
	if err != nil {
		summary.PrevExpenses = 0
	}
	summary.PrevProfit = summary.PrevRevenue - summary.PrevExpenses

	// Growth percentage
	if summary.PrevRevenue > 0 {
		summary.RevenueGrowthPct = float64(summary.Revenue-summary.PrevRevenue) / float64(summary.PrevRevenue) * 100
	}

	return summary, nil
}
