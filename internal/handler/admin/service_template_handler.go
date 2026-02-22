package admin

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

type ServiceTemplateHandler struct {
	svc      *service.ServiceTemplateService
	validate *validator.Validate
}

func NewServiceTemplateHandler(svc *service.ServiceTemplateService, validate *validator.Validate) *ServiceTemplateHandler {
	return &ServiceTemplateHandler{svc: svc, validate: validate}
}

// --- Categories ---

func (h *ServiceTemplateHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	categories, total, err := h.svc.ListCategories(r.Context(), params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, categories, params.ToMeta(total))
}

func (h *ServiceTemplateHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateServiceCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	cat, err := h.svc.CreateCategory(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, cat)
}

func (h *ServiceTemplateHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid category ID"))
		return
	}

	var req domain.UpdateServiceCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	// FIX 4: Add validation for update request
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	cat, err := h.svc.UpdateCategory(r.Context(), id, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cat)
}

func (h *ServiceTemplateHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid category ID"))
		return
	}

	if err := h.svc.DeleteCategory(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Templates ---

func (h *ServiceTemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	templates, total, err := h.svc.ListTemplates(r.Context(), params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, templates, params.ToMeta(total))
}

func (h *ServiceTemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateServiceTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	tmpl, err := h.svc.CreateTemplate(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, tmpl)
}

func (h *ServiceTemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid template ID"))
		return
	}

	var req domain.UpdateServiceTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	// FIX 4: Add validation for update request
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	tmpl, err := h.svc.UpdateTemplate(r.Context(), id, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tmpl)
}

func (h *ServiceTemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid template ID"))
		return
	}

	if err := h.svc.DeleteTemplate(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Tenant Service Prices ---

func (h *ServiceTemplateHandler) SetTenantPrice(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	var req domain.SetTenantServicePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	price, err := h.svc.SetTenantPrice(r.Context(), tenantID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, price)
}
