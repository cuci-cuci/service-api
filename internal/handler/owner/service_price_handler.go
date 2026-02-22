package owner

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type ServicePriceHandler struct {
	svc      *service.ServiceTemplateService
	validate *validator.Validate
}

func NewServicePriceHandler(svc *service.ServiceTemplateService, validate *validator.Validate) *ServicePriceHandler {
	return &ServicePriceHandler{svc: svc, validate: validate}
}

func (h *ServicePriceHandler) ListServicesWithPrices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	services, err := h.svc.GetServicesWithPrices(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, services)
}

func (h *ServicePriceHandler) SetPrice(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	templateID := chi.URLParam(r, "templateId")

	var body struct {
		Price int64 `json:"price" validate:"required,gt=0"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(body); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	req := domain.SetTenantServicePriceRequest{
		ServiceTemplateID: templateID,
		Price:             body.Price,
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

func (h *ServicePriceHandler) BulkSetPrices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req domain.BulkSetTenantServicePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	var results []domain.TenantServicePrice
	for _, p := range req.Prices {
		price, err := h.svc.SetTenantPrice(r.Context(), tenantID, p)
		if err != nil {
			response.Error(w, err)
			return
		}
		results = append(results, *price)
	}

	if results == nil {
		results = []domain.TenantServicePrice{}
	}

	response.JSON(w, http.StatusOK, results)
}
