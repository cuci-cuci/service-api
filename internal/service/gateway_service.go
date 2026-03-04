package service

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/config"
	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/encrypt"
)

type GatewayService struct {
	db         *pgxpool.Pool
	cfg        *config.Config
	httpClient *http.Client
}

func NewGatewayService(db *pgxpool.Pool, cfg *config.Config) *GatewayService {
	return &GatewayService{
		db:  db,
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.XenditHTTPTimeout) * time.Second,
		},
	}
}

// GetConfig returns the public gateway config for a tenant (no key material).
func (s *GatewayService) GetConfig(ctx context.Context, tenantID uuid.UUID) (*domain.GatewayConfigResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var cfg domain.PaymentGatewayConfig
	err := q.QueryRow(ctx, `
		SELECT id, tenant_id, gateway, is_enabled,
		       COALESCE(secret_key_encrypted, ''), public_key,
		       COALESCE(webhook_token_encrypted, ''), enabled_types, key_version,
		       created_at, updated_at
		FROM payment_gateway_configs
		WHERE tenant_id = $1 AND gateway = 'xendit'
	`, tenantID).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Gateway, &cfg.IsEnabled,
		&cfg.SecretKeyEncrypted, &cfg.PublicKey,
		&cfg.WebhookTokenEncrypted, &cfg.EnabledTypes, &cfg.KeyVersion,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Internal("failed to get gateway config", err)
	}

	return toGatewayConfigResponse(&cfg), nil
}

// UpsertConfig creates or updates the gateway config for a tenant.
func (s *GatewayService) UpsertConfig(ctx context.Context, tenantID uuid.UUID, req domain.UpsertGatewayConfigRequest) (*domain.GatewayConfigResponse, error) {
	if s.cfg.GatewayEncryptionKey == "" {
		return nil, apperror.Internal("gateway encryption key not configured", nil)
	}

	// Validate secret key format
	trimmedKey := strings.TrimSpace(req.SecretKey)
	if !strings.HasPrefix(trimmedKey, "xnd_") {
		return nil, apperror.Validation("secret key must start with 'xnd_'")
	}

	q := middleware.GetQuerier(ctx, s.db)

	encryptedSecret, err := encrypt.Encrypt(s.cfg.GatewayEncryptionKey, trimmedKey)
	if err != nil {
		return nil, apperror.Internal("failed to encrypt gateway secret", err)
	}

	var encryptedWebhookToken *string
	if req.WebhookToken != "" {
		enc, err := encrypt.Encrypt(s.cfg.GatewayEncryptionKey, req.WebhookToken)
		if err != nil {
			return nil, apperror.Internal("failed to encrypt webhook token", err)
		}
		encryptedWebhookToken = &enc
	}

	var publicKey *string
	if req.PublicKey != "" {
		publicKey = &req.PublicKey
	}

	now := time.Now()
	var cfg domain.PaymentGatewayConfig

	err = q.QueryRow(ctx, `
		INSERT INTO payment_gateway_configs
			(tenant_id, gateway, is_enabled, secret_key_encrypted, public_key,
			 webhook_token_encrypted, enabled_types, created_at, updated_at)
		VALUES ($1, 'xendit', $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (tenant_id, gateway) DO UPDATE SET
			is_enabled = EXCLUDED.is_enabled,
			secret_key_encrypted = EXCLUDED.secret_key_encrypted,
			public_key = EXCLUDED.public_key,
			webhook_token_encrypted = EXCLUDED.webhook_token_encrypted,
			enabled_types = EXCLUDED.enabled_types,
			updated_at = EXCLUDED.updated_at
		RETURNING id, tenant_id, gateway, is_enabled,
		          COALESCE(secret_key_encrypted, ''), public_key,
		          COALESCE(webhook_token_encrypted, ''), enabled_types, key_version,
		          created_at, updated_at
	`, tenantID, req.IsEnabled, encryptedSecret, publicKey,
		encryptedWebhookToken, req.EnabledTypes, now,
	).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Gateway, &cfg.IsEnabled,
		&cfg.SecretKeyEncrypted, &cfg.PublicKey,
		&cfg.WebhookTokenEncrypted, &cfg.EnabledTypes, &cfg.KeyVersion,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, apperror.Internal("failed to upsert gateway config", err)
	}

	return toGatewayConfigResponse(&cfg), nil
}

// SetEnabled toggles is_enabled for a tenant's gateway config.
func (s *GatewayService) SetEnabled(ctx context.Context, tenantID uuid.UUID, enabled bool) (*domain.GatewayConfigResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var cfg domain.PaymentGatewayConfig
	err := q.QueryRow(ctx, `
		UPDATE payment_gateway_configs
		SET is_enabled = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND gateway = 'xendit'
		RETURNING id, tenant_id, gateway, is_enabled,
		          COALESCE(secret_key_encrypted, ''), public_key,
		          COALESCE(webhook_token_encrypted, ''), enabled_types, key_version,
		          created_at, updated_at
	`, enabled, tenantID).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Gateway, &cfg.IsEnabled,
		&cfg.SecretKeyEncrypted, &cfg.PublicKey,
		&cfg.WebhookTokenEncrypted, &cfg.EnabledTypes, &cfg.KeyVersion,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("gateway config not found")
		}
		return nil, apperror.Internal("failed to update gateway config", err)
	}

	return toGatewayConfigResponse(&cfg), nil
}

// CreateGatewayPayment initiates a new gateway payment record.
// Phase 2: creates the DB record only. Phase 3 will add actual Xendit API calls.
// Uses INSERT ON CONFLICT to handle concurrent requests atomically.
// If a previous attempt is EXPIRED/FAILED, it resets the record for retry.
func (s *GatewayService) CreateGatewayPayment(ctx context.Context, tenantID uuid.UUID, req domain.CreateGatewayPaymentRequest) (*domain.GatewayPaymentResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	// Verify gateway config exists and is enabled
	var cfgEnabled bool
	err := q.QueryRow(ctx, `
		SELECT is_enabled FROM payment_gateway_configs
		WHERE tenant_id = $1 AND gateway = 'xendit'
	`, tenantID).Scan(&cfgEnabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.Validation("payment gateway not configured for this tenant")
		}
		return nil, apperror.Internal("failed to check gateway config", err)
	}
	if !cfgEnabled {
		return nil, apperror.Validation("payment gateway is disabled")
	}

	txID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		return nil, apperror.Validation("invalid transaction_id")
	}
	paymentItemID, err := uuid.Parse(req.PaymentItemID)
	if err != nil {
		return nil, apperror.Validation("invalid payment_item_id")
	}

	// Verify the transaction exists (RLS enforces tenant scope)
	var exists bool
	err = q.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM transactions WHERE id = $1)
	`, txID).Scan(&exists)
	if err != nil {
		return nil, apperror.Internal("failed to verify transaction", err)
	}
	if !exists {
		return nil, apperror.NotFound("transaction not found")
	}

	// Generate idempotent external_id
	externalID := fmt.Sprintf("lpos-%s", paymentItemID.String())
	now := time.Now()

	// Atomic upsert: insert new record, or if existing is EXPIRED/FAILED reset it for retry.
	// If existing is PENDING/ACTIVE/PAID, return it as-is (true idempotency).
	var resp domain.GatewayPaymentResponse
	err = q.QueryRow(ctx, `
		INSERT INTO transaction_gateway_payments
			(tenant_id, transaction_id, payment_item_id, gateway, gateway_type,
			 external_id, amount, gateway_status, created_at, updated_at)
		VALUES ($1, $2, $3, 'xendit', $4, $5, $6, 'PENDING', $7, $7)
		ON CONFLICT (external_id) DO UPDATE SET
			gateway_status = CASE
				WHEN transaction_gateway_payments.gateway_status IN ('EXPIRED', 'FAILED')
				THEN 'PENDING'
				ELSE transaction_gateway_payments.gateway_status
			END,
			gateway_payment_url = CASE
				WHEN transaction_gateway_payments.gateway_status IN ('EXPIRED', 'FAILED')
				THEN NULL
				ELSE transaction_gateway_payments.gateway_payment_url
			END,
			gateway_ref_id = CASE
				WHEN transaction_gateway_payments.gateway_status IN ('EXPIRED', 'FAILED')
				THEN NULL
				ELSE transaction_gateway_payments.gateway_ref_id
			END,
			expires_at = CASE
				WHEN transaction_gateway_payments.gateway_status IN ('EXPIRED', 'FAILED')
				THEN NULL
				ELSE transaction_gateway_payments.expires_at
			END,
			updated_at = $7
		RETURNING id, external_id, gateway_status, gateway_type, amount,
		          gateway_payment_url, qr_string, expires_at, created_at
	`, tenantID, txID, paymentItemID, req.GatewayType, externalID, req.Amount, now,
	).Scan(
		&resp.ID, &resp.ExternalID, &resp.GatewayStatus, &resp.GatewayType,
		&resp.Amount, &resp.GatewayPaymentURL, &resp.QRString, &resp.ExpiresAt, &resp.CreatedAt,
	)
	if err != nil {
		return nil, apperror.Internal("failed to create gateway payment", err)
	}

	// If status is PENDING (new or reset from EXPIRED/FAILED), call Xendit API.
	// Use QR Codes API for QRIS (returns qr_string), Invoice API for others.
	if resp.GatewayStatus == "PENDING" {
		secretKey, err := s.getDecryptedSecretKey(ctx, tenantID)
		if err != nil {
			return nil, err
		}

		var gatewayRefID string
		var gatewayPaymentURL *string
		var qrString *string
		var expiresAt *time.Time
		var rawJSON []byte

		if req.GatewayType == "qris" {
			// Use QR Codes API for QRIS — returns qr_string for direct QR rendering
			if s.cfg.XenditQRCallbackURL == "" {
				return nil, apperror.Internal("XENDIT_QR_CALLBACK_URL not configured", nil)
			}
			qrResp, err := s.callXenditCreateQRCode(secretKey, xenditQRCodeRequest{
				ExternalID:  externalID,
				Amount:      req.Amount,
				CallbackURL: s.cfg.XenditQRCallbackURL,
			})
			if err != nil {
				slog.Error("xendit create QR code failed",
					"external_id", externalID, "error", err)
				return nil, apperror.Internal("failed to create payment with gateway", err)
			}
			gatewayRefID = qrResp.ID
			qrString = &qrResp.QRString
			rawJSON = qrResp.RawJSON
			// QR Codes API doesn't return expiry; use configured invoice duration
			expiry := time.Now().Add(time.Duration(s.cfg.XenditInvoiceDuration) * time.Second)
			expiresAt = &expiry
		} else {
			// Use Invoice API for bank_transfer, ewallet
			invoiceResp, err := s.callXenditCreateInvoice(secretKey, xenditInvoiceRequest{
				ExternalID:     externalID,
				Amount:         req.Amount,
				PaymentMethods: mapGatewayTypeToPaymentMethods(req.GatewayType),
			})
			if err != nil {
				slog.Error("xendit create invoice failed",
					"external_id", externalID, "error", err)
				return nil, apperror.Internal("failed to create payment with gateway", err)
			}
			gatewayRefID = invoiceResp.ID
			gatewayPaymentURL = &invoiceResp.InvoiceURL
			expiresAt = &invoiceResp.ExpiryDate
			rawJSON = invoiceResp.RawJSON

			// Parse VA details from Xendit invoice response for inline display
			if len(invoiceResp.AvailableBanks) > 0 {
				for _, bank := range invoiceResp.AvailableBanks {
					if bank.BankAccountNumber != "" {
						resp.VirtualAccounts = append(resp.VirtualAccounts, domain.VirtualAccountDetail{
							BankCode:      bank.BankCode,
							AccountNumber: bank.BankAccountNumber,
							BankName:      bankCodeToName(bank.BankCode),
						})
					}
				}
			}
		}

		// Update DB with Xendit response: set ACTIVE, store URL/QR + ref + expiry.
		// Only update if still PENDING (webhook may have already set PAID).
		err = q.QueryRow(ctx, `
			UPDATE transaction_gateway_payments
			SET gateway_status = 'ACTIVE',
			    gateway_ref_id = $1,
			    gateway_payment_url = $2,
			    qr_string = $3,
			    expires_at = $4,
			    gateway_response = $5,
			    updated_at = $6
			WHERE external_id = $7 AND gateway_status = 'PENDING'
			RETURNING id, external_id, gateway_status, gateway_type, amount,
			          gateway_payment_url, qr_string, expires_at, created_at
		`, gatewayRefID, gatewayPaymentURL, qrString, expiresAt,
			rawJSON, time.Now(), externalID,
		).Scan(
			&resp.ID, &resp.ExternalID, &resp.GatewayStatus, &resp.GatewayType,
			&resp.Amount, &resp.GatewayPaymentURL, &resp.QRString, &resp.ExpiresAt, &resp.CreatedAt,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				// Webhook already updated status — re-read current state
				err = q.QueryRow(ctx, `
					SELECT id, external_id, gateway_status, gateway_type, amount,
					       gateway_payment_url, qr_string, expires_at, created_at
					FROM transaction_gateway_payments
					WHERE external_id = $1
				`, externalID).Scan(
					&resp.ID, &resp.ExternalID, &resp.GatewayStatus, &resp.GatewayType,
					&resp.Amount, &resp.GatewayPaymentURL, &resp.QRString, &resp.ExpiresAt, &resp.CreatedAt,
				)
				if err != nil {
					return nil, apperror.Internal("failed to re-read gateway payment", err)
				}
			} else {
				slog.Error("failed to update payment after xendit call",
					"external_id", externalID, "error", err)
				return nil, apperror.Internal("failed to update gateway payment", err)
			}
		}
	}

	return &resp, nil
}

// GetPaymentStatus returns the current status of a gateway payment.
func (s *GatewayService) GetPaymentStatus(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.GatewayPaymentStatusResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var resp domain.GatewayPaymentStatusResponse
	err := q.QueryRow(ctx, `
		SELECT external_id, gateway_status, paid_at
		FROM transaction_gateway_payments
		WHERE external_id = $1 AND tenant_id = $2
	`, externalID, tenantID).Scan(
		&resp.ExternalID, &resp.GatewayStatus, &resp.PaidAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("gateway payment not found")
		}
		return nil, apperror.Internal("failed to get gateway payment status", err)
	}

	resp.IsFinal = isFinalStatus(resp.GatewayStatus)
	return &resp, nil
}

// ListPayments returns a paginated list of gateway payments for a tenant.
func (s *GatewayService) ListPayments(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]domain.GatewayPaymentListItem, int, error) {
	// Defensive pagination bounds
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Validate status filter
	if status != "" {
		validStatuses := map[string]bool{"PENDING": true, "ACTIVE": true, "PAID": true, "EXPIRED": true, "FAILED": true, "CANCELLED": true}
		if !validStatuses[status] {
			return nil, 0, apperror.Validation("invalid status filter")
		}
	}

	q := middleware.GetQuerier(ctx, s.db)

	// Count total
	countQuery := `SELECT COUNT(*) FROM transaction_gateway_payments WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if status != "" {
		countQuery += fmt.Sprintf(` AND gateway_status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}

	var total int
	if err := q.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Internal("failed to count gateway payments", err)
	}

	if total == 0 {
		return []domain.GatewayPaymentListItem{}, 0, nil
	}

	// Fetch page
	dataQuery := `
		SELECT id, external_id, transaction_id, gateway_type, amount,
		       gateway_status, paid_at, expires_at, created_at
		FROM transaction_gateway_payments
		WHERE tenant_id = $1`

	dataArgs := []interface{}{tenantID}
	dataArgIdx := 2

	if status != "" {
		dataQuery += fmt.Sprintf(` AND gateway_status = $%d`, dataArgIdx)
		dataArgs = append(dataArgs, status)
		dataArgIdx++
	}

	dataQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, dataArgIdx, dataArgIdx+1)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := q.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list gateway payments", err)
	}
	defer rows.Close()

	var items []domain.GatewayPaymentListItem
	for rows.Next() {
		var item domain.GatewayPaymentListItem
		if err := rows.Scan(
			&item.ID, &item.ExternalID, &item.TransactionID, &item.GatewayType,
			&item.Amount, &item.GatewayStatus, &item.PaidAt, &item.ExpiresAt, &item.CreatedAt,
		); err != nil {
			return nil, 0, apperror.Internal("failed to scan gateway payment", err)
		}
		items = append(items, item)
	}

	if items == nil {
		items = []domain.GatewayPaymentListItem{}
	}

	return items, total, nil
}

// ProcessWebhook validates and processes an inbound Xendit webhook.
// This runs without RLS (public endpoint), so we use the pool directly.
// Always returns nil (HTTP 200) to Xendit on success OR on non-auth errors
// to prevent infinite retries. Only returns error for auth failures.
func (s *GatewayService) ProcessWebhook(ctx context.Context, callbackToken string, payload []byte) error {
	// Validate callback token against global token
	if s.cfg.XenditWebhookToken == "" {
		slog.Error("xendit webhook token not configured, rejecting webhook")
		return apperror.Unauthorized("webhook not configured")
	}
	if subtle.ConstantTimeCompare([]byte(callbackToken), []byte(s.cfg.XenditWebhookToken)) != 1 {
		return apperror.Unauthorized("invalid callback token")
	}

	// Parse the webhook payload — Xendit uses different formats:
	// Invoice: { external_id, status, paid_at }
	// QR Code: { event: "qr.payment", status: "COMPLETED", qr_code: { external_id } }
	var webhookData struct {
		Event       string `json:"event"`
		ExternalID  string `json:"external_id"`
		ReferenceID string `json:"reference_id"`
		Status      string `json:"status"`
		PaidAt      string `json:"paid_at"`
		QRCode      *struct {
			ExternalID string `json:"external_id"`
		} `json:"qr_code"`
	}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		slog.Error("failed to parse webhook payload", "error", err)
		return nil // Return nil = HTTP 200 to prevent Xendit retries
	}

	// Resolve external_id from the appropriate location:
	// 1. QR Code webhook: qr_code.external_id
	// 2. Invoice webhook: top-level external_id
	// 3. Fallback: reference_id
	resolvedExternalID := webhookData.ExternalID
	if webhookData.QRCode != nil && webhookData.QRCode.ExternalID != "" {
		resolvedExternalID = webhookData.QRCode.ExternalID
	}
	if resolvedExternalID == "" {
		resolvedExternalID = webhookData.ReferenceID
	}
	if resolvedExternalID == "" {
		slog.Warn("webhook missing external_id and reference_id", "payload_size", len(payload))
		return nil // Return nil = HTTP 200
	}

	// Validate status field
	if webhookData.Status == "" {
		slog.Warn("webhook missing status field", "external_id", resolvedExternalID)
		return nil
	}

	// Only process our own payments (prefixed with "lpos-")
	if !strings.HasPrefix(resolvedExternalID, "lpos-") {
		slog.Debug("ignoring webhook for non-lpos external_id", "external_id", resolvedExternalID)
		return nil
	}

	// Map Xendit status to our gateway_status
	gatewayStatus := mapXenditStatus(webhookData.Status)

	// Parse paid_at from Xendit if available, otherwise use server time
	var paidAt *time.Time
	if gatewayStatus == "PAID" {
		if t, err := time.Parse(time.RFC3339, webhookData.PaidAt); err == nil {
			paidAt = &t
		} else {
			now := time.Now()
			paidAt = &now
		}
	}

	// Use pool directly (no RLS context in webhook path).
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		slog.Error("webhook: failed to acquire connection",
			"external_id", resolvedExternalID, "error", err)
		return nil // Return nil = HTTP 200, will retry via Xendit
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		slog.Error("webhook: failed to begin transaction",
			"external_id", resolvedExternalID, "error", err)
		return nil
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("webhook: failed to rollback tx",
				"external_id", resolvedExternalID, "error", err)
		}
	}()

	// Set superadmin role to bypass RLS
	if _, err := tx.Exec(ctx, `SET LOCAL "app.current_role" = 'superadmin'`); err != nil {
		slog.Error("webhook: failed to set session role",
			"external_id", resolvedExternalID, "error", err)
		return nil
	}

	now := time.Now()
	tag, err := tx.Exec(ctx, `
		UPDATE transaction_gateway_payments
		SET gateway_status = $1, paid_at = COALESCE($2, paid_at),
		    webhook_payload = $3, updated_at = $4
		WHERE external_id = $5
		  AND gateway_status NOT IN ('PAID', 'EXPIRED', 'FAILED', 'CANCELLED')
	`, gatewayStatus, paidAt, payload, now, resolvedExternalID)
	if err != nil {
		slog.Error("webhook: failed to update gateway payment",
			"external_id", resolvedExternalID, "error", err)
		return nil
	}

	if tag.RowsAffected() == 0 {
		slog.Debug("webhook: no rows updated (already finalized or not found)",
			"external_id", resolvedExternalID, "status", gatewayStatus)
	} else {
		slog.Info("webhook: payment status updated",
			"external_id", resolvedExternalID, "status", gatewayStatus)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("webhook: failed to commit",
			"external_id", resolvedExternalID, "error", err)
		return nil
	}

	return nil
}

// --- Xendit API helpers ---

// getDecryptedSecretKey retrieves and decrypts the tenant's Xendit secret key.
func (s *GatewayService) getDecryptedSecretKey(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if s.cfg.GatewayEncryptionKey == "" {
		return "", apperror.Internal("gateway encryption key not configured", nil)
	}

	q := middleware.GetQuerier(ctx, s.db)
	var encrypted string
	err := q.QueryRow(ctx, `
		SELECT COALESCE(secret_key_encrypted, '')
		FROM payment_gateway_configs
		WHERE tenant_id = $1 AND gateway = 'xendit'
	`, tenantID).Scan(&encrypted)
	if err != nil {
		return "", apperror.Internal("failed to get gateway secret key", err)
	}
	if encrypted == "" {
		return "", apperror.Validation("gateway secret key not configured")
	}

	plaintext, err := encrypt.Decrypt(s.cfg.GatewayEncryptionKey, encrypted)
	if err != nil {
		return "", apperror.Internal("failed to decrypt gateway secret key", err)
	}
	return plaintext, nil
}

type xenditInvoiceRequest struct {
	ExternalID     string
	Amount         int64
	PaymentMethods []string
}

type xenditInvoiceResponse struct {
	ID             string                    `json:"id"`
	InvoiceURL     string                    `json:"invoice_url"`
	Status         string                    `json:"status"`
	ExpiryDate     time.Time                 `json:"expiry_date"`
	AvailableBanks []xenditAvailableBank     `json:"available_banks"`
	RawJSON        []byte                    // Store the full response for auditing
}

type xenditAvailableBank struct {
	BankCode          string `json:"bank_code"`
	CollectionType    string `json:"collection_type"`
	BankAccountNumber string `json:"bank_account_number"`
	AccountHolderName string `json:"account_holder_name"`
}

// callXenditCreateInvoice calls POST /v2/invoices on the Xendit API.
func (s *GatewayService) callXenditCreateInvoice(secretKey string, req xenditInvoiceRequest) (*xenditInvoiceResponse, error) {
	body := map[string]interface{}{
		"external_id":      req.ExternalID,
		"amount":           req.Amount,
		"currency":         s.cfg.XenditCurrency,
		"payment_methods":  req.PaymentMethods,
		"description":      s.cfg.XenditDescription,
		"invoice_duration": s.cfg.XenditInvoiceDuration,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.cfg.XenditAPIURL+"/v2/invoices", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Xendit uses HTTP Basic Auth: base64(secretKey + ":")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(secretKey+":")))
	httpReq.Header.Set("Idempotency-Key", req.ExternalID)

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		var xenditErr struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		if jsonErr := json.Unmarshal(respBody, &xenditErr); jsonErr == nil && xenditErr.ErrorCode != "" {
			slog.Error("xendit API error",
				"status", httpResp.StatusCode,
				"error_code", xenditErr.ErrorCode,
				"message", xenditErr.Message,
				"external_id", req.ExternalID)
			return nil, fmt.Errorf("xendit: %s - %s", xenditErr.ErrorCode, xenditErr.Message)
		}
		slog.Error("xendit API error",
			"status", httpResp.StatusCode,
			"external_id", req.ExternalID)
		return nil, fmt.Errorf("xendit API returned %d", httpResp.StatusCode)
	}

	var result xenditInvoiceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	result.RawJSON = respBody

	return &result, nil
}

type xenditQRCodeRequest struct {
	ExternalID  string
	Amount      int64
	CallbackURL string
}

type xenditQRCodeResponse struct {
	ID       string `json:"id"`
	QRString string `json:"qr_string"`
	Status   string `json:"status"`
	RawJSON  []byte
}

// callXenditCreateQRCode calls POST /qr_codes on the Xendit API.
// Returns a qr_string that can be rendered directly as a QRIS QR code.
func (s *GatewayService) callXenditCreateQRCode(secretKey string, req xenditQRCodeRequest) (*xenditQRCodeResponse, error) {
	body := map[string]interface{}{
		"external_id":  req.ExternalID,
		"type":         "DYNAMIC",
		"callback_url": req.CallbackURL,
		"amount":       req.Amount,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.cfg.XenditAPIURL+"/qr_codes", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(secretKey+":")))
	httpReq.Header.Set("Idempotency-Key", req.ExternalID)

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		var xenditErr struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		if jsonErr := json.Unmarshal(respBody, &xenditErr); jsonErr == nil && xenditErr.ErrorCode != "" {
			slog.Error("xendit QR API error",
				"status", httpResp.StatusCode,
				"error_code", xenditErr.ErrorCode,
				"message", xenditErr.Message,
				"external_id", req.ExternalID)
			return nil, fmt.Errorf("xendit: %s - %s", xenditErr.ErrorCode, xenditErr.Message)
		}
		return nil, fmt.Errorf("xendit QR API returned %d", httpResp.StatusCode)
	}

	var result xenditQRCodeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	result.RawJSON = respBody

	return &result, nil
}

// mapGatewayTypeToPaymentMethods maps our internal gateway type to Xendit payment_methods.
func mapGatewayTypeToPaymentMethods(gatewayType string) []string {
	switch gatewayType {
	case "qris":
		return []string{"QRIS"}
	case "virtual_account":
		return []string{"BCA", "BNI", "MANDIRI", "PERMATA", "BRI"}
	case "ewallet":
		return []string{"OVO", "DANA", "SHOPEEPAY", "LINKAJA"}
	default:
		return []string{"QRIS"} // Fallback to QRIS
	}
}

func toGatewayConfigResponse(cfg *domain.PaymentGatewayConfig) *domain.GatewayConfigResponse {
	enabledTypes := cfg.EnabledTypes
	if enabledTypes == nil {
		enabledTypes = []string{}
	}
	return &domain.GatewayConfigResponse{
		ID:           cfg.ID,
		Gateway:      cfg.Gateway,
		IsEnabled:    cfg.IsEnabled,
		HasSecretKey: cfg.SecretKeyEncrypted != "",
		PublicKey:    cfg.PublicKey,
		EnabledTypes: enabledTypes,
		CreatedAt:    cfg.CreatedAt,
		UpdatedAt:    cfg.UpdatedAt,
	}
}

func isFinalStatus(status string) bool {
	switch status {
	case "PAID", "EXPIRED", "FAILED", "CANCELLED":
		return true
	default:
		return false
	}
}

func bankCodeToName(code string) string {
	names := map[string]string{
		"BCA":     "Bank BCA",
		"BNI":     "Bank BNI",
		"BRI":     "Bank BRI",
		"MANDIRI": "Bank Mandiri",
		"PERMATA": "Bank Permata",
		"BSI":     "Bank BSI",
		"CIMB":    "Bank CIMB Niaga",
		"SAHABAT_SAMPOERNA": "Bank Sahabat Sampoerna",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

func mapXenditStatus(xenditStatus string) string {
	switch xenditStatus {
	case "COMPLETED", "PAID", "SUCCEEDED":
		return "PAID"
	case "PENDING":
		return "PENDING"
	case "ACTIVE":
		return "ACTIVE"
	case "EXPIRED":
		return "EXPIRED"
	case "FAILED":
		return "FAILED"
	case "CANCELLED":
		return "CANCELLED"
	default:
		slog.Warn("unknown xendit status, defaulting to PENDING", "status", xenditStatus)
		return "PENDING"
	}
}
