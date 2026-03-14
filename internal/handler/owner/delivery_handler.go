package owner

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
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

func (h *DeliveryHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Optional outlet_id filter
	var outletID *uuid.UUID
	if oid := r.URL.Query().Get("outlet_id"); oid != "" {
		parsed, parseErr := uuid.Parse(oid)
		if parseErr != nil {
			response.Error(w, apperror.Validation("invalid outlet_id"))
			return
		}
		outletID = &parsed
	}

	zones, err := h.svc.ListAllZones(r.Context(), tenantID, outletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, zones)
}

func (h *DeliveryHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req domain.CreateDeliveryZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	zone, err := h.svc.CreateZone(r.Context(), tenantID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, zone)
}

func (h *DeliveryHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	zoneID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid delivery zone ID"))
		return
	}

	var req domain.UpdateDeliveryZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	zone, err := h.svc.UpdateZone(r.Context(), tenantID, zoneID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, zone)
}

func (h *DeliveryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	zoneID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid delivery zone ID"))
		return
	}

	if err := h.svc.DeleteZone(r.Context(), tenantID, zoneID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Pickup Requests ---

func (h *DeliveryHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var outletID *uuid.UUID
	if oid := r.URL.Query().Get("outlet_id"); oid != "" {
		parsed, parseErr := uuid.Parse(oid)
		if parseErr == nil {
			outletID = &parsed
		}
	}
	status := r.URL.Query().Get("status")

	requests, svcErr := h.svc.ListPickupRequests(r.Context(), tenantID, outletID, status)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, requests)
}

func (h *DeliveryHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var req domain.CreatePickupRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	pickup, svcErr := h.svc.CreatePickupRequest(r.Context(), tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, pickup)
}

func (h *DeliveryHandler) UpdateRequestStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	requestID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid request ID"))
		return
	}
	var req domain.UpdatePickupRequestStatus
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	if svcErr := h.svc.UpdatePickupRequestStatus(r.Context(), tenantID, requestID, req); svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
