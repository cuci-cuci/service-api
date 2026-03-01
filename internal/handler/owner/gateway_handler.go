package owner

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type GatewayHandler struct {
	svc      *service.GatewayService
	validate *validator.Validate
}

func NewGatewayHandler(svc *service.GatewayService, validate *validator.Validate) *GatewayHandler {
	return &GatewayHandler{svc: svc, validate: validate}
}

func (h *GatewayHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	cfg, svcErr := h.svc.GetConfig(r.Context(), tenantID)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *GatewayHandler) UpsertConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req domain.UpsertGatewayConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	cfg, svcErr := h.svc.UpsertConfig(r.Context(), tenantID, req)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *GatewayHandler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req domain.UpdateGatewayEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	cfg, svcErr := h.svc.SetEnabled(r.Context(), tenantID, req.IsEnabled)
	if svcErr != nil {
		response.Error(w, svcErr)
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}
