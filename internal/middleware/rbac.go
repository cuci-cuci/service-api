package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := GetUserRole(r.Context())
			if !roleSet[userRole] {
				response.Error(w, apperror.Forbidden("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireSuperadmin() func(http.Handler) http.Handler {
	return RequireRole("superadmin")
}

func RequireTenantAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetUserRole(r.Context())
			if role == "superadmin" {
				next.ServeHTTP(w, r)
				return
			}

			tenantIDParam := chi.URLParam(r, "id")
			if tenantIDParam == "" {
				tenantIDParam = chi.URLParam(r, "tenantID")
			}

			if tenantIDParam == "" {
				next.ServeHTTP(w, r)
				return
			}

			paramID, err := uuid.Parse(tenantIDParam)
			if err != nil {
				response.Error(w, apperror.NewAppError(http.StatusBadRequest, "invalid tenant ID"))
				return
			}

			userTenantID := GetTenantID(r.Context())
			if userTenantID == nil || *userTenantID != paramID {
				response.Error(w, apperror.Forbidden("access denied to this tenant"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
