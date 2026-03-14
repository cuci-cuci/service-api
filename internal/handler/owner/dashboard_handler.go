package owner

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type OwnerDashboardHandler struct {
	svc *service.OwnerDashboardService
}

func NewOwnerDashboardHandler(svc *service.OwnerDashboardService) *OwnerDashboardHandler {
	return &OwnerDashboardHandler{svc: svc}
}

func parseOutletID(r *http.Request) *uuid.UUID {
	oidStr := r.URL.Query().Get("outlet_id")
	if oidStr == "" {
		return nil
	}
	oid, err := uuid.Parse(oidStr)
	if err != nil {
		return nil
	}
	return &oid
}

func (h *OwnerDashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	outletID := parseOutletID(r)

	summary, err := h.svc.GetSummary(r.Context(), tenantID, outletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}

func (h *OwnerDashboardHandler) GetCashierPerformance(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	days := 30
	if p := r.URL.Query().Get("days"); p != "" {
		if parsed, parseErr := strconv.Atoi(p); parseErr == nil && parsed > 0 {
			days = parsed
		}
	}

	outletID := parseOutletID(r)

	results, err := h.svc.GetCashierPerformance(r.Context(), tenantID, days, outletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *OwnerDashboardHandler) GetCustomerInsights(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	days := 30
	if p := r.URL.Query().Get("days"); p != "" {
		if parsed, parseErr := strconv.Atoi(p); parseErr == nil && parsed > 0 {
			days = parsed
		}
	}

	insights, err := h.svc.GetCustomerInsights(r.Context(), tenantID, days)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, insights)
}

func (h *OwnerDashboardHandler) GetGoals(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	goals, err := h.svc.GetGoals(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, goals)
}

func (h *OwnerDashboardHandler) UpsertGoal(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req struct {
		GoalType    string `json:"goal_type"`
		TargetValue int64  `json:"target_value"`
	}
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if req.GoalType == "" || req.TargetValue <= 0 {
		response.Error(w, apperror.Validation("goal_type and positive target_value required"))
		return
	}

	if err := h.svc.UpsertGoal(r.Context(), tenantID, req.GoalType, req.TargetValue); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
