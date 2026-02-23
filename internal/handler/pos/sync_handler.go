package pos

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type SyncHandler struct {
	svc        *service.SyncService
	billingSvc *service.BillingService
	validate   *validator.Validate
}

func NewSyncHandler(svc *service.SyncService, billingSvc *service.BillingService, validate *validator.Validate) *SyncHandler {
	return &SyncHandler{svc: svc, billingSvc: billingSvc, validate: validate}
}

func (h *SyncHandler) Upload(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	var req domain.SyncUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.Validation("invalid request body"))
		return
	}

	if len(req.Transactions) == 0 {
		response.Error(w, apperror.Validation("transactions array is required"))
		return
	}

	// Use the first transaction's outlet_id as the outlet for this sync session
	outletID := req.Transactions[0].OutletID
	if outletID == uuid.Nil {
		response.Error(w, apperror.Validation("outlet_id is required in transactions"))
		return
	}

	// Check subscription transaction limit before uploading
	if h.billingSvc != nil {
		if err := h.billingSvc.CheckTransactionLimit(r.Context(), *tenantID); err != nil {
			response.Error(w, err)
			return
		}
	}

	result, err := h.svc.Upload(r.Context(), *tenantID, outletID, req.Transactions)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Track usage
	if h.billingSvc != nil && result.Inserted > 0 {
		_ = h.billingSvc.IncrementUsage(r.Context(), *tenantID, result.Inserted)
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *SyncHandler) Download(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == nil {
		response.Error(w, apperror.Forbidden("tenant context required"))
		return
	}

	currentVersion := 0
	if v := r.URL.Query().Get("current_config_version"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			currentVersion = parsed
		}
	}

	result, err := h.svc.Download(r.Context(), *tenantID, currentVersion)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
