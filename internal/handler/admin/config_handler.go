package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type ConfigHandler struct {
	svc      *service.ConfigService
	validate *validator.Validate
}

func NewConfigHandler(svc *service.ConfigService, validate *validator.Validate) *ConfigHandler {
	return &ConfigHandler{svc: svc, validate: validate}
}

func (h *ConfigHandler) GetCurrentConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	config, err := h.svc.GetCurrentConfig(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, config)
}

func (h *ConfigHandler) PushConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
		return
	}

	userID := middleware.GetUserID(r.Context())

	config, err := h.svc.PushConfig(r.Context(), tenantID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, config)
}

func (h *ConfigHandler) BroadcastConfig(w http.ResponseWriter, r *http.Request) {
	var req domain.PushConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	userID := middleware.GetUserID(r.Context())

	configs, err := h.svc.BroadcastConfig(r.Context(), req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, configs)
}
