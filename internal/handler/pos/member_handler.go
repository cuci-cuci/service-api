package pos

import (
	"encoding/json"
	"net/http"

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
