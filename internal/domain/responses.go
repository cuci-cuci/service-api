package domain

import (
	"time"

	"github.com/google/uuid"
)

type APIResponse[T any] struct {
	Data T               `json:"data"`
	Meta *PaginationMeta `json:"meta,omitempty"`
}

type APIError struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SyncDownloadResponse struct {
	ConfigVersion  int                `json:"config_version"`
	Config         *ConfigVersion     `json:"config,omitempty"`
	Services       []ServiceWithPrice `json:"services"`
	Categories     []ServiceCategory  `json:"categories"`
	Members        []Member           `json:"members"`
	Outlets        []Outlet           `json:"outlets"`
	PaymentMethods []PaymentMethod    `json:"payment_methods"`
}

type ServiceWithPrice struct {
	ServiceTemplate
	TenantPrice *int64 `json:"tenant_price,omitempty"`
}

type RevenueByTenantResponse struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	Revenue    int64     `json:"revenue"`
	TxCount    int       `json:"transaction_count"`
}

type RevenueByOutletResponse struct {
	OutletID   uuid.UUID `json:"outlet_id"`
	OutletName string    `json:"outlet_name"`
	Revenue    int64     `json:"revenue"`
	TxCount    int       `json:"transaction_count"`
}

type TransactionStatsResponse struct {
	TotalTransactions int   `json:"total_transactions"`
	TotalRevenue      int64 `json:"total_revenue"`
	AvgTransaction    int64 `json:"avg_transaction"`
}

type SyncUploadResult struct {
	Received int `json:"received"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

type SyncHealthResponse struct {
	TenantID         uuid.UUID    `json:"tenant_id"`
	LastSync         *time.Time   `json:"last_sync,omitempty"`
	LastSyncStatus   string       `json:"last_sync_status"`
	TotalSessions    int          `json:"total_sessions"`
	RecentSessions   []SyncSession `json:"recent_sessions"`
}

type OrderDetailResponse struct {
	Order
	Transaction  *Transaction     `json:"transaction,omitempty"`
	StatusLogs   []OrderStatusLog `json:"status_logs"`
	CustomerName *string          `json:"customer_name,omitempty"`
	TotalAmount  int64            `json:"total_amount"`
	OrderNumber  string           `json:"order_number"`
}

func ToUserResponse(u User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		TenantID:  u.TenantID,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
