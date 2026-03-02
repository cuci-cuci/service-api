package service

import (
	"bytes"
	"context"
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
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewGatewayService(db *pgxpool.Pool, cfg *config.Config) *GatewayService {
	return &GatewayService{db: db, cfg: cfg}
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

	q := middleware.GetQuerier(ctx, s.db)

	encryptedSecret, err := encrypt.Encrypt(s.cfg.GatewayEncryptionKey, req.SecretKey)
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
		          gateway_payment_url, expires_at, created_at
	`, tenantID, txID, paymentItemID, req.GatewayType, externalID, req.Amount, now,
	).Scan(
		&resp.ID, &resp.ExternalID, &resp.GatewayStatus, &resp.GatewayType,
		&resp.Amount, &resp.GatewayPaymentURL, &resp.ExpiresAt, &resp.CreatedAt,
	)
	if err != nil {
		return nil, apperror.Internal("failed to create gateway payment", err)
	}

	// If status is PENDING (new or reset from EXPIRED/FAILED), call Xendit to create invoice.
	if resp.GatewayStatus == "PENDING" {
		secretKey, err := s.getDecryptedSecretKey(ctx, tenantID)
		if err != nil {
			return nil, err
		}

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

		// Update DB with Xendit response: set ACTIVE, store URL + ref + expiry
		err = q.QueryRow(ctx, `
			UPDATE transaction_gateway_payments
			SET gateway_status = 'ACTIVE',
			    gateway_ref_id = $1,
			    gateway_payment_url = $2,
			    expires_at = $3,
			    gateway_response = $4,
			    updated_at = $5
			WHERE external_id = $6
			RETURNING id, external_id, gateway_status, gateway_type, amount,
			          gateway_payment_url, expires_at, created_at
		`, invoiceResp.ID, invoiceResp.InvoiceURL, invoiceResp.ExpiryDate,
			invoiceResp.RawJSON, time.Now(), externalID,
		).Scan(
			&resp.ID, &resp.ExternalID, &resp.GatewayStatus, &resp.GatewayType,
			&resp.Amount, &resp.GatewayPaymentURL, &resp.ExpiresAt, &resp.CreatedAt,
		)
		if err != nil {
			slog.Error("failed to update payment after xendit call",
				"external_id", externalID, "error", err)
			return nil, apperror.Internal("failed to update gateway payment", err)
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
	if callbackToken != s.cfg.XenditWebhookToken {
		return apperror.Unauthorized("invalid callback token")
	}

	// Parse the webhook payload — Xendit uses different field names:
	// Invoice: external_id, QR: reference_id, VA: external_id
	var webhookData struct {
		ExternalID  string `json:"external_id"`
		ReferenceID string `json:"reference_id"`
		Status      string `json:"status"`
		PaidAt      string `json:"paid_at"`
	}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		slog.Error("failed to parse webhook payload", "error", err)
		return nil // Return nil = HTTP 200 to prevent Xendit retries
	}

	// Use external_id, fall back to reference_id (QR Code webhooks)
	resolvedExternalID := webhookData.ExternalID
	if resolvedExternalID == "" {
		resolvedExternalID = webhookData.ReferenceID
	}
	if resolvedExternalID == "" {
		slog.Warn("webhook missing external_id and reference_id", "payload_size", len(payload))
		return nil // Return nil = HTTP 200
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
		slog.Error("webhook: failed to acquire connection", "error", err)
		return nil // Return nil = HTTP 200, will retry via Xendit
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		slog.Error("webhook: failed to begin transaction", "error", err)
		return nil
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("webhook: failed to rollback tx", "error", err)
		}
	}()

	// Set superadmin role to bypass RLS
	if _, err := tx.Exec(ctx, `SET LOCAL "app.current_role" = 'superadmin'`); err != nil {
		slog.Error("webhook: failed to set session role", "error", err)
		return nil
	}

	now := time.Now()
	tag, err := tx.Exec(ctx, `
		UPDATE transaction_gateway_payments
		SET gateway_status = $1, paid_at = COALESCE($2, paid_at),
		    webhook_payload = $3, updated_at = $4
		WHERE external_id = $5
		  AND gateway_status NOT IN ('PAID', 'CANCELLED')
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
		slog.Error("webhook: failed to commit", "error", err)
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
	ID         string    `json:"id"`
	InvoiceURL string    `json:"invoice_url"`
	Status     string    `json:"status"`
	ExpiryDate time.Time `json:"expiry_date"`
	RawJSON    []byte    // Store the full response for auditing
}

// callXenditCreateInvoice calls POST /v2/invoices on the Xendit API.
func (s *GatewayService) callXenditCreateInvoice(secretKey string, req xenditInvoiceRequest) (*xenditInvoiceResponse, error) {
	body := map[string]interface{}{
		"external_id":      req.ExternalID,
		"amount":           req.Amount,
		"currency":         "IDR",
		"payment_methods":  req.PaymentMethods,
		"description":      "LaundryPOS Payment",
		"invoice_duration": 1800, // 30 minutes
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

	client := &http.Client{Timeout: 15 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		slog.Error("xendit API error",
			"status", httpResp.StatusCode,
			"body", string(respBody))
		return nil, fmt.Errorf("xendit API returned %d", httpResp.StatusCode)
	}

	var result xenditInvoiceResponse
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
