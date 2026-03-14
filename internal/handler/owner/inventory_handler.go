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

type InventoryHandler struct {
	svc      *service.InventoryService
	validate *validator.Validate
}

func NewInventoryHandler(svc *service.InventoryService, v *validator.Validate) *InventoryHandler {
	return &InventoryHandler{svc: svc, validate: v}
}

func (h *InventoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	cats, err := h.svc.ListCategories(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cats)
}

func (h *InventoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var req domain.CreateSupplyCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	cat, svcErr := h.svc.CreateCategory(r.Context(), tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, cat)
}

func (h *InventoryHandler) ListSupplies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var categoryID *uuid.UUID
	if cid := r.URL.Query().Get("category_id"); cid != "" {
		parsed, parseErr := uuid.Parse(cid)
		if parseErr == nil {
			categoryID = &parsed
		}
	}
	lowStock := r.URL.Query().Get("low_stock") == "true"

	supplies, err := h.svc.ListSupplies(r.Context(), tenantID, categoryID, lowStock)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, supplies)
}

func (h *InventoryHandler) CreateSupply(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var req domain.CreateSupplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	supply, svcErr := h.svc.CreateSupply(r.Context(), tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, supply)
}

func (h *InventoryHandler) UpdateSupply(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	supplyID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid supply id"))
		return
	}
	var req domain.UpdateSupplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	if svcErr := h.svc.UpdateSupply(r.Context(), tenantID, supplyID, req); svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *InventoryHandler) RecordMovement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	userID := middleware.GetUserID(r.Context())

	var req domain.StockMovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	if svcErr := h.svc.RecordMovement(r.Context(), tenantID, userID, req); svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *InventoryHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var supplyID *uuid.UUID
	if sid := r.URL.Query().Get("supply_id"); sid != "" {
		parsed, parseErr := uuid.Parse(sid)
		if parseErr == nil {
			supplyID = &parsed
		}
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	movements, err := h.svc.ListMovements(r.Context(), tenantID, supplyID, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, movements)
}

func (h *InventoryHandler) GetLowStockAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	alerts, err := h.svc.GetLowStockAlerts(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, alerts)
}

// --- Service-Supply Mappings ---

func (h *InventoryHandler) ListMappings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var serviceTemplateID *uuid.UUID
	if sid := r.URL.Query().Get("service_template_id"); sid != "" {
		parsed, parseErr := uuid.Parse(sid)
		if parseErr == nil {
			serviceTemplateID = &parsed
		}
	}
	mappings, err := h.svc.ListMappings(r.Context(), tenantID, serviceTemplateID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, mappings)
}

func (h *InventoryHandler) CreateMapping(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var req domain.CreateServiceSupplyMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	mapping, svcErr := h.svc.CreateMapping(r.Context(), tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, mapping)
}

func (h *InventoryHandler) UpdateMapping(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	mappingID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid mapping id"))
		return
	}
	var req domain.UpdateServiceSupplyMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}
	if svcErr := h.svc.UpdateMapping(r.Context(), tenantID, mappingID, req); svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *InventoryHandler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	mappingID, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		response.Error(w, apperror.Validation("invalid mapping id"))
		return
	}
	if svcErr := h.svc.DeleteMapping(r.Context(), tenantID, mappingID); svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *InventoryHandler) GetServiceCosts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	costs, err := h.svc.GetServiceCosts(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, costs)
}
