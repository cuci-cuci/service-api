package pos

import (
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type OutletHandler struct {
	svc *service.OutletService
}

func NewOutletHandler(svc *service.OutletService) *OutletHandler {
	return &OutletHandler{svc: svc}
}

func (h *OutletHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	outlets, _, err := h.svc.ListByTenant(r.Context(), *tenantID, pagination.Params{Page: 1, PerPage: pagination.MaxPerPage})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, outlets)
}
