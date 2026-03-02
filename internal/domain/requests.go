package domain

import "github.com/google/uuid"

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type CreateTenantRequest struct {
	Name string `json:"name" validate:"required"`
	Slug string `json:"slug" validate:"required,alphanum,min=2,max=50"`
}

type UpdateTenantRequest struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}

type CreateOutletRequest struct {
	Name    string `json:"name" validate:"required"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type UpdateOutletRequest struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Phone    string `json:"phone"`
	IsActive *bool  `json:"is_active"`
}

type CreateServiceCategoryRequest struct {
	Name      string `json:"name" validate:"required"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

type UpdateServiceCategoryRequest struct {
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder *int   `json:"sort_order"`
	IsActive  *bool  `json:"is_active"`
}

type CreateServiceTemplateRequest struct {
	CategoryID             string `json:"category_id" validate:"required,uuid"`
	Name                   string `json:"name" validate:"required"`
	PricingUnit            string `json:"pricing_unit" validate:"required,oneof=kg pcs meter pair sqm"`
	BasePrice              int64  `json:"base_price" validate:"required,gt=0"`
	EstimatedDurationHours int    `json:"estimated_duration_hours"`
	SortOrder              int    `json:"sort_order"`
}

type UpdateServiceTemplateRequest struct {
	Name                   string `json:"name"`
	PricingUnit            string `json:"pricing_unit" validate:"omitempty,oneof=kg pcs meter pair sqm"`
	BasePrice              *int64 `json:"base_price" validate:"omitempty,gt=0"`
	EstimatedDurationHours *int   `json:"estimated_duration_hours"`
	SortOrder              *int   `json:"sort_order"`
	IsActive               *bool  `json:"is_active"`
}

type SetTenantServicePriceRequest struct {
	ServiceTemplateID string `json:"service_template_id" validate:"required,uuid"`
	Price             int64  `json:"price" validate:"required,gt=0"`
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
	Role     string `json:"role" validate:"required,oneof=tenant_owner cashier"`
	TenantID string `json:"tenant_id" validate:"required,uuid"`
}

type PushConfigRequest struct {
	TenantIDs []uuid.UUID `json:"tenant_ids"`
	Broadcast bool        `json:"broadcast"`
}

type ToggleFeatureFlagRequest struct {
	Enabled bool `json:"enabled"`
}

type CreateFeatureFlagRequest struct {
	Key            string `json:"key" validate:"required"`
	Description    string `json:"description"`
	DefaultEnabled bool   `json:"default_enabled"`
}

type UpdateFeatureFlagRequest struct {
	Description    *string `json:"description"`
	DefaultEnabled *bool   `json:"default_enabled"`
}

type SyncUploadRequest struct {
	Transactions []Transaction `json:"transactions" validate:"required,dive"`
}

type SyncDownloadRequest struct {
	CurrentConfigVersion int `json:"current_config_version"`
}

type CreateMemberRequest struct {
	Name            string `json:"name" validate:"required"`
	Phone           string `json:"phone" validate:"required"`
	Email           string `json:"email" validate:"omitempty,email"`
	Tier            string `json:"tier" validate:"omitempty,oneof=bronze silver gold platinum"`
	DiscountPercent int    `json:"discount_percent" validate:"omitempty,gte=0,lte=100"`
	ReferralCode    string `json:"referral_code" validate:"omitempty,max=8"`
}

type UpdateMemberRequest struct {
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	Email           string `json:"email" validate:"omitempty,email"`
	Tier            string `json:"tier" validate:"omitempty,oneof=bronze silver gold platinum"`
	DiscountPercent *int   `json:"discount_percent" validate:"omitempty,gte=0,lte=100"`
}

type MemberLookupRequest struct {
	Phone string `json:"phone" validate:"required"`
}

type CreatePaymentMethodRequest struct {
	Name      string `json:"name" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=cash qris bank_transfer ewallet other"`
	SortOrder int    `json:"sort_order"`
}

type UpdatePaymentMethodRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type" validate:"omitempty,oneof=cash qris bank_transfer ewallet other"`
	SortOrder *int   `json:"sort_order"`
	IsActive  *bool  `json:"is_active"`
}

type CreateCashierRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
	OutletID string `json:"outlet_id" validate:"omitempty,uuid"`
}

type UpdateCashierRequest struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
	OutletID string `json:"outlet_id" validate:"omitempty,uuid"`
}

type BulkSetTenantServicePriceRequest struct {
	Prices []SetTenantServicePriceRequest `json:"prices" validate:"required,dive"`
}

type RegisterRequest struct {
	BusinessName string `json:"business_name" validate:"required,min=2,max=100"`
	Slug         string `json:"slug" validate:"required,alphanum,min=2,max=50"`
	Email        string `json:"email" validate:"required,email"`
	Password     string `json:"password" validate:"required,min=8"`
	Phone        string `json:"phone" validate:"omitempty"`
	OwnerName    string `json:"owner_name" validate:"required,min=2"`
}

type CreateOrderRequest struct {
	TransactionID          *string `json:"transaction_id"`
	OutletID               string  `json:"outlet_id" validate:"required,uuid"`
	EstimatedDurationHours int     `json:"estimated_duration_hours"`
	Notes                  string  `json:"notes"`
	CustomerPhone          string  `json:"customer_phone"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=received washing drying ironing done picked_up cancelled"`
	Notes  string `json:"notes"`
}

type OpenShiftRequest struct {
	OutletID    string `json:"outlet_id" validate:"required,uuid"`
	OpeningCash int64  `json:"opening_cash" validate:"gte=0"`
}

type CloseShiftRequest struct {
	ClosingCash int64  `json:"closing_cash" validate:"gte=0"`
	Notes       string `json:"notes"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type UpsertGatewayConfigRequest struct {
	SecretKey    string   `json:"secret_key" validate:"required"`
	PublicKey    string   `json:"public_key"`
	WebhookToken string   `json:"webhook_token"`
	EnabledTypes []string `json:"enabled_types" validate:"omitempty,dive,oneof=qris virtual_account ewallet"`
	IsEnabled    bool     `json:"is_enabled"`
}

type UpdateGatewayEnabledRequest struct {
	IsEnabled bool `json:"is_enabled"`
}

type CreateGatewayPaymentRequest struct {
	TransactionID string `json:"transaction_id" validate:"required,uuid"`
	PaymentItemID string `json:"payment_item_id" validate:"required,uuid"`
	GatewayType   string `json:"gateway_type" validate:"required,oneof=qris virtual_account ewallet"`
	Amount        int64  `json:"amount" validate:"required,gte=1000,lte=100000000"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
}
