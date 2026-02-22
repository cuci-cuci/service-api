package owner

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type CashierHandler struct {
	svc      *service.UserService
	validate *validator.Validate
}

func NewCashierHandler(svc *service.UserService, validate *validator.Validate) *CashierHandler {
	return &CashierHandler{svc: svc, validate: validate}
}

func (h *CashierHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	params := pagination.ParseFromRequest(r)
	users, total, err := h.svc.ListByTenantAndRole(r.Context(), tenantID, domain.RoleCashier, params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, users, params.ToMeta(total))
}

func (h *CashierHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req domain.CreateCashierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	// Convert to CreateUserRequest
	createReq := domain.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     domain.RoleCashier,
		TenantID: tenantID.String(),
	}

	user, err := h.svc.Create(r.Context(), createReq)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, user)
}

func (h *CashierHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid cashier ID"))
		return
	}

	var req domain.UpdateCashierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	user, err := h.svc.UpdateUser(r.Context(), id, req.Name, req.IsActive, req.OutletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}
