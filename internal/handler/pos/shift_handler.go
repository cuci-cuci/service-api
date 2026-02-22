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

type ShiftHandler struct {
	svc      *service.ShiftService
	validate *validator.Validate
}

func NewShiftHandler(svc *service.ShiftService, validate *validator.Validate) *ShiftHandler {
	return &ShiftHandler{svc: svc, validate: validate}
}

func (h *ShiftHandler) Open(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}
	userID := middleware.GetUserID(r.Context())

	var req domain.OpenShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(w, apperror.Validation(err.Error()))
		return
	}

	shift, svcErr := h.svc.OpenShift(r.Context(), *tenantID, userID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusCreated, shift)
}

func (h *ShiftHandler) Close(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// First get the current open shift for this cashier
	current, err := h.svc.GetCurrentShift(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	if current == nil {
		response.Error(w, apperror.NotFound("no open shift found"))
		return
	}

	var req domain.CloseShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(w, apperror.Validation(err.Error()))
		return
	}

	shift, svcErr := h.svc.CloseShift(r.Context(), current.ID, userID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, shift)
}

func (h *ShiftHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	shift, err := h.svc.GetCurrentShift(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Return null data if no open shift (not an error)
	response.JSON(w, http.StatusOK, shift)
}

func (h *ShiftHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	params := pagination.ParseFromRequest(r)

	shifts, total, err := h.svc.ListShifts(r.Context(), *tenantID, params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, shifts, params.ToMeta(total))
}

func (h *ShiftHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, apperror.Validation("invalid shift ID"))
		return
	}

	summary, svcErr := h.svc.GetShiftSummary(r.Context(), id)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}
