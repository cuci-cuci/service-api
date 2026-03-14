package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type SyncService struct {
	db            *pgxpool.Pool
	configService *ConfigService
	templateSvc   *ServiceTemplateService
	orderSvc      *OrderService
	memberSvc     *MemberService
	inventorySvc  *InventoryService
	notifSvc      *NotificationService
}

func NewSyncService(db *pgxpool.Pool, configService *ConfigService, templateSvc *ServiceTemplateService, orderSvc *OrderService, memberSvc *MemberService) *SyncService {
	return &SyncService{db: db, configService: configService, templateSvc: templateSvc, orderSvc: orderSvc, memberSvc: memberSvc}
}

// SetInventoryService sets the inventory service for auto-deduct on sync.
func (s *SyncService) SetInventoryService(invSvc *InventoryService) {
	s.inventorySvc = invSvc
}

// SetNotificationService sets the notification service for low stock alerts after deduction.
func (s *SyncService) SetNotificationService(notifSvc *NotificationService) {
	s.notifSvc = notifSvc
}

func (s *SyncService) Upload(ctx context.Context, tenantID uuid.UUID, outletID uuid.UUID, transactions []domain.Transaction) (*domain.SyncUploadResult, error) {
	q := middleware.GetQuerier(ctx, s.db)
	startedAt := time.Now()

	// FIX 7: Use pgx.Batch for bulk inserts instead of individual Exec calls in a loop.
	batch := &pgx.Batch{}
	now := time.Now()
	for _, tx := range transactions {
		batch.Queue(
			`INSERT INTO transactions (id, tenant_id, outlet_id, local_order_number, customer_name, member_id,
			 items, subtotal, discount_amount, tax_amount, total_amount, payment_status, payments,
			 status, config_version_id, notes, created_by, created_at, synced_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
			 ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, synced_at = EXCLUDED.synced_at`,
			tx.ID, tenantID, outletID, tx.LocalOrderNumber, tx.CustomerName, tx.MemberID,
			tx.Items, tx.Subtotal, tx.DiscountAmount, tx.TaxAmount, tx.TotalAmount,
			tx.PaymentStatus, tx.Payments, tx.Status, tx.ConfigVersionID, tx.Notes,
			tx.CreatedBy, tx.CreatedAt, now)
	}

	br := q.SendBatch(ctx, batch)
	inserted := 0
	for range transactions {
		ct, err := br.Exec()
		if err != nil {
			slog.Error("failed to insert transaction in batch", "error", err)
			continue
		}
		if ct.RowsAffected() > 0 {
			inserted++
		}
	}
	if err := br.Close(); err != nil {
		slog.Error("failed to close batch", "error", err)
	}

	// Pre-fetch open shifts for this outlet to avoid N+1 queries
	openShifts := make(map[uuid.UUID]uuid.UUID) // cashierID -> shiftID
	shiftRows, err := q.Query(ctx,
		`SELECT cashier_id, id FROM shifts WHERE outlet_id = $1 AND status = 'open'`, outletID)
	if err == nil {
		defer shiftRows.Close()
		for shiftRows.Next() {
			var cashierID, shiftID uuid.UUID
			if err := shiftRows.Scan(&cashierID, &shiftID); err == nil {
				openShifts[cashierID] = shiftID
			}
		}
	}

	// Link transactions to open shifts using pre-fetched map
	shiftBatch := &pgx.Batch{}
	for _, tx := range transactions {
		var shiftID uuid.UUID
		if tx.ShiftID != nil {
			// Use the shift_id provided by the POS client
			shiftID = *tx.ShiftID
		} else if sid, ok := openShifts[tx.CreatedBy]; ok {
			shiftID = sid
		} else {
			continue
		}
		shiftBatch.Queue(
			`INSERT INTO shift_transactions (shift_id, transaction_id) VALUES ($1, $2) ON CONFLICT (shift_id, transaction_id) DO NOTHING`,
			shiftID, tx.ID)
	}
	if shiftBatch.Len() > 0 {
		sbr := q.SendBatch(ctx, shiftBatch)
		for i := 0; i < shiftBatch.Len(); i++ {
			if _, err := sbr.Exec(); err != nil {
				slog.Error("failed to link transaction to shift", "error", err)
			}
		}
		if err := sbr.Close(); err != nil {
			slog.Error("failed to close shift batch", "error", err)
		}
	}

	// Auto-create orders for newly inserted transactions
	for _, tx := range transactions {
		if err := s.orderSvc.CreateFromTransaction(ctx, tenantID, tx, tx.CreatedBy); err != nil {
			slog.Warn("failed to create order from transaction", "transaction_id", tx.ID, "error", err)
		}
	}

	// Update member spending for transactions with a member_id
	for _, tx := range transactions {
		if tx.MemberID != nil {
			_, err := s.memberSvc.UpdateSpending(ctx, *tx.MemberID, tx.TotalAmount)
			if err != nil {
				slog.Warn("failed to update member spending", "member_id", tx.MemberID, "error", err)
			}
			// Award referral points on first transaction
			s.memberSvc.AwardReferralPoints(ctx, *tx.MemberID)
		}
	}

	// Auto-deduct stock for completed/paid transactions
	if s.inventorySvc != nil {
		s.autoDeductStockForTransactions(ctx, tenantID, transactions)
	}

	// Record sync session
	completedAt := time.Now()
	_, syncErr := q.Exec(ctx,
		`INSERT INTO sync_sessions (id, tenant_id, outlet_id, direction, status, transaction_count, started_at, completed_at)
		 VALUES ($1, $2, $3, 'upload', 'completed', $4, $5, $6)`,
		uuid.New(), tenantID, outletID, len(transactions), startedAt, completedAt)
	if syncErr != nil {
		slog.Error("failed to record sync session", "error", syncErr)
	}

	return &domain.SyncUploadResult{
		Received: len(transactions),
		Inserted: inserted,
		Skipped:  len(transactions) - inserted,
	}, nil
}

func (s *SyncService) Download(ctx context.Context, tenantID uuid.UUID, currentVersion int) (*domain.SyncDownloadResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)
	resp := &domain.SyncDownloadResponse{}

	// Get latest config version
	cv, err := s.configService.GetCurrentConfig(ctx, tenantID)
	if err != nil {
		if appErr, ok := apperror.IsAppError(err); ok && appErr.Code == 404 {
			resp.ConfigVersion = 0
			resp.Services = []domain.ServiceWithPrice{}
			resp.Categories = []domain.ServiceCategory{}
			resp.Members = []domain.Member{}
			return resp, nil
		}
		return nil, err
	}

	resp.ConfigVersion = cv.Version

	if cv.Version > currentVersion {
		resp.Config = cv
	}

	// Always return fresh data — services + categories via service methods
	services, err := s.templateSvc.GetServicesWithPrices(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp.Services = services

	categories, _, err := s.templateSvc.ListCategories(ctx, pagination.Params{Page: 1, PerPage: pagination.MaxPerPage})
	if err != nil {
		return nil, err
	}
	resp.Categories = categories

	// Batch 4 remaining queries into a single round-trip (members + outlets + payment methods + delivery zones)
	dlBatch := &pgx.Batch{}
	dlBatch.Queue(
		`SELECT id, tenant_id, name, phone, email, tier, discount_percent, total_points, total_spending,
		        referral_code, referred_by_member_id, has_first_transaction, created_at
		 FROM members WHERE tenant_id = $1 OR tenant_id IS NULL`, tenantID)
	dlBatch.Queue(
		`SELECT id, tenant_id, name, address, phone, is_active FROM outlets WHERE tenant_id = $1 AND is_active = true`, tenantID)
	dlBatch.Queue(
		`SELECT id, tenant_id, name, type, is_active, sort_order, created_at FROM payment_methods WHERE tenant_id = $1 AND is_active = true ORDER BY sort_order`, tenantID)
	dlBatch.Queue(
		`SELECT id, tenant_id, outlet_id, name, district, fee, estimated_minutes, is_active, created_at, updated_at
		 FROM delivery_zones WHERE tenant_id = $1 AND is_active = true ORDER BY name`, tenantID)

	dlBR := q.SendBatch(ctx, dlBatch)
	defer dlBR.Close()

	// Result 1: Members
	memberRows, err := dlBR.Query()
	if err != nil {
		return nil, apperror.Internal("failed to get members", err)
	}
	var members []domain.Member
	for memberRows.Next() {
		var m domain.Member
		if err := memberRows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.TotalSpending,
			&m.ReferralCode, &m.ReferredByMemberID, &m.HasFirstTransaction, &m.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan member", err)
		}
		members = append(members, m)
	}
	memberRows.Close()
	if members == nil {
		members = []domain.Member{}
	}
	resp.Members = members

	// Result 2: Outlets
	outletRows, err := dlBR.Query()
	if err != nil {
		return nil, apperror.Internal("failed to get outlets", err)
	}
	var outlets []domain.Outlet
	for outletRows.Next() {
		var o domain.Outlet
		if err := outletRows.Scan(&o.ID, &o.TenantID, &o.Name, &o.Address, &o.Phone, &o.IsActive); err != nil {
			return nil, apperror.Internal("failed to scan outlet", err)
		}
		outlets = append(outlets, o)
	}
	outletRows.Close()
	if outlets == nil {
		outlets = []domain.Outlet{}
	}
	resp.Outlets = outlets

	// Result 3: Payment Methods
	pmRows, err := dlBR.Query()
	if err != nil {
		return nil, apperror.Internal("failed to get payment methods", err)
	}
	var paymentMethods []domain.PaymentMethod
	for pmRows.Next() {
		var pm domain.PaymentMethod
		if err := pmRows.Scan(&pm.ID, &pm.TenantID, &pm.Name, &pm.Type, &pm.IsActive, &pm.SortOrder, &pm.CreatedAt); err != nil {
			return nil, apperror.Internal("failed to scan payment method", err)
		}
		paymentMethods = append(paymentMethods, pm)
	}
	pmRows.Close()
	if paymentMethods == nil {
		paymentMethods = []domain.PaymentMethod{}
	}
	resp.PaymentMethods = paymentMethods

	// Result 4: Delivery Zones
	dzRows, err := dlBR.Query()
	if err != nil {
		return nil, apperror.Internal("failed to get delivery zones", err)
	}
	var deliveryZones []domain.DeliveryZone
	for dzRows.Next() {
		var dz domain.DeliveryZone
		if err := dzRows.Scan(&dz.ID, &dz.TenantID, &dz.OutletID, &dz.Name, &dz.District, &dz.Fee, &dz.EstimatedMinutes, &dz.IsActive, &dz.CreatedAt, &dz.UpdatedAt); err != nil {
			return nil, apperror.Internal("failed to scan delivery zone", err)
		}
		deliveryZones = append(deliveryZones, dz)
	}
	dzRows.Close()
	if deliveryZones == nil {
		deliveryZones = []domain.DeliveryZone{}
	}
	resp.DeliveryZones = deliveryZones

	return resp, nil
}

func (s *SyncService) GetSyncHealth(ctx context.Context, tenantID uuid.UUID) (*domain.SyncHealthResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	resp := &domain.SyncHealthResponse{
		TenantID: tenantID,
	}

	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM sync_sessions WHERE tenant_id = $1", tenantID).
		Scan(&resp.TotalSessions)
	if err != nil {
		return nil, apperror.Internal("failed to count sync sessions", err)
	}

	var lastSync *time.Time
	var lastStatus *string
	err = q.QueryRow(ctx,
		`SELECT completed_at, status FROM sync_sessions WHERE tenant_id = $1 ORDER BY started_at DESC LIMIT 1`, tenantID).
		Scan(&lastSync, &lastStatus)
	if err != nil {
		// No sessions yet - that's ok
		resp.LastSyncStatus = "none"
	} else {
		resp.LastSync = lastSync
		if lastStatus != nil {
			resp.LastSyncStatus = *lastStatus
		}
	}

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, outlet_id, direction, status, transaction_count, error_message, started_at, completed_at
		 FROM sync_sessions WHERE tenant_id = $1 ORDER BY started_at DESC LIMIT 10`, tenantID)
	if err != nil {
		return nil, apperror.Internal("failed to list sync sessions", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ss domain.SyncSession
		if err := rows.Scan(&ss.ID, &ss.TenantID, &ss.OutletID, &ss.Direction, &ss.Status,
			&ss.TransactionCount, &ss.ErrorMessage, &ss.StartedAt, &ss.CompletedAt); err != nil {
			return nil, apperror.Internal("failed to scan sync session", err)
		}
		resp.RecentSessions = append(resp.RecentSessions, ss)
	}
	if resp.RecentSessions == nil {
		resp.RecentSessions = []domain.SyncSession{}
	}

	return resp, nil
}

// autoDeductStockForTransactions deducts stock for each synced transaction's items
// based on the service-supply mappings. This runs during sync upload.
func (s *SyncService) autoDeductStockForTransactions(ctx context.Context, tenantID uuid.UUID, transactions []domain.Transaction) {
	q := middleware.GetQuerier(ctx, s.db)

	type txItem struct {
		ServiceID string  `json:"serviceId"`
		Quantity  float64 `json:"quantity"`
	}

	for _, tx := range transactions {
		if tx.Items == nil {
			continue
		}

		var items []txItem
		if err := json.Unmarshal(tx.Items, &items); err != nil {
			slog.Warn("autoDeductStock: failed to parse items", "tx_id", tx.ID, "err", err)
			continue
		}

		for _, item := range items {
			serviceID, parseErr := uuid.Parse(item.ServiceID)
			if parseErr != nil {
				continue
			}

			rows, err := q.Query(ctx, `
				SELECT ssm.supply_id, ssm.quantity_per_unit
				FROM service_supply_mappings ssm
				WHERE ssm.tenant_id = $1 AND ssm.service_template_id = $2
			`, tenantID, serviceID)
			if err != nil {
				slog.Warn("autoDeductStock: failed to query mappings", "service_id", serviceID, "err", err)
				continue
			}

			type mappingRow struct {
				supplyID        uuid.UUID
				quantityPerUnit float64
			}
			var mappings []mappingRow
			for rows.Next() {
				var mr mappingRow
				if scanErr := rows.Scan(&mr.supplyID, &mr.quantityPerUnit); scanErr != nil {
					continue
				}
				mappings = append(mappings, mr)
			}
			rows.Close()

			for _, mapping := range mappings {
				deductQty := mapping.quantityPerUnit * item.Quantity
				notes := fmt.Sprintf("Auto-deduct: Sync TX %s", tx.ID.String()[:8])

				_, err := q.Exec(ctx, `
					INSERT INTO stock_movements (tenant_id, supply_id, movement_type, quantity, notes, created_by)
					VALUES ($1, $2, 'out', $3, $4, $5)
				`, tenantID, mapping.supplyID, deductQty, notes, tx.CreatedBy)
				if err != nil {
					slog.Warn("autoDeductStock: failed to insert movement", "supply_id", mapping.supplyID, "err", err)
					continue
				}

				_, err = q.Exec(ctx, `
					UPDATE supplies SET current_stock = current_stock - $1, updated_at = NOW()
					WHERE id = $2 AND tenant_id = $3
				`, deductQty, mapping.supplyID, tenantID)
				if err != nil {
					slog.Warn("autoDeductStock: failed to update stock", "supply_id", mapping.supplyID, "err", err)
				}
			}
		}
	}

	// After deducting stock, check for low stock and send WhatsApp alert
	if s.notifSvc != nil {
		go func() {
			alertCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			alerts, err := s.inventorySvc.GetLowStockAlerts(alertCtx, tenantID)
			if err != nil || len(alerts) == 0 {
				return
			}

			var items []LowStockAlertItem
			for _, a := range alerts {
				items = append(items, LowStockAlertItem{
					Name:         a.Name,
					CurrentStock: a.CurrentStock,
					Unit:         a.Unit,
					MinStock:     a.MinStock,
				})
			}
			s.notifSvc.SendLowStockAlert(alertCtx, tenantID, items)
		}()
	}
}
