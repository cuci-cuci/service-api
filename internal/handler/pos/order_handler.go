package pos

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type OrderHandler struct {
	svc      *service.OrderService
	validate *validator.Validate
}

func NewOrderHandler(svc *service.OrderService, validate *validator.Validate) *OrderHandler {
	return &OrderHandler{svc: svc, validate: validate}
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	params := pagination.ParseFromRequest(r)
	status := r.URL.Query().Get("status")
	outletID := r.URL.Query().Get("outlet_id")

	orders, total, err := h.svc.List(r.Context(), *tenantID, params, status, outletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, orders, params.ToMeta(total))
}

func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, apperror.Validation("invalid order ID"))
		return
	}

	order, svcErr := h.svc.GetByID(r.Context(), id)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}
	userID := middleware.GetUserID(r.Context())

	var req domain.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(w, apperror.Validation(err.Error()))
		return
	}

	order, svcErr := h.svc.Create(r.Context(), *tenantID, userID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, apperror.Validation("invalid order ID"))
		return
	}
	userID := middleware.GetUserID(r.Context())

	var req domain.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(w, apperror.Validation(err.Error()))
		return
	}

	order, svcErr := h.svc.UpdateStatus(r.Context(), id, userID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, order)
}

func (h *OrderHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	outletID := r.URL.Query().Get("outlet_id")

	orders, err := h.svc.ListActive(r.Context(), *tenantID, outletID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, orders)
}
