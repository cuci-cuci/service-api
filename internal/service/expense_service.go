package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type ExpenseService struct {
	db *pgxpool.Pool
}

func NewExpenseService(db *pgxpool.Pool) *ExpenseService {
	return &ExpenseService{db: db}
}

// --- Categories ---

func (s *ExpenseService) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]domain.ExpenseCategory, error) {
	q := middleware.GetQuerier(ctx, s.db)
	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, name, icon, is_active, created_at, updated_at
		 FROM expense_categories WHERE tenant_id = $1 AND is_active = true ORDER BY name`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list expense categories", err)
	}
	defer rows.Close()

	var categories []domain.ExpenseCategory
	for rows.Next() {
		var c domain.ExpenseCategory
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Icon, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, apperror.Internal("failed to scan expense category", err)
		}
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []domain.ExpenseCategory{}
	}
	return categories, nil
}

func (s *ExpenseService) CreateCategory(ctx context.Context, tenantID uuid.UUID, req domain.CreateExpenseCategoryRequest) (*domain.ExpenseCategory, error) {
	q := middleware.GetQuerier(ctx, s.db)
	icon := req.Icon
	if icon == "" {
		icon = "receipt"
	}

	var c domain.ExpenseCategory
	err := q.QueryRow(ctx,
		`INSERT INTO expense_categories (tenant_id, name, icon)
		 VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, name, icon, is_active, created_at, updated_at`,
		tenantID, req.Name, icon).
		Scan(&c.ID, &c.TenantID, &c.Name, &c.Icon, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create expense category", err)
	}
	return &c, nil
}

// --- Expenses CRUD ---

func (s *ExpenseService) CreateExpense(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, req domain.CreateExpenseRequest) (*domain.Expense, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var e domain.Expense
	err := q.QueryRow(ctx,
		`INSERT INTO expenses (tenant_id, outlet_id, category_id, amount, description, expense_date, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, tenant_id, outlet_id, category_id, amount, description, expense_date, receipt_url, created_by, created_at, updated_at`,
		tenantID, req.OutletID, req.CategoryID, req.Amount, req.Description, req.ExpenseDate, userID).
		Scan(&e.ID, &e.TenantID, &e.OutletID, &e.CategoryID, &e.Amount, &e.Description, &e.ExpenseDate, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create expense", err)
	}
	return &e, nil
}

func (s *ExpenseService) ListExpenses(ctx context.Context, tenantID uuid.UUID, startDate, endDate string, categoryID, outletID *uuid.UUID, page, perPage int) ([]domain.Expense, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	baseQuery := `FROM expenses e
		LEFT JOIN expense_categories ec ON e.category_id = ec.id
		LEFT JOIN outlets o ON e.outlet_id = o.id
		WHERE e.tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if startDate != "" {
		baseQuery += fmt.Sprintf(" AND e.expense_date >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		baseQuery += fmt.Sprintf(" AND e.expense_date <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}
	if categoryID != nil {
		baseQuery += fmt.Sprintf(" AND e.category_id = $%d", argIdx)
		args = append(args, *categoryID)
		argIdx++
	}
	if outletID != nil {
		baseQuery += fmt.Sprintf(" AND e.outlet_id = $%d", argIdx)
		args = append(args, *outletID)
		argIdx++
	}

	// Count
	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count expenses", err)
	}

	// Paginated query
	offset := (page - 1) * perPage
	selectQuery := fmt.Sprintf(
		`SELECT e.id, e.tenant_id, e.outlet_id, e.category_id, e.amount, e.description, e.expense_date,
		        e.receipt_url, e.created_by, e.created_at, e.updated_at, ec.name, o.name
		 %s ORDER BY e.expense_date DESC, e.created_at DESC LIMIT $%d OFFSET $%d`,
		baseQuery, argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := q.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list expenses", err)
	}
	defer rows.Close()

	var expenses []domain.Expense
	for rows.Next() {
		var e domain.Expense
		if err := rows.Scan(&e.ID, &e.TenantID, &e.OutletID, &e.CategoryID, &e.Amount, &e.Description,
			&e.ExpenseDate, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &e.CategoryName, &e.OutletName); err != nil {
			return nil, 0, apperror.Internal("failed to scan expense", err)
		}
		expenses = append(expenses, e)
	}
	if expenses == nil {
		expenses = []domain.Expense{}
	}
	return expenses, total, nil
}

func (s *ExpenseService) GetExpense(ctx context.Context, tenantID, expenseID uuid.UUID) (*domain.Expense, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var e domain.Expense
	err := q.QueryRow(ctx,
		`SELECT e.id, e.tenant_id, e.outlet_id, e.category_id, e.amount, e.description, e.expense_date,
		        e.receipt_url, e.created_by, e.created_at, e.updated_at, ec.name, o.name
		 FROM expenses e
		 LEFT JOIN expense_categories ec ON e.category_id = ec.id
		 LEFT JOIN outlets o ON e.outlet_id = o.id
		 WHERE e.id = $1 AND e.tenant_id = $2`, expenseID, tenantID).
		Scan(&e.ID, &e.TenantID, &e.OutletID, &e.CategoryID, &e.Amount, &e.Description,
			&e.ExpenseDate, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &e.CategoryName, &e.OutletName)
	if err != nil {
		return nil, apperror.NotFound("expense not found")
	}
	return &e, nil
}

func (s *ExpenseService) UpdateExpense(ctx context.Context, tenantID, expenseID uuid.UUID, req domain.UpdateExpenseRequest) (*domain.Expense, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var e domain.Expense
	err := q.QueryRow(ctx,
		`UPDATE expenses SET
			category_id = COALESCE($3, category_id),
			outlet_id = COALESCE($4, outlet_id),
			amount = COALESCE($5, amount),
			description = COALESCE($6, description),
			expense_date = COALESCE($7, expense_date),
			updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, outlet_id, category_id, amount, description, expense_date, receipt_url, created_by, created_at, updated_at`,
		expenseID, tenantID, req.CategoryID, req.OutletID, req.Amount, req.Description, req.ExpenseDate).
		Scan(&e.ID, &e.TenantID, &e.OutletID, &e.CategoryID, &e.Amount, &e.Description, &e.ExpenseDate, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, apperror.NotFound("expense not found")
	}
	return &e, nil
}

func (s *ExpenseService) DeleteExpense(ctx context.Context, tenantID, expenseID uuid.UUID) error {
	q := middleware.GetQuerier(ctx, s.db)

	ct, err := q.Exec(ctx, `DELETE FROM expenses WHERE id = $1 AND tenant_id = $2`, expenseID, tenantID)
	if err != nil {
		return apperror.Internal("failed to delete expense", err)
	}
	if ct.RowsAffected() == 0 {
		return apperror.NotFound("expense not found")
	}
	return nil
}

// --- Recurring Expenses ---

func (s *ExpenseService) CreateRecurringExpense(ctx context.Context, tenantID uuid.UUID, req domain.CreateRecurringExpenseRequest) (*domain.RecurringExpense, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var re domain.RecurringExpense
	err := q.QueryRow(ctx,
		`INSERT INTO recurring_expenses (tenant_id, category_id, outlet_id, amount, description, frequency, next_due_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, tenant_id, category_id, outlet_id, amount, description, frequency, next_due_date, is_active, created_at, updated_at`,
		tenantID, req.CategoryID, req.OutletID, req.Amount, req.Description, req.Frequency, req.NextDueDate).
		Scan(&re.ID, &re.TenantID, &re.CategoryID, &re.OutletID, &re.Amount, &re.Description, &re.Frequency, &re.NextDueDate, &re.IsActive, &re.CreatedAt, &re.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create recurring expense", err)
	}
	return &re, nil
}

func (s *ExpenseService) ListRecurringExpenses(ctx context.Context, tenantID uuid.UUID) ([]domain.RecurringExpense, error) {
	q := middleware.GetQuerier(ctx, s.db)

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, category_id, outlet_id, amount, description, frequency, next_due_date, is_active, created_at, updated_at
		 FROM recurring_expenses WHERE tenant_id = $1 AND is_active = true ORDER BY next_due_date`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list recurring expenses", err)
	}
	defer rows.Close()

	var expenses []domain.RecurringExpense
	for rows.Next() {
		var re domain.RecurringExpense
		if err := rows.Scan(&re.ID, &re.TenantID, &re.CategoryID, &re.OutletID, &re.Amount, &re.Description, &re.Frequency, &re.NextDueDate, &re.IsActive, &re.CreatedAt, &re.UpdatedAt); err != nil {
			return nil, apperror.Internal("failed to scan recurring expense", err)
		}
		expenses = append(expenses, re)
	}
	if expenses == nil {
		expenses = []domain.RecurringExpense{}
	}
	return expenses, nil
}

// --- P&L and Cash Flow Reports ---

func (s *ExpenseService) GetPnLReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate string) (*domain.PnLReport, error) {
	q := middleware.GetQuerier(ctx, s.db)

	report := &domain.PnLReport{
		StartDate: startDate,
		EndDate:   endDate,
		Period:    startDate + " to " + endDate,
	}

	// Revenue from transactions
	err := q.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_amount), 0)
		 FROM transactions
		 WHERE tenant_id = $1 AND created_at::date BETWEEN $2 AND $3 AND status = 'completed'`,
		tenantID, startDate, endDate).Scan(&report.Revenue)
	if err != nil {
		return nil, apperror.Internal("failed to get revenue", err)
	}

	// Expenses total
	err = q.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)
		 FROM expenses
		 WHERE tenant_id = $1 AND expense_date BETWEEN $2 AND $3`,
		tenantID, startDate, endDate).Scan(&report.Expenses)
	if err != nil {
		return nil, apperror.Internal("failed to get expenses total", err)
	}

	report.GrossProfit = report.Revenue - report.Expenses
	if report.Revenue > 0 {
		report.MarginPercent = float64(report.GrossProfit) / float64(report.Revenue) * 100
	}

	// Expense breakdown by category
	rows, err := q.Query(ctx,
		`SELECT COALESCE(ec.name, 'Tanpa Kategori'), SUM(e.amount)
		 FROM expenses e LEFT JOIN expense_categories ec ON e.category_id = ec.id
		 WHERE e.tenant_id = $1 AND e.expense_date BETWEEN $2 AND $3
		 GROUP BY ec.name ORDER BY SUM(e.amount) DESC`,
		tenantID, startDate, endDate)
	if err != nil {
		return nil, apperror.Internal("failed to get expense breakdown", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.ExpenseByCategoryItem
		if err := rows.Scan(&item.CategoryName, &item.Amount); err != nil {
			return nil, apperror.Internal("failed to scan expense category breakdown", err)
		}
		report.ExpenseByCategory = append(report.ExpenseByCategory, item)
	}
	if report.ExpenseByCategory == nil {
		report.ExpenseByCategory = []domain.ExpenseByCategoryItem{}
	}

	return report, nil
}

func (s *ExpenseService) GetCashFlowReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate string) (*domain.CashFlowReport, error) {
	q := middleware.GetQuerier(ctx, s.db)

	report := &domain.CashFlowReport{
		Period: startDate + " to " + endDate,
	}

	// Cash in: revenue from all completed transactions
	err := q.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_amount), 0)
		 FROM transactions
		 WHERE tenant_id = $1 AND created_at::date BETWEEN $2 AND $3 AND status = 'completed'`,
		tenantID, startDate, endDate).Scan(&report.CashIn)
	if err != nil {
		return nil, apperror.Internal("failed to get cash in", err)
	}

	// Cash out: expenses
	err = q.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)
		 FROM expenses
		 WHERE tenant_id = $1 AND expense_date BETWEEN $2 AND $3`,
		tenantID, startDate, endDate).Scan(&report.CashOut)
	if err != nil {
		return nil, apperror.Internal("failed to get cash out", err)
	}

	report.NetFlow = report.CashIn - report.CashOut
	return report, nil
}

// ProcessDueRecurring processes recurring expenses that are due today.
func (s *ExpenseService) ProcessDueRecurring(ctx context.Context) error {
	q := middleware.GetQuerier(ctx, s.db)
	today := time.Now().Format("2006-01-02")

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, category_id, outlet_id, amount, description, frequency
		 FROM recurring_expenses WHERE is_active = true AND next_due_date <= $1`, today)
	if err != nil {
		return apperror.Internal("failed to get due recurring expenses", err)
	}
	defer rows.Close()

	type dueItem struct {
		ID          uuid.UUID
		TenantID    uuid.UUID
		CategoryID  *uuid.UUID
		OutletID    *uuid.UUID
		Amount      int64
		Description *string
		Frequency   string
	}

	var items []dueItem
	for rows.Next() {
		var d dueItem
		if err := rows.Scan(&d.ID, &d.TenantID, &d.CategoryID, &d.OutletID, &d.Amount, &d.Description, &d.Frequency); err != nil {
			continue
		}
		items = append(items, d)
	}

	for _, d := range items {
		// Create expense entry
		_, err := q.Exec(ctx,
			`INSERT INTO expenses (tenant_id, outlet_id, category_id, amount, description, expense_date)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			d.TenantID, d.OutletID, d.CategoryID, d.Amount, d.Description, today)
		if err != nil {
			continue
		}

		// Advance next_due_date
		var nextDate string
		switch d.Frequency {
		case "daily":
			nextDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		case "weekly":
			nextDate = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		case "monthly":
			nextDate = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
		case "yearly":
			nextDate = time.Now().AddDate(1, 0, 0).Format("2006-01-02")
		}
		_, _ = q.Exec(ctx, `UPDATE recurring_expenses SET next_due_date = $1 WHERE id = $2`, nextDate, d.ID)
	}

	return nil
}

// GetTaxReport generates a monthly tax report (PPN 11%).
func (s *ExpenseService) GetTaxReport(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.TaxReport, error) {
	q := middleware.GetQuerier(ctx, s.db)

	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	// Last day of month
	endDate := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	report := &domain.TaxReport{Month: month, Year: year}

	// Summary totals
	err := q.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(subtotal), 0), COALESCE(SUM(discount_amount), 0), COALESCE(SUM(total_amount), 0)
		FROM transactions
		WHERE tenant_id = $1 AND created_at::date BETWEEN $2 AND $3 AND status = 'completed'
	`, tenantID, startDate, endDate).Scan(&report.TransactionCount, &report.GrossSales, &report.TotalDiscount, &report.NetSales)
	if err != nil {
		return nil, apperror.Internal("failed to get tax summary", err)
	}

	report.TaxableBase = report.NetSales
	report.PPNAmount = int64(float64(report.TaxableBase) * 0.11)

	// Per-outlet breakdown
	rows, err := q.Query(ctx, `
		SELECT t.outlet_id, COALESCE(o.name, 'Unknown'), COUNT(*),
		       COALESCE(SUM(t.subtotal), 0), COALESCE(SUM(t.discount_amount), 0), COALESCE(SUM(t.total_amount), 0)
		FROM transactions t
		LEFT JOIN outlets o ON t.outlet_id = o.id
		WHERE t.tenant_id = $1 AND t.created_at::date BETWEEN $2 AND $3 AND t.status = 'completed'
		GROUP BY t.outlet_id, o.name
		ORDER BY SUM(t.total_amount) DESC
	`, tenantID, startDate, endDate)
	if err != nil {
		return nil, apperror.Internal("failed to get outlet tax breakdown", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.TaxByOutlet
		if err := rows.Scan(&item.OutletID, &item.OutletName, &item.TxCount,
			&item.GrossSales, &item.Discount, &item.NetSales); err != nil {
			return nil, apperror.Internal("failed to scan outlet breakdown", err)
		}
		item.PPNAmount = int64(float64(item.NetSales) * 0.11)
		report.OutletBreakdown = append(report.OutletBreakdown, item)
	}
	if report.OutletBreakdown == nil {
		report.OutletBreakdown = []domain.TaxByOutlet{}
	}

	return report, nil
}
