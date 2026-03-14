package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/service"
)

type Scheduler struct {
	notifSvc     *service.NotificationService
	inventorySvc *service.InventoryService
}

func New(notifSvc *service.NotificationService, inventorySvc *service.InventoryService) *Scheduler {
	return &Scheduler{notifSvc: notifSvc, inventorySvc: inventorySvc}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
	slog.Info("daily summary scheduler started")
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case now := <-ticker.C:
			s.checkAndSendSummaries(ctx, now)
		}
	}
}

func (s *Scheduler) checkAndSendSummaries(ctx context.Context, now time.Time) {
	settings, err := s.notifSvc.GetAllTenantsWithSummaryEnabled(ctx)
	if err != nil {
		slog.Error("scheduler: failed to get tenants", "error", err)
		return
	}

	currentHHMM := now.Format("15:04")

	for _, ns := range settings {
		// Skip if already sent today
		if ns.DailySummaryLastSentAt != nil {
			if ns.DailySummaryLastSentAt.Format("2006-01-02") == now.Format("2006-01-02") {
				continue
			}
		}

		// Check if it's time
		scheduledTime := strings.TrimSpace(ns.DailySummaryTime)
		if scheduledTime == "" {
			scheduledTime = "20:00"
		}
		if currentHHMM != scheduledTime {
			continue
		}

		if ns.OwnerPhone == nil || ns.FonnteAPIToken == nil {
			continue
		}

		go func(tenantID uuid.UUID, phone, token string) {
			sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.notifSvc.BuildAndSendDailySummary(sendCtx, tenantID, phone, token); err != nil {
				slog.Error("scheduler: failed to send daily summary", "tenant_id", tenantID, "error", err)
			} else {
				slog.Info("scheduler: daily summary sent", "tenant_id", tenantID)
			}

			// Also check low stock and send alert alongside the daily summary
			if s.inventorySvc != nil {
				s.sendLowStockAlert(sendCtx, tenantID)
			}
		}(ns.TenantID, *ns.OwnerPhone, *ns.FonnteAPIToken)
	}
}

// sendLowStockAlert checks for low stock items and sends a WhatsApp alert.
func (s *Scheduler) sendLowStockAlert(ctx context.Context, tenantID uuid.UUID) {
	alerts, err := s.inventorySvc.GetLowStockAlerts(ctx, tenantID)
	if err != nil {
		slog.Warn("scheduler: failed to get low stock alerts", "tenant_id", tenantID, "error", err)
		return
	}
	if len(alerts) == 0 {
		return
	}

	var items []service.LowStockAlertItem
	for _, a := range alerts {
		items = append(items, service.LowStockAlertItem{
			Name:         a.Name,
			CurrentStock: a.CurrentStock,
			Unit:         a.Unit,
			MinStock:     a.MinStock,
		})
	}
	s.notifSvc.SendLowStockAlert(ctx, tenantID, items)
	slog.Info("scheduler: low stock alert sent", "tenant_id", tenantID, "items", len(items))
}
