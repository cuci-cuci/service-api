package pos

import (
	"net/http"

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
