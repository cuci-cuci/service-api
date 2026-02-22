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
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type UserHandler struct {
	svc      *service.UserService
	audit    *service.AuditService
	validate *validator.Validate
}

func NewUserHandler(svc *service.UserService, audit *service.AuditService, validate *validator.Validate) *UserHandler {
	return &UserHandler{svc: svc, audit: audit, validate: validate}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	users, total, err := h.svc.List(r.Context(), params)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, users, params.ToMeta(total))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationErrors(err))
		return
	}

	user, err := h.svc.Create(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	// FIX 5: Audit log for user creation
	actorID := middleware.GetUserID(r.Context())
	claims := middleware.GetClaims(r.Context())
	actorName := ""
	if claims != nil {
		actorName = claims.Subject
	}
	h.audit.LogAction(r.Context(), actorID, actorName, "create", "user", user.ID.String(), nil, user, user.TenantID)

	response.JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid user ID"))
		return
	}

	var req struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	user, err := h.svc.UpdateUser(r.Context(), id, req.Name, req.IsActive, "")
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid user ID"))
		return
	}

	isActive := false
	_, err = h.svc.UpdateUser(r.Context(), id, "", &isActive, "")
	if err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
