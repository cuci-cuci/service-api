package admin

import (
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

func (h *AnalyticsHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	revenue, err := h.svc.RevenueByTenant(r.Context(), startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, revenue)
}

func (h *AnalyticsHandler) TransactionStats(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	stats, err := h.svc.TransactionStats(r.Context(), startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}
