package pos

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type MemberHandler struct {
	svc *service.MemberService
}

func NewMemberHandler(svc *service.MemberService) *MemberHandler {
	return &MemberHandler{svc: svc}
}

func (h *MemberHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		response.Error(w, apperror.Validation("phone query parameter is required"))
		return
	}

	member, err := h.svc.LookupByPhone(r.Context(), phone)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, member)
}

func (h *MemberHandler) Search(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		response.Error(w, apperror.Validation("phone query parameter is required"))
		return
	}

	member, err := h.svc.LookupByPhone(r.Context(), phone)
	if err != nil {
		// Return empty array instead of 404 for search
		if appErr, ok := apperror.IsAppError(err); ok && appErr.Code == 404 {
			response.JSON(w, http.StatusOK, map[string]any{"data": []any{}})
			return
		}
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": []any{member}})
}

func (h *MemberHandler) Register(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	var req domain.CreateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if req.Name == "" || req.Phone == "" {
		response.Error(w, apperror.Validation("name and phone are required"))
		return
	}

	member, err := h.svc.Create(r.Context(), tenantID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"data": member})
}

func (h *MemberHandler) RedeemPoints(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Unauthorized("tenant not found"))
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.Validation("invalid member id"))
		return
	}

	var req struct {
		Points int `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	discountAmount, remainingPoints, err := h.svc.RedeemPoints(r.Context(), *tenantID, memberID, req.Points)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"discount_amount":  discountAmount,
			"remaining_points": remainingPoints,
			"points_redeemed":  req.Points,
		},
	})
}

func (h *MemberHandler) AwardPoints(w http.ResponseWriter, r *http.Request) {
	memberID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.Validation("invalid member id"))
		return
	}

	var req struct {
		TransactionAmount int64 `json:"transaction_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	pointsEarned, err := h.svc.AwardPoints(r.Context(), memberID, req.TransactionAmount)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"points_earned": pointsEarned,
		},
	})
}
