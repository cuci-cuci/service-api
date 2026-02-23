package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SubscriptionPlan struct {
	ID                      uuid.UUID       `json:"id"`
	Name                    string          `json:"name"`
	Slug                    string          `json:"slug"`
	PriceMonthly            int             `json:"price_monthly"`
	MaxOutlets              int             `json:"max_outlets"`
	MaxTransactionsPerMonth int             `json:"max_transactions_per_month"`
	Features                json.RawMessage `json:"features"`
	IsActive                bool            `json:"is_active"`
	SortOrder               int             `json:"sort_order"`
	CreatedAt               time.Time       `json:"created_at"`
}

type TenantSubscription struct {
	ID                    uuid.UUID  `json:"id"`
	TenantID              uuid.UUID  `json:"tenant_id"`
	PlanID                uuid.UUID  `json:"plan_id"`
	Status                string     `json:"status"`
	CurrentPeriodStart    time.Time  `json:"current_period_start"`
	CurrentPeriodEnd      time.Time  `json:"current_period_end"`
	CancelAtPeriodEnd     bool       `json:"cancel_at_period_end"`
	PaymentGateway        *string    `json:"payment_gateway,omitempty"`
	GatewaySubscriptionID *string    `json:"gateway_subscription_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type TenantUsage struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	TransactionCount int       `json:"transaction_count"`
	OutletCount      int       `json:"outlet_count"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Invoice struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	SubscriptionID    *uuid.UUID `json:"subscription_id,omitempty"`
	Amount            int        `json:"amount"`
	Status            string     `json:"status"`
	PaymentGateway    *string    `json:"payment_gateway,omitempty"`
	GatewayInvoiceID  *string    `json:"gateway_invoice_id,omitempty"`
	GatewayPaymentURL *string    `json:"gateway_payment_url,omitempty"`
	Description       *string    `json:"description,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// SubscriptionInfo is a combined view returned to clients.
type SubscriptionInfo struct {
	Plan              SubscriptionPlan `json:"plan"`
	Subscription      *TenantSubscription `json:"subscription,omitempty"`
	Usage             *TenantUsage        `json:"usage,omitempty"`
	IsWithinLimits    bool                `json:"is_within_limits"`
	TransactionLimit  int                 `json:"transaction_limit"`
	TransactionsUsed  int                 `json:"transactions_used"`
	OutletLimit       int                 `json:"outlet_limit"`
	OutletsUsed       int                 `json:"outlets_used"`
}
