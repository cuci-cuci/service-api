package owner

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type FinanceHandler struct {
	svc      *service.ExpenseService
	validate *validator.Validate
}

func NewFinanceHandler(svc *service.ExpenseService, validate *validator.Validate) *FinanceHandler {
	return &FinanceHandler{svc: svc, validate: validate}
}

// --- Categories ---

func (h *FinanceHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	categories, err := h.svc.ListCategories(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, categories)
}

func (h *FinanceHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var req domain.CreateExpenseCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	category, err := h.svc.CreateCategory(r.Context(), tenantID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, category)
}

// --- Expenses CRUD ---

func (h *FinanceHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	userID := middleware.GetUserID(r.Context())

	var req domain.CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	expense, err := h.svc.CreateExpense(r.Context(), tenantID, userID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, expense)
}

func (h *FinanceHandler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	q := r.URL.Query()
	startDate := q.Get("start_date")
	endDate := q.Get("end_date")

	var categoryID, outletID *uuid.UUID
	if s := q.Get("category_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			categoryID = &id
		}
	}
	if s := q.Get("outlet_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			outletID = &id
		}
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	expenses, total, err := h.svc.ListExpenses(r.Context(), tenantID, startDate, endDate, categoryID, outletID, page, perPage)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"data":  expenses,
		"total": total,
		"page":  page,
	})
}

func (h *FinanceHandler) GetExpense(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	expenseID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid expense ID"))
		return
	}

	expense, err := h.svc.GetExpense(r.Context(), tenantID, expenseID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, expense)
}

func (h *FinanceHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	expenseID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid expense ID"))
		return
	}

	var req domain.UpdateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	expense, err := h.svc.UpdateExpense(r.Context(), tenantID, expenseID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, expense)
}

func (h *FinanceHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	expenseID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid expense ID"))
		return
	}

	if err := h.svc.DeleteExpense(r.Context(), tenantID, expenseID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "expense deleted"})
}

// --- Recurring Expenses ---

func (h *FinanceHandler) CreateRecurringExpense(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var req domain.CreateRecurringExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	re, err := h.svc.CreateRecurringExpense(r.Context(), tenantID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, re)
}

func (h *FinanceHandler) ListRecurringExpenses(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	expenses, err := h.svc.ListRecurringExpenses(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, expenses)
}

// --- Reports ---

func (h *FinanceHandler) GetPnLReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		response.Error(w, apperror.Validation("start_date and end_date are required"))
		return
	}

	report, err := h.svc.GetPnLReport(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, report)
}

func (h *FinanceHandler) GetCashFlowReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		response.Error(w, apperror.Validation("start_date and end_date are required"))
		return
	}

	report, err := h.svc.GetCashFlowReport(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, report)
}

func (h *FinanceHandler) GetTaxReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")
	if monthStr == "" || yearStr == "" {
		response.Error(w, apperror.Validation("month and year are required"))
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		response.Error(w, apperror.Validation("invalid month"))
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2020 {
		response.Error(w, apperror.Validation("invalid year"))
		return
	}

	report, err := h.svc.GetTaxReport(r.Context(), tenantID, month, year)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": report})
}
