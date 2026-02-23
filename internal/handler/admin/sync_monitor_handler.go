package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

func (h *SyncMonitorHandler) OutletHealth(w http.ResponseWriter, r *http.Request) {
	health, err := h.svc.GetOutletHealth(r.Context())
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

// StreamHealth provides a Server-Sent Events endpoint that pushes outlet health
// and global health data every 5 seconds until the client disconnects.
func (h *SyncMonitorHandler) StreamHealth(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send immediately on connect, then every tick
	send := func() {
		global, err := h.svc.GetGlobalHealth(r.Context())
		if err != nil {
			slog.Error("SSE: failed to get global health", "error", err)
			return
		}
		outlets, err := h.svc.GetOutletHealth(r.Context())
		if err != nil {
			slog.Error("SSE: failed to get outlet health", "error", err)
			return
		}

		payload := map[string]any{
			"global":  global,
			"outlets": outlets,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("SSE: failed to marshal", "error", err)
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	send()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
