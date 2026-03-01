package handler

import (
	"io"
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type WebhookHandler struct {
	gatewaySvc *service.GatewayService
}

func NewWebhookHandler(gatewaySvc *service.GatewayService) *WebhookHandler {
	return &WebhookHandler{gatewaySvc: gatewaySvc}
}

// XenditCallback handles POST /webhooks/xendit.
// Public endpoint; authentication is via the x-callback-token header.
func (h *WebhookHandler) XenditCallback(w http.ResponseWriter, r *http.Request) {
	callbackToken := r.Header.Get("x-callback-token")
	if callbackToken == "" {
		response.Error(w, apperror.Unauthorized("missing x-callback-token"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		response.Error(w, apperror.Validation("failed to read request body"))
		return
	}

	if err := h.gatewaySvc.ProcessWebhook(r.Context(), callbackToken, body); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "received"})
}
