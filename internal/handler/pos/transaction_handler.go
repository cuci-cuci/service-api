package pos

import (
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type TransactionHandler struct {
	svc *service.MemberService
}

func NewTransactionHandler(svc *service.MemberService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	params := pagination.ParseFromRequest(r)
	transactions, total, err := h.svc.GetTransactionsByTenant(r.Context(), *tenantID, params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, transactions, params.ToMeta(total))
}
