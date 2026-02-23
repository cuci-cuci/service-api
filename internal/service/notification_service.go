package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/config"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type NotificationSettings struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	WhatsAppEnabled  bool      `json:"whatsapp_enabled"`
	FonnteAPIToken   *string   `json:"fonnte_api_token,omitempty"`
	NotifyOnReceived bool      `json:"notify_on_received"`
	NotifyOnDone     bool      `json:"notify_on_done"`
	NotifyOnPickedUp bool      `json:"notify_on_picked_up"`
}

type NotificationService struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewNotificationService(db *pgxpool.Pool, cfg *config.Config) *NotificationService {
	return &NotificationService{db: db, cfg: cfg}
}

// GetSettings returns notification settings for a tenant.
func (s *NotificationService) GetSettings(ctx context.Context, tenantID uuid.UUID) (*NotificationSettings, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var ns NotificationSettings
	err := q.QueryRow(ctx,
		`SELECT id, tenant_id, whatsapp_enabled, fonnte_api_token, notify_on_received, notify_on_done, notify_on_picked_up
		 FROM tenant_notification_settings WHERE tenant_id = $1`, tenantID).
		Scan(&ns.ID, &ns.TenantID, &ns.WhatsAppEnabled, &ns.FonnteAPIToken,
			&ns.NotifyOnReceived, &ns.NotifyOnDone, &ns.NotifyOnPickedUp)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &NotificationSettings{TenantID: tenantID}, nil
		}
		return nil, apperror.Internal("failed to get notification settings", err)
	}
	return &ns, nil
}

// UpsertSettings creates or updates notification settings.
func (s *NotificationService) UpsertSettings(ctx context.Context, tenantID uuid.UUID, req UpdateNotificationSettingsRequest) (*NotificationSettings, error) {
	q := middleware.GetQuerier(ctx, s.db)

	now := time.Now()
	var ns NotificationSettings
	err := q.QueryRow(ctx,
		`INSERT INTO tenant_notification_settings (id, tenant_id, whatsapp_enabled, fonnte_api_token, notify_on_received, notify_on_done, notify_on_picked_up, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   whatsapp_enabled = EXCLUDED.whatsapp_enabled,
		   fonnte_api_token = EXCLUDED.fonnte_api_token,
		   notify_on_received = EXCLUDED.notify_on_received,
		   notify_on_done = EXCLUDED.notify_on_done,
		   notify_on_picked_up = EXCLUDED.notify_on_picked_up,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, tenant_id, whatsapp_enabled, fonnte_api_token, notify_on_received, notify_on_done, notify_on_picked_up`,
		uuid.New(), tenantID, req.WhatsAppEnabled, req.FonnteAPIToken,
		req.NotifyOnReceived, req.NotifyOnDone, req.NotifyOnPickedUp, now).
		Scan(&ns.ID, &ns.TenantID, &ns.WhatsAppEnabled, &ns.FonnteAPIToken,
			&ns.NotifyOnReceived, &ns.NotifyOnDone, &ns.NotifyOnPickedUp)
	if err != nil {
		return nil, apperror.Internal("failed to upsert notification settings", err)
	}
	return &ns, nil
}

// NotifyOrderStatus sends a WhatsApp notification for an order status change.
// This is fire-and-forget — errors are logged but don't block the caller.
func (s *NotificationService) NotifyOrderStatus(ctx context.Context, tenantID uuid.UUID, orderID uuid.UUID, phone string, status string, customerName string, orderNumber string, trackingToken string) {
	if phone == "" {
		return
	}

	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil || !settings.WhatsAppEnabled || settings.FonnteAPIToken == nil || *settings.FonnteAPIToken == "" {
		return
	}

	// Check if this status triggers a notification
	shouldNotify := false
	switch status {
	case "received":
		shouldNotify = settings.NotifyOnReceived
	case "done":
		shouldNotify = settings.NotifyOnDone
	case "picked_up":
		shouldNotify = settings.NotifyOnPickedUp
	}
	if !shouldNotify {
		return
	}

	message := s.buildMessage(status, customerName, orderNumber, trackingToken)
	if message == "" {
		return
	}

	// Send async — don't block request
	go s.sendAndLog(tenantID, orderID, phone, message, *settings.FonnteAPIToken)
}

func (s *NotificationService) buildMessage(status, customerName, orderNumber, trackingToken string) string {
	name := customerName
	if name == "" {
		name = "Pelanggan"
	}

	trackingInfo := ""
	if trackingToken != "" {
		trackingInfo = fmt.Sprintf("\n\nLacak pesanan: %s", trackingToken)
	}

	switch status {
	case "received":
		return fmt.Sprintf("Halo %s! 👋\n\nPesanan laundry Anda (#%s) sudah kami terima dan sedang diproses.\n\nKami akan kirim notifikasi ketika cucian Anda selesai.%s\n\nTerima kasih! 🙏",
			name, orderNumber, trackingInfo)
	case "done":
		return fmt.Sprintf("Halo %s! ✨\n\nCucian Anda (#%s) sudah selesai dan siap diambil!\n\nSilakan datang ke outlet kami untuk mengambil cucian Anda.%s\n\nTerima kasih! 🙏",
			name, orderNumber, trackingInfo)
	case "picked_up":
		return fmt.Sprintf("Halo %s! 🎉\n\nCucian Anda (#%s) sudah diambil.\n\nTerima kasih telah menggunakan jasa kami! Sampai jumpa lagi 👋",
			name, orderNumber)
	}
	return ""
}

func (s *NotificationService) sendAndLog(tenantID, orderID uuid.UUID, phone, message, apiToken string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	status := "sent"
	providerResp := ""

	err := s.sendFonnte(apiToken, phone, message)
	if err != nil {
		status = "failed"
		providerResp = err.Error()
		slog.Error("failed to send WhatsApp notification",
			"error", err, "tenant_id", tenantID, "order_id", orderID, "phone", phone)
	}

	// Log the notification
	_, logErr := s.db.Exec(ctx,
		`INSERT INTO notification_logs (id, tenant_id, order_id, phone, message, status, provider_response, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.New(), tenantID, orderID, phone, message, status, providerResp, time.Now())
	if logErr != nil {
		slog.Error("failed to log notification", "error", logErr)
	}
}

func (s *NotificationService) sendFonnte(apiToken, phone, message string) error {
	data := url.Values{}
	data.Set("target", phone)
	data.Set("message", message)

	req, err := http.NewRequest("POST", s.cfg.FonnteAPIURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("fonnte API error: status %d", resp.StatusCode)
	}
	return nil
}

type UpdateNotificationSettingsRequest struct {
	WhatsAppEnabled  bool    `json:"whatsapp_enabled"`
	FonnteAPIToken   *string `json:"fonnte_api_token"`
	NotifyOnReceived bool    `json:"notify_on_received"`
	NotifyOnDone     bool    `json:"notify_on_done"`
	NotifyOnPickedUp bool    `json:"notify_on_picked_up"`
}
