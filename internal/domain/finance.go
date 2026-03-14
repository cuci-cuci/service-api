package domain

import (
	"time"

	"github.com/google/uuid"
)

// --- Models ---

type ExpenseCategory struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Expense struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	OutletID    *uuid.UUID `json:"outlet_id,omitempty"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	Amount      int64      `json:"amount"`
	Description *string    `json:"description,omitempty"`
	ExpenseDate string     `json:"expense_date"` // YYYY-MM-DD
	ReceiptURL  *string    `json:"receipt_url,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Joined fields
	CategoryName *string `json:"category_name,omitempty"`
	OutletName   *string `json:"outlet_name,omitempty"`
}

type RecurringExpense struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	OutletID    *uuid.UUID `json:"outlet_id,omitempty"`
	Amount      int64      `json:"amount"`
	Description *string    `json:"description,omitempty"`
	Frequency   string     `json:"frequency"`
	NextDueDate string     `json:"next_due_date"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// --- Requests ---

type CreateExpenseRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	OutletID    *uuid.UUID `json:"outlet_id"`
	Amount      int64      `json:"amount" validate:"required,gt=0"`
	Description string     `json:"description"`
	ExpenseDate string     `json:"expense_date" validate:"required"`
}

type UpdateExpenseRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	OutletID    *uuid.UUID `json:"outlet_id"`
	Amount      *int64     `json:"amount" validate:"omitempty,gt=0"`
	Description *string    `json:"description"`
	ExpenseDate *string    `json:"expense_date"`
}

type CreateExpenseCategoryRequest struct {
	Name string `json:"name" validate:"required,max=100"`
	Icon string `json:"icon"`
}

type CreateRecurringExpenseRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	OutletID    *uuid.UUID `json:"outlet_id"`
	Amount      int64      `json:"amount" validate:"required,gt=0"`
	Description string     `json:"description"`
	Frequency   string     `json:"frequency" validate:"required,oneof=daily weekly monthly yearly"`
	NextDueDate string     `json:"next_due_date" validate:"required"`
}

// --- Responses ---

type PnLReport struct {
	Period            string                   `json:"period"`
	StartDate         string                   `json:"start_date"`
	EndDate           string                   `json:"end_date"`
	Revenue           int64                    `json:"revenue"`
	Expenses          int64                    `json:"expenses"`
	GrossProfit       int64                    `json:"gross_profit"`
	MarginPercent     float64                  `json:"margin_percent"`
	ExpenseByCategory []ExpenseByCategoryItem  `json:"expense_by_category"`
}

type ExpenseByCategoryItem struct {
	CategoryName string `json:"category_name"`
	Amount       int64  `json:"amount"`
}

type CashFlowReport struct {
	Period  string `json:"period"`
	CashIn  int64  `json:"cash_in"`
	CashOut int64  `json:"cash_out"`
	NetFlow int64  `json:"net_flow"`
}
