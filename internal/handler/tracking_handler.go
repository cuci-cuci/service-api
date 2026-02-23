package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type TrackingHandler struct {
	orderSvc *service.OrderService
}

func NewTrackingHandler(orderSvc *service.OrderService) *TrackingHandler {
	return &TrackingHandler{orderSvc: orderSvc}
}

func (h *TrackingHandler) GetByToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "tracking token required"})
		return
	}

	order, err := h.orderSvc.GetByTrackingToken(r.Context(), token)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, order)
}
