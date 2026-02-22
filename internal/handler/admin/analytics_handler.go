package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
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

func (h *AnalyticsHandler) RevenueByOutlet(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		response.Error(w, apperror.NewAppError(400, "invalid tenant UUID"))
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	results, svcErr := h.svc.RevenueByOutlet(r.Context(), tenantID, startDate, endDate)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, results)
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
