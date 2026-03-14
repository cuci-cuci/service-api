package admin

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type AdminInventoryHandler struct {
	svc *service.AdminInventoryService
}

func NewAdminInventoryHandler(svc *service.AdminInventoryService) *AdminInventoryHandler {
	return &AdminInventoryHandler{svc: svc}
}

func (h *AdminInventoryHandler) ListSupplies(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		response.Error(w, apperror.Validation("tenant_id is required"))
		return
	}
	tid, err := uuid.Parse(tenantIDStr)
	if err != nil {
		response.Error(w, apperror.Validation("invalid tenant_id"))
		return
	}

	supplies, svcErr := h.svc.ListSupplies(r.Context(), tid)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, supplies)
}

func (h *AdminInventoryHandler) LowStockAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.svc.LowStockAlertsGlobal(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, alerts)
}

func (h *AdminInventoryHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.Summary(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}
