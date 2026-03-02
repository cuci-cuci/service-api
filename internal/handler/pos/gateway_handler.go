package pos

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/pkg/validation"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type GatewayHandler struct {
	svc      *service.GatewayService
	validate *validator.Validate
}

func NewGatewayHandler(svc *service.GatewayService, validate *validator.Validate) *GatewayHandler {
	return &GatewayHandler{svc: svc, validate: validate}
}

func (h *GatewayHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	var req domain.CreateGatewayPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validation.Errors(err))
		return
	}

	result, svcErr := h.svc.CreateGatewayPayment(r.Context(), *tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func (h *GatewayHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	externalID := chi.URLParam(r, "externalID")
	if externalID == "" {
		response.Error(w, apperror.Validation("externalID is required"))
		return
	}

	status, svcErr := h.svc.GetPaymentStatus(r.Context(), *tenantID, externalID)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, status)
}
