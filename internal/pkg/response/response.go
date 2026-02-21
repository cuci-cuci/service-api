package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

func JSON[T any](w http.ResponseWriter, statusCode int, data T) {
	resp := domain.APIResponse[T]{Data: data}
	writeJSON(w, statusCode, resp)
}

func JSONWithMeta[T any](w http.ResponseWriter, statusCode int, data T, meta *domain.PaginationMeta) {
	resp := domain.APIResponse[T]{Data: data, Meta: meta}
	writeJSON(w, statusCode, resp)
}

func Error(w http.ResponseWriter, err error) {
	if appErr, ok := apperror.IsAppError(err); ok {
		apiErr := domain.APIError{
			Code:    appErr.Code,
			Message: appErr.Message,
		}
		writeJSON(w, appErr.Code, apiErr)
		return
	}

	slog.Error("unexpected error", "error", err)
	apiErr := domain.APIError{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	}
	writeJSON(w, http.StatusInternalServerError, apiErr)
}

func ValidationError(w http.ResponseWriter, errors map[string]string) {
	apiErr := domain.APIError{
		Code:    http.StatusUnprocessableEntity,
		Message: "validation error",
		Details: errors,
	}
	writeJSON(w, http.StatusUnprocessableEntity, apiErr)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
