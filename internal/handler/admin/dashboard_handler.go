package admin

import (
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}

func (h *DashboardHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetRevenue(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, data)
}

func (h *DashboardHandler) TransactionList(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	transactions, total, err := h.svc.ListTransactions(r.Context(), params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, transactions, params.ToMeta(total))
}

func (h *DashboardHandler) TenantHealth(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetTenantHealth(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, data)
}

func (h *DashboardHandler) TenantHealthOverview(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.TenantHealthOverview(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, data)
}
