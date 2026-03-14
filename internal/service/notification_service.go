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
	ID                     uuid.UUID  `json:"id"`
	TenantID               uuid.UUID  `json:"tenant_id"`
	WhatsAppEnabled        bool       `json:"whatsapp_enabled"`
	FonnteAPIToken         *string    `json:"fonnte_api_token,omitempty"`
	NotifyOnReceived       bool       `json:"notify_on_received"`
	NotifyOnDone           bool       `json:"notify_on_done"`
	NotifyOnPickedUp       bool       `json:"notify_on_picked_up"`
	DailySummaryEnabled    bool       `json:"daily_summary_enabled"`
	DailySummaryTime       string     `json:"daily_summary_time"`
	OwnerPhone             *string    `json:"owner_phone,omitempty"`
	DailySummaryLastSentAt *time.Time `json:"-"`
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
		`SELECT id, tenant_id, whatsapp_enabled, fonnte_api_token,
		        notify_on_received, notify_on_done, notify_on_picked_up,
		        daily_summary_enabled, daily_summary_time, owner_phone
		 FROM tenant_notification_settings WHERE tenant_id = $1`, tenantID).
		Scan(&ns.ID, &ns.TenantID, &ns.WhatsAppEnabled, &ns.FonnteAPIToken,
			&ns.NotifyOnReceived, &ns.NotifyOnDone, &ns.NotifyOnPickedUp,
			&ns.DailySummaryEnabled, &ns.DailySummaryTime, &ns.OwnerPhone)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &NotificationSettings{TenantID: tenantID, DailySummaryTime: "20:00"}, nil
		}
		return nil, apperror.Internal("failed to get notification settings", err)
	}
	return &ns, nil
}

// UpsertSettings creates or updates notification settings.
func (s *NotificationService) UpsertSettings(ctx context.Context, tenantID uuid.UUID, req UpdateNotificationSettingsRequest) (*NotificationSettings, error) {
	q := middleware.GetQuerier(ctx, s.db)

	now := time.Now()
	summaryTime := req.DailySummaryTime
	if summaryTime == "" {
		summaryTime = "20:00"
	}

	var ns NotificationSettings
	err := q.QueryRow(ctx,
		`INSERT INTO tenant_notification_settings
		   (id, tenant_id, whatsapp_enabled, fonnte_api_token, notify_on_received, notify_on_done, notify_on_picked_up,
		    daily_summary_enabled, daily_summary_time, owner_phone, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   whatsapp_enabled = EXCLUDED.whatsapp_enabled,
		   fonnte_api_token = EXCLUDED.fonnte_api_token,
		   notify_on_received = EXCLUDED.notify_on_received,
		   notify_on_done = EXCLUDED.notify_on_done,
		   notify_on_picked_up = EXCLUDED.notify_on_picked_up,
		   daily_summary_enabled = EXCLUDED.daily_summary_enabled,
		   daily_summary_time = EXCLUDED.daily_summary_time,
		   owner_phone = EXCLUDED.owner_phone,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, tenant_id, whatsapp_enabled, fonnte_api_token, notify_on_received, notify_on_done, notify_on_picked_up,
		           daily_summary_enabled, daily_summary_time, owner_phone`,
		uuid.New(), tenantID, req.WhatsAppEnabled, req.FonnteAPIToken,
		req.NotifyOnReceived, req.NotifyOnDone, req.NotifyOnPickedUp,
		req.DailySummaryEnabled, summaryTime, req.OwnerPhone, now).
		Scan(&ns.ID, &ns.TenantID, &ns.WhatsAppEnabled, &ns.FonnteAPIToken,
			&ns.NotifyOnReceived, &ns.NotifyOnDone, &ns.NotifyOnPickedUp,
			&ns.DailySummaryEnabled, &ns.DailySummaryTime, &ns.OwnerPhone)
	if err != nil {
		return nil, apperror.Internal("failed to upsert notification settings", err)
	}
	return &ns, nil
}

// GetAllTenantsWithSummaryEnabled returns tenants that have daily summary enabled.
func (s *NotificationService) GetAllTenantsWithSummaryEnabled(ctx context.Context) ([]NotificationSettings, error) {
	rows, err := s.db.Query(ctx,
		`SELECT tenant_id, fonnte_api_token, daily_summary_time, owner_phone, daily_summary_last_sent_at
		 FROM tenant_notification_settings
		 WHERE daily_summary_enabled = true
		   AND fonnte_api_token IS NOT NULL AND fonnte_api_token != ''
		   AND owner_phone IS NOT NULL AND owner_phone != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []NotificationSettings
	for rows.Next() {
		var ns NotificationSettings
		if err := rows.Scan(&ns.TenantID, &ns.FonnteAPIToken, &ns.DailySummaryTime, &ns.OwnerPhone, &ns.DailySummaryLastSentAt); err != nil {
			return nil, err
		}
		results = append(results, ns)
	}
	return results, nil
}

// BuildAndSendDailySummary builds and sends a daily revenue summary to the owner.
func (s *NotificationService) BuildAndSendDailySummary(ctx context.Context, tenantID uuid.UUID, phone, apiToken string) error {
	// Query today's revenue and transaction count
	var revenue int64
	var txCount int
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_amount),0), COUNT(*)
		 FROM transactions
		 WHERE tenant_id = $1 AND status = 'completed'
		   AND created_at::date = CURRENT_DATE`, tenantID).Scan(&revenue, &txCount)
	if err != nil {
		return fmt.Errorf("query daily stats: %w", err)
	}

	// Query top 3 services today
	type topService struct {
		name    string
		revenue int64
	}
	var topServices []topService
	rows, err := s.db.Query(ctx,
		`SELECT item->>'serviceName' as svc_name, COALESCE(SUM((item->>'subtotal')::bigint),0) as rev
		 FROM transactions t, jsonb_array_elements(t.items) as item
		 WHERE t.tenant_id = $1 AND t.status = 'completed'
		   AND t.created_at::date = CURRENT_DATE
		 GROUP BY svc_name
		 ORDER BY rev DESC LIMIT 3`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ts topService
			if err := rows.Scan(&ts.name, &ts.revenue); err == nil {
				topServices = append(topServices, ts)
			}
		}
	}

	// Build message
	today := time.Now().Format("02 Jan 2006")
	avg := int64(0)
	if txCount > 0 {
		avg = revenue / int64(txCount)
	}

	msg := fmt.Sprintf("📊 *Ringkasan Harian - %s*\n\n", today)
	msg += fmt.Sprintf("💰 Pendapatan: Rp %s\n", formatRupiah(revenue))
	msg += fmt.Sprintf("🧾 Transaksi: %d\n", txCount)
	msg += fmt.Sprintf("📈 Rata-rata: Rp %s\n", formatRupiah(avg))

	if len(topServices) > 0 {
		msg += "\n*Layanan Terlaris:*\n"
		for i, svc := range topServices {
			msg += fmt.Sprintf("%d. %s - Rp %s\n", i+1, svc.name, formatRupiah(svc.revenue))
		}
	}

	msg += "\nTerima kasih atas kerja keras Anda hari ini! 💪"

	if err := s.sendFonnte(apiToken, phone, msg); err != nil {
		return fmt.Errorf("send summary: %w", err)
	}

	// Mark as sent
	_, err = s.db.Exec(ctx,
		`UPDATE tenant_notification_settings SET daily_summary_last_sent_at = NOW()
		 WHERE tenant_id = $1`, tenantID)
	return err
}

func formatRupiah(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteByte('.')
		}
		result.WriteRune(c)
	}
	return result.String()
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

// SendLowStockAlert sends a WhatsApp notification to the tenant owner about low stock items.
// This is fire-and-forget — errors are logged but don't block the caller.
func (s *NotificationService) SendLowStockAlert(ctx context.Context, tenantID uuid.UUID, lowStockItems []LowStockAlertItem) {
	if len(lowStockItems) == 0 {
		return
	}

	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil || !settings.WhatsAppEnabled || settings.FonnteAPIToken == nil || *settings.FonnteAPIToken == "" {
		return
	}
	if settings.OwnerPhone == nil || *settings.OwnerPhone == "" {
		return
	}

	msg := "⚠️ *Peringatan Stok Rendah*\n\n"
	msg += "Beberapa bahan baku sudah di bawah batas minimum:\n\n"
	for _, item := range lowStockItems {
		msg += fmt.Sprintf("• *%s*: %.1f %s (min: %.1f)\n", item.Name, item.CurrentStock, item.Unit, item.MinStock)
	}
	msg += "\nSegera lakukan restock untuk menghindari kehabisan bahan. 📦"

	go func() {
		if err := s.sendFonnte(*settings.FonnteAPIToken, *settings.OwnerPhone, msg); err != nil {
			slog.Error("failed to send low stock alert",
				"error", err, "tenant_id", tenantID, "items", len(lowStockItems))
		} else {
			slog.Info("low stock alert sent", "tenant_id", tenantID, "items", len(lowStockItems))
		}
	}()
}

// LowStockAlertItem is a simplified struct for low stock notification.
type LowStockAlertItem struct {
	Name         string
	CurrentStock float64
	Unit         string
	MinStock     float64
}

type UpdateNotificationSettingsRequest struct {
	WhatsAppEnabled     bool    `json:"whatsapp_enabled"`
	FonnteAPIToken      *string `json:"fonnte_api_token"`
	NotifyOnReceived    bool    `json:"notify_on_received"`
	NotifyOnDone        bool    `json:"notify_on_done"`
	NotifyOnPickedUp    bool    `json:"notify_on_picked_up"`
	DailySummaryEnabled bool    `json:"daily_summary_enabled"`
	DailySummaryTime    string  `json:"daily_summary_time"`
	OwnerPhone          *string `json:"owner_phone"`
}
