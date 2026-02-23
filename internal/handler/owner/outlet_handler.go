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

type OutletHandler struct {
	svc        *service.OutletService
	billingSvc *service.BillingService
	validate   *validator.Validate
}

func NewOutletHandler(svc *service.OutletService, billingSvc *service.BillingService, validate *validator.Validate) *OutletHandler {
	return &OutletHandler{svc: svc, billingSvc: billingSvc, validate: validate}
}

func (h *OutletHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	params := pagination.ParseFromRequest(r)
	outlets, total, err := h.svc.ListByTenant(r.Context(), tenantID, params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, outlets, params.ToMeta(total))
}

func (h *OutletHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Check outlet limit before creating
	if h.billingSvc != nil {
		if limitErr := h.billingSvc.CheckOutletLimit(r.Context(), tenantID); limitErr != nil {
			response.Error(w, limitErr)
			return
		}
	}

	var req domain.CreateOutletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	outlet, err := h.svc.Create(r.Context(), tenantID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, outlet)
}

func (h *OutletHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid outlet ID"))
		return
	}

	var req domain.UpdateOutletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	outlet, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, outlet)
}
