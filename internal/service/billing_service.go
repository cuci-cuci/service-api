package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type BillingService struct {
	db *pgxpool.Pool
}

func NewBillingService(db *pgxpool.Pool) *BillingService {
	return &BillingService{db: db}
}

// ListPlans returns all active subscription plans.
func (s *BillingService) ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx,
		`SELECT id, name, slug, price_monthly, max_outlets, max_transactions_per_month, features, is_active, sort_order, created_at
		 FROM subscription_plans WHERE is_active = true ORDER BY sort_order`)
	if err != nil {
		return nil, apperror.Internal("failed to list plans", err)
	}
	defer rows.Close()

	var plans []domain.SubscriptionPlan
	for rows.Next() {
		var p domain.SubscriptionPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.PriceMonthly, &p.MaxOutlets,
			&p.MaxTransactionsPerMonth, &p.Features, &p.IsActive, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan plan", err)
		}
		plans = append(plans, p)
	}
	if plans == nil {
		plans = []domain.SubscriptionPlan{}
	}
	return plans, nil
}

// GetSubscriptionInfo returns the current subscription status and usage for a tenant.
func (s *BillingService) GetSubscriptionInfo(ctx context.Context, tenantID uuid.UUID) (*domain.SubscriptionInfo, error) {
	q := middleware.GetQuerier(ctx, s.db)

	// Get current subscription (or nil)
	var sub domain.TenantSubscription
	var hasSub bool
	err := q.QueryRow(ctx,
		`SELECT id, tenant_id, plan_id, status, current_period_start, current_period_end,
		        cancel_at_period_end, payment_gateway, gateway_subscription_id, created_at, updated_at
		 FROM tenant_subscriptions WHERE tenant_id = $1`, tenantID).
		Scan(&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Status, &sub.CurrentPeriodStart,
			&sub.CurrentPeriodEnd, &sub.CancelAtPeriodEnd, &sub.PaymentGateway,
			&sub.GatewaySubscriptionID, &sub.CreatedAt, &sub.UpdatedAt)
	if err == nil {
		hasSub = true
	} else if err != pgx.ErrNoRows {
		return nil, apperror.Internal("failed to get subscription", err)
	}

	// Get the plan (from subscription, or fallback to free)
	var plan domain.SubscriptionPlan
	var planQuery string
	var planArg any
	if hasSub {
		planQuery = `SELECT id, name, slug, price_monthly, max_outlets, max_transactions_per_month, features, is_active, sort_order, created_at FROM subscription_plans WHERE id = $1`
		planArg = sub.PlanID
	} else {
		planQuery = `SELECT id, name, slug, price_monthly, max_outlets, max_transactions_per_month, features, is_active, sort_order, created_at FROM subscription_plans WHERE slug = $1`
		planArg = "free"
	}

	err = q.QueryRow(ctx, planQuery, planArg).
		Scan(&plan.ID, &plan.Name, &plan.Slug, &plan.PriceMonthly, &plan.MaxOutlets,
			&plan.MaxTransactionsPerMonth, &plan.Features, &plan.IsActive, &plan.SortOrder, &plan.CreatedAt)
	if err != nil {
		slog.Error("failed to get plan", "error", err)
		return nil, apperror.Internal("failed to get plan", err)
	}

	// Get current month usage
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var usage domain.TenantUsage
	var hasUsage bool
	err = q.QueryRow(ctx,
		`SELECT id, tenant_id, period_start, period_end, transaction_count, outlet_count, updated_at
		 FROM tenant_usage WHERE tenant_id = $1 AND period_start = $2`, tenantID, periodStart).
		Scan(&usage.ID, &usage.TenantID, &usage.PeriodStart, &usage.PeriodEnd,
			&usage.TransactionCount, &usage.OutletCount, &usage.UpdatedAt)
	if err == nil {
		hasUsage = true
	} else if err != pgx.ErrNoRows {
		return nil, apperror.Internal("failed to get usage", err)
	}

	// Count current outlets
	var outletCount int
	err = q.QueryRow(ctx,
		`SELECT COUNT(*) FROM outlets WHERE tenant_id = $1 AND is_active = true`, tenantID).Scan(&outletCount)
	if err != nil {
		return nil, apperror.Internal("failed to count outlets", err)
	}

	txUsed := 0
	if hasUsage {
		txUsed = usage.TransactionCount
	}

	withinLimits := true
	if plan.MaxTransactionsPerMonth > 0 && txUsed >= plan.MaxTransactionsPerMonth {
		withinLimits = false
	}
	if plan.MaxOutlets > 0 && outletCount > plan.MaxOutlets {
		withinLimits = false
	}

	info := &domain.SubscriptionInfo{
		Plan:             plan,
		IsWithinLimits:   withinLimits,
		TransactionLimit: plan.MaxTransactionsPerMonth,
		TransactionsUsed: txUsed,
		OutletLimit:      plan.MaxOutlets,
		OutletsUsed:      outletCount,
	}
	if hasSub {
		info.Subscription = &sub
	}
	if hasUsage {
		info.Usage = &usage
	}

	return info, nil
}

// IncrementUsage increments the transaction count for the current billing period.
// Called during sync upload.
func (s *BillingService) IncrementUsage(ctx context.Context, tenantID uuid.UUID, txCount int) error {
	q := middleware.GetQuerier(ctx, s.db)

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, -1)

	_, err := q.Exec(ctx,
		`INSERT INTO tenant_usage (id, tenant_id, period_start, period_end, transaction_count, outlet_count, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 0, $6)
		 ON CONFLICT (tenant_id, period_start) DO UPDATE SET
		   transaction_count = tenant_usage.transaction_count + $5,
		   updated_at = $6`,
		uuid.New(), tenantID, periodStart, periodEnd, txCount, now)
	if err != nil {
		slog.Error("failed to increment usage", "error", err, "tenant_id", tenantID)
		return apperror.Internal("failed to update usage", err)
	}
	return nil
}

// CheckTransactionLimit checks if the tenant can create more transactions this period.
// Returns nil if OK, or an error if limit exceeded.
func (s *BillingService) CheckTransactionLimit(ctx context.Context, tenantID uuid.UUID) error {
	info, err := s.GetSubscriptionInfo(ctx, tenantID)
	if err != nil {
		return err
	}
	if !info.IsWithinLimits {
		return apperror.NewAppError(402, "Batas transaksi bulanan tercapai. Upgrade paket untuk melanjutkan.")
	}
	return nil
}

// CheckOutletLimit checks if the tenant can create more outlets.
func (s *BillingService) CheckOutletLimit(ctx context.Context, tenantID uuid.UUID) error {
	info, err := s.GetSubscriptionInfo(ctx, tenantID)
	if err != nil {
		return err
	}
	if info.Plan.MaxOutlets > 0 && info.OutletsUsed >= info.Plan.MaxOutlets {
		return apperror.NewAppError(402, "Batas outlet tercapai. Upgrade paket untuk menambah outlet.")
	}
	return nil
}

// AssignFreePlan creates a subscription entry on the free plan for a new tenant.
func (s *BillingService) AssignFreePlan(ctx context.Context, tenantID uuid.UUID) error {
	q := middleware.GetQuerier(ctx, s.db)

	var planID uuid.UUID
	err := q.QueryRow(ctx, `SELECT id FROM subscription_plans WHERE slug = 'free' AND is_active = true`).Scan(&planID)
	if err != nil {
		slog.Error("failed to find free plan", "error", err)
		return apperror.Internal("failed to find free plan", err)
	}

	now := time.Now()
	periodEnd := now.AddDate(100, 0, 0) // Free plan never expires

	_, err = q.Exec(ctx,
		`INSERT INTO tenant_subscriptions (id, tenant_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at)
		 VALUES ($1, $2, $3, 'active', $4, $5, $4, $4)
		 ON CONFLICT (tenant_id) DO NOTHING`,
		uuid.New(), tenantID, planID, now, periodEnd)
	if err != nil {
		slog.Error("failed to assign free plan", "error", err)
		return apperror.Internal("failed to assign free plan", err)
	}
	return nil
}
