package handler

import (
	"io"
	"log/slog"
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
// Always returns HTTP 200 to Xendit except for auth failures (401),
// to prevent infinite webhook retries.
func (h *WebhookHandler) XenditCallback(w http.ResponseWriter, r *http.Request) {
	callbackToken := r.Header.Get("x-callback-token")
	if callbackToken == "" {
		response.Error(w, apperror.Unauthorized("missing x-callback-token"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		slog.Error("webhook: failed to read body", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{"status": "received"})
		return
	}

	if err := h.gatewaySvc.ProcessWebhook(r.Context(), callbackToken, body); err != nil {
		// Only return non-200 for auth failures so Xendit stops retrying bad tokens
		if appErr, ok := apperror.IsAppError(err); ok && appErr.Code == http.StatusUnauthorized {
			response.Error(w, err)
			return
		}
		// For all other errors, return 200 to prevent infinite retries
		slog.Error("webhook: processing failed", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{"status": "received"})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "received"})
}
