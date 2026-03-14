package owner

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type StaffHandler struct {
	svc *service.StaffActivityService
}

func NewStaffHandler(svc *service.StaffActivityService) *StaffHandler {
	return &StaffHandler{svc: svc}
}

func (h *StaffHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var userID *uuid.UUID
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		parsed, parseErr := uuid.Parse(uid)
		if parseErr == nil {
			userID = &parsed
		}
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	activities, err := h.svc.ListActivities(r.Context(), tenantID, userID, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, activities)
}

func (h *StaffHandler) GetSummaries(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, parseErr := strconv.Atoi(d); parseErr == nil && parsed > 0 {
			days = parsed
		}
	}
	summaries, err := h.svc.GetStaffSummaries(r.Context(), tenantID, days)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, summaries)
}
