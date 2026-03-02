package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Outlet struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Phone     string    `json:"phone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ServiceCategory struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type ServiceTemplate struct {
	ID                     uuid.UUID `json:"id"`
	CategoryID             uuid.UUID `json:"category_id"`
	Name                   string    `json:"name"`
	PricingUnit            string    `json:"pricing_unit"`
	BasePrice              int64     `json:"base_price"`
	EstimatedDurationHours int       `json:"estimated_duration_hours"`
	IsActive               bool      `json:"is_active"`
	SortOrder              int       `json:"sort_order"`
	CreatedAt              time.Time `json:"created_at"`
}

type TenantServicePrice struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	ServiceTemplateID uuid.UUID `json:"service_template_id"`
	Price             int64     `json:"price"`
	IsActive          bool      `json:"is_active"`
}

type ConfigVersion struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Version   int             `json:"version"`
	Data      json.RawMessage `json:"data"`
	CreatedBy uuid.UUID       `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
}

type FeatureFlag struct {
	ID             uuid.UUID `json:"id"`
	Key            string    `json:"key"`
	Description    string    `json:"description"`
	DefaultEnabled bool      `json:"default_enabled"`
}

type TenantFeatureFlag struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	FeatureFlagID uuid.UUID `json:"feature_flag_id"`
	Enabled       bool      `json:"enabled"`
}

type Transaction struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	OutletID         uuid.UUID       `json:"outlet_id"`
	LocalOrderNumber string          `json:"local_order_number"`
	CustomerName     *string         `json:"customer_name,omitempty"`
	MemberID         *uuid.UUID      `json:"member_id,omitempty"`
	Items            json.RawMessage `json:"items"`
	Subtotal         int64           `json:"subtotal"`
	DiscountAmount   int64           `json:"discount_amount"`
	TaxAmount        int64           `json:"tax_amount"`
	TotalAmount      int64           `json:"total_amount"`
	PaymentStatus    string          `json:"payment_status"`
	Payments         json.RawMessage `json:"payments"`
	Status           string          `json:"status"`
	ConfigVersionID  uuid.UUID       `json:"config_version_id"`
	Notes            *string         `json:"notes,omitempty"`
	CreatedBy        uuid.UUID       `json:"created_by"`
	ShiftID                *uuid.UUID      `json:"shift_id,omitempty"`
	CustomerPhone          *string         `json:"customer_phone,omitempty"`
	EstimatedDurationHours *int            `json:"estimated_duration_hours,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
	SyncedAt               *time.Time      `json:"synced_at,omitempty"`
}

type Member struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            *uuid.UUID `json:"tenant_id,omitempty"`
	Name                string     `json:"name"`
	Phone               string     `json:"phone"`
	Email               string     `json:"email"`
	Tier                string     `json:"tier"`
	DiscountPercent     int        `json:"discount_percent"`
	TotalPoints         int        `json:"total_points"`
	TotalSpending       int64      `json:"total_spending"`
	ReferralCode        *string    `json:"referral_code,omitempty"`
	ReferredByMemberID  *uuid.UUID `json:"referred_by_member_id,omitempty"`
	HasFirstTransaction bool       `json:"has_first_transaction"`
	CreatedAt           time.Time  `json:"created_at"`
}

type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	ActorID    uuid.UUID       `json:"actor_id"`
	ActorName  string          `json:"actor_name"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	OldValue   json.RawMessage `json:"old_value,omitempty"`
	NewValue   json.RawMessage `json:"new_value,omitempty"`
	TenantID   *uuid.UUID      `json:"tenant_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type SyncSession struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	OutletID         uuid.UUID  `json:"outlet_id"`
	Direction        string     `json:"direction"`
	Status           string     `json:"status"`
	TransactionCount int        `json:"transaction_count"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type PaymentMethod struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	IsActive  bool      `json:"is_active"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type Order struct {
	ID                    uuid.UUID  `json:"id"`
	TransactionID         *uuid.UUID `json:"transaction_id,omitempty"`
	TenantID              uuid.UUID  `json:"tenant_id"`
	OutletID              uuid.UUID  `json:"outlet_id"`
	Status                string     `json:"status"`
	EstimatedCompletionAt *time.Time `json:"estimated_completion_at,omitempty"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	PickedUpAt            *time.Time `json:"picked_up_at,omitempty"`
	Notes                 *string    `json:"notes,omitempty"`
	CustomerPhone         *string    `json:"customer_phone,omitempty"`
	TrackingToken         *string    `json:"tracking_token,omitempty"`
	CreatedBy             uuid.UUID  `json:"created_by"`
	UpdatedBy             uuid.UUID  `json:"updated_by"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type OrderStatusLog struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	FromStatus *string   `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	ChangedBy  uuid.UUID `json:"changed_by"`
	Notes      *string   `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type PaymentGatewayConfig struct {
	ID                    uuid.UUID `json:"id"`
	TenantID              uuid.UUID `json:"tenant_id"`
	Gateway               string    `json:"gateway"`
	IsEnabled             bool      `json:"is_enabled"`
	SecretKeyEncrypted    string    `json:"-"`
	PublicKey             *string   `json:"public_key,omitempty"`
	WebhookTokenEncrypted string    `json:"-"`
	EnabledTypes          []string  `json:"enabled_types"`
	KeyVersion            int       `json:"key_version"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type TransactionGatewayPayment struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	TransactionID     uuid.UUID       `json:"transaction_id"`
	PaymentItemID     *uuid.UUID      `json:"payment_item_id,omitempty"`
	Gateway           string          `json:"gateway"`
	GatewayType       string          `json:"gateway_type"`
	ExternalID        string          `json:"external_id"`
	GatewayRefID      *string         `json:"gateway_ref_id,omitempty"`
	Amount            int64           `json:"amount"`
	GatewayStatus     string          `json:"gateway_status"`
	GatewayPaymentURL *string         `json:"gateway_payment_url,omitempty"`
	GatewayResponse   json.RawMessage `json:"-"`
	WebhookPayload    json.RawMessage `json:"-"`
	ExpiresAt         *time.Time      `json:"expires_at,omitempty"`
	PaidAt            *time.Time      `json:"paid_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Shift struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	OutletID       uuid.UUID  `json:"outlet_id"`
	CashierID      uuid.UUID  `json:"cashier_id"`
	OpeningCash    int64      `json:"opening_cash"`
	ClosingCash    *int64     `json:"closing_cash,omitempty"`
	ExpectedCash   *int64     `json:"expected_cash,omitempty"`
	CashDifference *int64     `json:"cash_difference,omitempty"`
	Status         string     `json:"status"`
	OpenedAt       time.Time  `json:"opened_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
