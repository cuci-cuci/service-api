package pos

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type DeliveryHandler struct {
	svc      *service.DeliveryService
	validate *validator.Validate
}

func NewDeliveryHandler(svc *service.DeliveryService, validate *validator.Validate) *DeliveryHandler {
	return &DeliveryHandler{svc: svc, validate: validate}
}

func (h *DeliveryHandler) ListZones(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}
	var outletID *uuid.UUID
	if oid := r.URL.Query().Get("outlet_id"); oid != "" {
		parsed, parseErr := uuid.Parse(oid)
		if parseErr == nil {
			outletID = &parsed
		}
	}
	zones, err := h.svc.ListZones(r.Context(), *tenantID, outletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, zones)
}

func (h *DeliveryHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}
	var req domain.CreatePickupRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(w, apperror.Validation(err.Error()))
		return
	}
	pickup, svcErr := h.svc.CreatePickupRequest(r.Context(), *tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, pickup)
}
