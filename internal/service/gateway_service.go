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
		SELECT id, tenant_id, gateway, is_enabled, secret_key_encrypted,
		       public_key, webhook_token_encrypted, enabled_types, key_version,
		       created_at, updated_at
		FROM payment_gateway_configs
		WHERE tenant_id = $1 AND gateway = 'xendit'
	`, tenantID).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Gateway, &cfg.IsEnabled,
		&cfg.SecretKeyEncrypted, &cfg.PublicKey, &cfg.WebhookTokenEncrypted,
		&cfg.EnabledTypes, &cfg.KeyVersion, &cfg.CreatedAt, &cfg.UpdatedAt,
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
		RETURNING id, tenant_id, gateway, is_enabled, secret_key_encrypted,
		          public_key, webhook_token_encrypted, enabled_types, key_version,
		          created_at, updated_at
	`, tenantID, req.IsEnabled, encryptedSecret, publicKey,
		encryptedWebhookToken, req.EnabledTypes, now,
	).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Gateway, &cfg.IsEnabled,
		&cfg.SecretKeyEncrypted, &cfg.PublicKey, &cfg.WebhookTokenEncrypted,
		&cfg.EnabledTypes, &cfg.KeyVersion, &cfg.CreatedAt, &cfg.UpdatedAt,
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
		RETURNING id, tenant_id, gateway, is_enabled, secret_key_encrypted,
		          public_key, webhook_token_encrypted, enabled_types, key_version,
		          created_at, updated_at
	`, enabled, tenantID).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Gateway, &cfg.IsEnabled,
		&cfg.SecretKeyEncrypted, &cfg.PublicKey, &cfg.WebhookTokenEncrypted,
		&cfg.EnabledTypes, &cfg.KeyVersion, &cfg.CreatedAt, &cfg.UpdatedAt,
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

	// Verify the transaction exists and belongs to this tenant
	var exists bool
	err = q.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM transactions WHERE id = $1 AND tenant_id = $2)
	`, txID, tenantID).Scan(&exists)
	if err != nil {
		return nil, apperror.Internal("failed to verify transaction", err)
	}
	if !exists {
		return nil, apperror.NotFound("transaction not found")
	}

	// Generate idempotent external_id
	externalID := fmt.Sprintf("lpos-%s", paymentItemID.String())

	// Check for existing active/pending payment (idempotency)
	var existingResp domain.GatewayPaymentResponse
	err = q.QueryRow(ctx, `
		SELECT id, external_id, gateway_status, gateway_type, amount,
		       gateway_payment_url, expires_at, created_at
		FROM transaction_gateway_payments
		WHERE external_id = $1
	`, externalID).Scan(
		&existingResp.ID, &existingResp.ExternalID, &existingResp.GatewayStatus,
		&existingResp.GatewayType, &existingResp.Amount, &existingResp.GatewayPaymentURL,
		&existingResp.ExpiresAt, &existingResp.CreatedAt,
	)
	if err == nil {
		// Already exists — return existing record
		return &existingResp, nil
	}
	if err != pgx.ErrNoRows {
		return nil, apperror.Internal("failed to check existing gateway payment", err)
	}

	// Insert new gateway payment record with PENDING status
	now := time.Now()
	var resp domain.GatewayPaymentResponse
	err = q.QueryRow(ctx, `
		INSERT INTO transaction_gateway_payments
			(tenant_id, transaction_id, payment_item_id, gateway, gateway_type,
			 external_id, amount, gateway_status, created_at, updated_at)
		VALUES ($1, $2, $3, 'xendit', $4, $5, $6, 'PENDING', $7, $7)
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

	// TODO Phase 3: Call Xendit API here, update gateway_ref_id, gateway_payment_url, expires_at

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

// ProcessWebhook validates and processes an inbound Xendit webhook.
// This runs without RLS (public endpoint), so we use the pool directly.
func (s *GatewayService) ProcessWebhook(ctx context.Context, callbackToken string, payload []byte) error {
	// Validate callback token
	if s.cfg.XenditWebhookToken == "" {
		return apperror.Internal("xendit webhook token not configured", nil)
	}
	if callbackToken != s.cfg.XenditWebhookToken {
		return apperror.Unauthorized("invalid callback token")
	}

	// Parse the webhook payload to extract external_id and status
	var webhookData struct {
		ExternalID string `json:"external_id"`
		Status     string `json:"status"`
		PaidAt     string `json:"paid_at"`
	}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		return apperror.Validation("invalid webhook payload")
	}

	if webhookData.ExternalID == "" {
		return apperror.Validation("missing external_id in webhook payload")
	}

	// Map Xendit status to our gateway_status
	gatewayStatus := mapXenditStatus(webhookData.Status)

	// Use pool directly (no RLS context in webhook path).
	// Acquire connection and set superadmin role for RLS bypass.
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return apperror.Internal("failed to acquire connection", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return apperror.Internal("failed to begin transaction", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("failed to rollback webhook tx", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, `SET LOCAL "app.current_role" = 'superadmin'`); err != nil {
		return apperror.Internal("failed to set session role", err)
	}

	// Update the gateway payment record
	now := time.Now()
	var paidAt *time.Time
	if gatewayStatus == "PAID" {
		paidAt = &now
	}

	tag, err := tx.Exec(ctx, `
		UPDATE transaction_gateway_payments
		SET gateway_status = $1, paid_at = COALESCE($2, paid_at),
		    webhook_payload = $3, updated_at = $4
		WHERE external_id = $5
		  AND gateway_status NOT IN ('PAID', 'CANCELLED')
	`, gatewayStatus, paidAt, payload, now, webhookData.ExternalID)
	if err != nil {
		return apperror.Internal("failed to update gateway payment", err)
	}

	if tag.RowsAffected() == 0 {
		slog.Warn("webhook for already finalized payment", "external_id", webhookData.ExternalID)
	}

	if err := tx.Commit(ctx); err != nil {
		return apperror.Internal("failed to commit webhook update", err)
	}

	return nil
}

func toGatewayConfigResponse(cfg *domain.PaymentGatewayConfig) *domain.GatewayConfigResponse {
	return &domain.GatewayConfigResponse{
		ID:           cfg.ID,
		Gateway:      cfg.Gateway,
		IsEnabled:    cfg.IsEnabled,
		HasSecretKey: cfg.SecretKeyEncrypted != "",
		PublicKey:    cfg.PublicKey,
		EnabledTypes: cfg.EnabledTypes,
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
	default:
		return "PENDING"
	}
}
