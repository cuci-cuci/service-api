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

type RevenueByServiceResponse struct {
	ServiceName string `json:"service_name"`
	Revenue     int64  `json:"revenue"`
	Quantity    int    `json:"quantity"`
}

type RevenueByPaymentMethodResponse struct {
	MethodName string `json:"method_name"`
	MethodType string `json:"method_type"`
	Revenue    int64  `json:"revenue"`
	TxCount    int    `json:"transaction_count"`
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

type ShiftSummaryResponse struct {
	Shift
	CashierName      string                  `json:"cashier_name"`
	TransactionCount int                     `json:"transaction_count"`
	TotalRevenue     int64                   `json:"total_revenue"`
	PaymentBreakdown []ShiftPaymentBreakdown `json:"payment_breakdown"`
}

type ShiftPaymentBreakdown struct {
	PaymentType string `json:"payment_type"`
	Count       int    `json:"count"`
	Amount      int64  `json:"amount"`
}

type OrderTrackingResponse struct {
	ID                    uuid.UUID             `json:"id"`
	Status                string                `json:"status"`
	EstimatedCompletionAt *time.Time            `json:"estimated_completion_at,omitempty"`
	CompletedAt           *time.Time            `json:"completed_at,omitempty"`
	PickedUpAt            *time.Time            `json:"picked_up_at,omitempty"`
	TrackingToken         string                `json:"tracking_token"`
	CreatedAt             time.Time             `json:"created_at"`
	CustomerName          *string               `json:"customer_name,omitempty"`
	OrderNumber           string                `json:"order_number"`
	BusinessName          string                `json:"business_name"`
	StatusHistory         []TrackingStatusEntry  `json:"status_history"`
}

type TrackingStatusEntry struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
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
