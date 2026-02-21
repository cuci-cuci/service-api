package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type FeatureFlagHandler struct {
	svc      *service.FeatureFlagService
	validate *validator.Validate
}

func NewFeatureFlagHandler(svc *service.FeatureFlagService, validate *validator.Validate) *FeatureFlagHandler {
	return &FeatureFlagHandler{svc: svc, validate: validate}
}

func (h *FeatureFlagHandler) ListFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := h.svc.ListFlags(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flags)
}

func (h *FeatureFlagHandler) CreateFlag(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	flag, err := h.svc.CreateFlag(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, flag)
}

func (h *FeatureFlagHandler) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid flag ID"))
		return
	}

	var req domain.UpdateFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	flag, err := h.svc.UpdateFlag(r.Context(), id, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flag)
}

func (h *FeatureFlagHandler) GetTenantFlags(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	flags, err := h.svc.GetTenantFlags(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flags)
}

func (h *FeatureFlagHandler) SetTenantFlag(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	var req domain.ToggleFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	// The flag ID comes from the request body or query - for simplicity, use a query param
	flagIDStr := r.URL.Query().Get("flag_id")
	if flagIDStr == "" {
		response.Error(w, apperror.Validation("flag_id query parameter is required"))
		return
	}

	flagID, err := uuid.Parse(flagIDStr)
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid flag ID"))
		return
	}

	if err := h.svc.SetTenantFlag(r.Context(), tenantID, flagID, req.Enabled); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
