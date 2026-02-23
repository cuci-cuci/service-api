package owner

import (
	"net/http"
	"strconv"

	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

func (h *AnalyticsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	stats, err := h.svc.TransactionStats(r.Context(), startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}

func (h *AnalyticsHandler) RevenueByOutlet(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	results, err := h.svc.RevenueByOutlet(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *AnalyticsHandler) RevenueByService(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	results, err := h.svc.RevenueByService(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *AnalyticsHandler) RevenueByPaymentMethod(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	results, err := h.svc.RevenueByPaymentMethod(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *AnalyticsHandler) DailyRevenue(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	period := 30
	if p := r.URL.Query().Get("period"); p != "" {
		if parsed, parseErr := strconv.Atoi(p); parseErr == nil && parsed > 0 {
			period = parsed
		}
	}

	results, err := h.svc.DailyRevenue(r.Context(), tenantID, period)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, results)
}
