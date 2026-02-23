package owner

import (
	"encoding/json"
	"net/http"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

type ConfigHandler struct {
	svc *service.ConfigService
}

func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// StoreSettings represents the editable store fields from config_versions.data
type StoreSettings struct {
	StoreName     string  `json:"store_name"`
	Address       string  `json:"address"`
	Phone         string  `json:"phone"`
	TaxRate       float64 `json:"tax_rate"`
	ReceiptFooter string  `json:"receipt_footer"`
}

func (h *ConfigHandler) GetStoreSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	cv, err := h.svc.GetCurrentConfig(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Extract store settings from config data
	var data map[string]any
	if err := json.Unmarshal(cv.Data, &data); err != nil {
		response.Error(w, err)
		return
	}

	settings := StoreSettings{
		StoreName:     stringFromMap(data, "storeName"),
		Address:       stringFromMap(data, "address"),
		Phone:         stringFromMap(data, "phone"),
		TaxRate:       floatFromMap(data, "taxRate"),
		ReceiptFooter: stringFromMap(data, "receiptFooter"),
	}

	response.JSON(w, http.StatusOK, settings)
}

func (h *ConfigHandler) UpdateStoreSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req StoreSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	cv, err := h.svc.UpdateStoreSettings(r.Context(), tenantID, userID, service.StoreSettingsUpdate{
		StoreName:     req.StoreName,
		Address:       req.Address,
		Phone:         req.Phone,
		TaxRate:       req.TaxRate,
		ReceiptFooter: req.ReceiptFooter,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, cv)
}

func stringFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func floatFromMap(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
