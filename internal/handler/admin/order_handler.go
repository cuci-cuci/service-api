package admin

import (
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type AdminOrderHandler struct {
	svc *service.AdminOrderService
}

func NewAdminOrderHandler(svc *service.AdminOrderService) *AdminOrderHandler {
	return &AdminOrderHandler{svc: svc}
}

func (h *AdminOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	status := r.URL.Query().Get("status")
	deliveryOnly := r.URL.Query().Get("delivery_only") == "true"

	orders, total, err := h.svc.List(r.Context(), params, status, deliveryOnly)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, orders, params.ToMeta(total))
}

func (h *AdminOrderHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.Summary(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}
