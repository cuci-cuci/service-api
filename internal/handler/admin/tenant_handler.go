package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type TenantHandler struct {
	svc      *service.TenantService
	audit    *service.AuditService
	validate *validator.Validate
}

func NewTenantHandler(svc *service.TenantService, audit *service.AuditService, validate *validator.Validate) *TenantHandler {
	return &TenantHandler{svc: svc, audit: audit, validate: validate}
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	tenants, total, err := h.svc.List(r.Context(), params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, tenants, params.ToMeta(total))
}

func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	tenant, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	tenant, err := h.svc.Create(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	// FIX 5: Audit log for tenant creation
	userID := middleware.GetUserID(r.Context())
	claims := middleware.GetClaims(r.Context())
	actorName := ""
	if claims != nil {
		actorName = claims.Subject
	}
	h.audit.LogAction(r.Context(), userID, actorName, "create", "tenant", tenant.ID.String(), nil, tenant, nil)

	response.JSON(w, http.StatusCreated, tenant)
}

func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	var req domain.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	// Get old value for audit log
	oldTenant, _ := h.svc.GetByID(r.Context(), id)

	tenant, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	// FIX 5: Audit log for tenant update
	userID := middleware.GetUserID(r.Context())
	claims := middleware.GetClaims(r.Context())
	actorName := ""
	if claims != nil {
		actorName = claims.Subject
	}
	h.audit.LogAction(r.Context(), userID, actorName, "update", "tenant", tenant.ID.String(), oldTenant, tenant, nil)

	response.JSON(w, http.StatusOK, tenant)
}

func validationErrors(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			errs[fe.Field()] = fe.Tag()
		}
	}
	return errs
}
