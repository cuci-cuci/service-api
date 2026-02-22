package owner

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/validation"
)

func validationErrors(err error) map[string]string {
	return validation.Errors(err)
}

func getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		return uuid.Nil, apperror.Forbidden("tenant context required")
	}
	return *tenantID, nil
}
