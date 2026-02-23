package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/pkg/validation"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type AuthHandler struct {
	svc        *service.AuthService
	billingSvc *service.BillingService
	validate   *validator.Validate
}

func NewAuthHandler(svc *service.AuthService, billingSvc *service.BillingService, validate *validator.Validate) *AuthHandler {
	return &AuthHandler{svc: svc, billingSvc: billingSvc, validate: validate}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validation.Errors(err))
		return
	}

	tokenResp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Assign free plan to the new tenant
	if h.billingSvc != nil && tokenResp.User.TenantID != nil {
		if planErr := h.billingSvc.AssignFreePlan(r.Context(), *tokenResp.User.TenantID); planErr != nil {
			slog.Error("failed to assign free plan", "error", planErr, "tenant_id", tokenResp.User.TenantID)
		}
	}

	response.JSON(w, http.StatusCreated, tokenResp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validation.Errors(err))
		return
	}

	tokenResp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tokenResp)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(w, apperror.Validation("refresh_token is required"))
		return
	}

	tokenResp, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tokenResp)
}

func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req domain.RequestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validation.Errors(err))
		return
	}

	token, err := h.svc.RequestPasswordReset(r.Context(), req.Email)
	if err != nil {
		response.Error(w, err)
		return
	}

	// In production, send token via email/WhatsApp instead of returning it
	resp := map[string]string{"message": "Jika email terdaftar, Anda akan menerima link reset password."}
	if token != "" {
		resp["reset_token"] = token // TODO: Remove in production — send via email/WA
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req domain.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validation.Errors(err))
		return
	}

	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Password berhasil direset. Silakan login."})
}
