package admin

import (
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type SyncMonitorHandler struct {
	svc *service.SyncMonitorService
}

func NewSyncMonitorHandler(svc *service.SyncMonitorService) *SyncMonitorHandler {
	return &SyncMonitorHandler{svc: svc}
}

func (h *SyncMonitorHandler) GlobalHealth(w http.ResponseWriter, r *http.Request) {
	health, err := h.svc.GetGlobalHealth(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, health)
}

func (h *SyncMonitorHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	sessions, total, err := h.svc.ListSessions(r.Context(), params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, sessions, params.ToMeta(total))
}
