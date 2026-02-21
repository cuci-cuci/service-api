package admin

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type AuditLogHandler struct {
	svc *service.AuditService
}

func NewAuditLogHandler(svc *service.AuditService) *AuditLogHandler {
	return &AuditLogHandler{svc: svc}
}

func (h *AuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)

	var tenantID *uuid.UUID
	role := middleware.GetUserRole(r.Context())
	if role != "superadmin" {
		tenantID = middleware.GetTenantID(r.Context())
	} else {
		if tid := r.URL.Query().Get("tenant_id"); tid != "" {
			if parsed, err := uuid.Parse(tid); err == nil {
				tenantID = &parsed
			}
		}
	}

	logs, total, err := h.svc.ListLogs(r.Context(), tenantID, params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, logs, params.ToMeta(total))
}
