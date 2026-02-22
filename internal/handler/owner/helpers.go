package owner

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

func validationErrors(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			errs[fe.Field()] = fe.Tag()
		}
	}
	return errs
}

func getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		return uuid.Nil, apperror.Forbidden("tenant context required")
	}
	return *tenantID, nil
}
