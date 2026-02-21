package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type UserHandler struct {
	svc      *service.UserService
	validate *validator.Validate
}

func NewUserHandler(svc *service.UserService, validate *validator.Validate) *UserHandler {
	return &UserHandler{svc: svc, validate: validate}
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
	response.JSON(w, http.StatusCreated, user)
}
