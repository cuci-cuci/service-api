package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

const memberSelectCols = `id, tenant_id, name, phone, email, tier, discount_percent, total_points, total_spending, referral_code, referred_by_member_id, has_first_transaction, created_at`

type MemberService struct {
	db *pgxpool.Pool
}

func NewMemberService(db *pgxpool.Pool) *MemberService {
	return &MemberService{db: db}
}

func scanMember(row interface{ Scan(dest ...any) error }) (domain.Member, error) {
	var m domain.Member
	err := row.Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
		&m.DiscountPercent, &m.TotalPoints, &m.TotalSpending,
		&m.ReferralCode, &m.ReferredByMemberID, &m.HasFirstTransaction, &m.CreatedAt)
	return m, err
}

func (s *MemberService) List(ctx context.Context, tenantID *uuid.UUID, params pagination.Params) ([]domain.Member, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	countQuery := "SELECT COUNT(*) FROM members"
	listQuery := `SELECT ` + memberSelectCols + ` FROM members`
	var args []any

	if tenantID != nil {
		countQuery += " WHERE tenant_id = $1 OR tenant_id IS NULL"
		listQuery += " WHERE tenant_id = $1 OR tenant_id IS NULL"
		args = append(args, *tenantID)
	}

	err := q.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		slog.Error("failed to count members", "error", err)
		return nil, 0, apperror.Internal("failed to count members", err)
	}

	argIdx := len(args) + 1
	listQuery += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := q.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list members", err)
	}
	defer rows.Close()

	var members []domain.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, 0, apperror.Internal("failed to scan member", err)
		}
		members = append(members, m)
	}

	if members == nil {
		members = []domain.Member{}
	}

	return members, total, nil
}

func (s *MemberService) Create(ctx context.Context, tenantID *uuid.UUID, req domain.CreateMemberRequest) (*domain.Member, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var exists bool
	err := q.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM members WHERE phone = $1)", req.Phone).Scan(&exists)
	if err != nil {
		return nil, apperror.Internal("failed to check phone", err)
	}
	if exists {
		return nil, apperror.Conflict("member with this phone already exists")
	}

	tier := req.Tier
	if tier == "" {
		tier = "bronze"
	}

	referralCode := generateReferralCode()

	// Resolve referrer
	var referredByMemberID *uuid.UUID
	if req.ReferralCode != "" {
		var referrerID uuid.UUID
		err := q.QueryRow(ctx,
			`SELECT id FROM members WHERE referral_code = $1`,
			strings.ToUpper(strings.TrimSpace(req.ReferralCode))).Scan(&referrerID)
		if err == nil {
			referredByMemberID = &referrerID
		}
		// Silently ignore invalid referral codes
	}

	m := domain.Member{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		Name:                req.Name,
		Phone:               req.Phone,
		Email:               req.Email,
		Tier:                tier,
		DiscountPercent:     req.DiscountPercent,
		TotalPoints:         0,
		ReferralCode:        &referralCode,
		ReferredByMemberID:  referredByMemberID,
		HasFirstTransaction: false,
		CreatedAt:           time.Now(),
	}

	_, err = q.Exec(ctx,
		`INSERT INTO members (id, tenant_id, name, phone, email, tier, discount_percent, total_points, referral_code, referred_by_member_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		m.ID, m.TenantID, m.Name, m.Phone, m.Email, m.Tier, m.DiscountPercent, m.TotalPoints, m.ReferralCode, m.ReferredByMemberID, m.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create member", err)
	}

	return &m, nil
}

func (s *MemberService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateMemberRequest) (*domain.Member, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var m domain.Member
	err := q.QueryRow(ctx,
		`SELECT `+memberSelectCols+` FROM members WHERE id = $1`, id).
		Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.TotalSpending,
			&m.ReferralCode, &m.ReferredByMemberID, &m.HasFirstTransaction, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("member not found")
		}
		return nil, apperror.Internal("failed to get member", err)
	}

	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Phone != "" {
		m.Phone = req.Phone
	}
	if req.Email != "" {
		m.Email = req.Email
	}
	if req.Tier != "" {
		m.Tier = req.Tier
	}
	if req.DiscountPercent != nil {
		m.DiscountPercent = *req.DiscountPercent
	}

	_, err = q.Exec(ctx,
		`UPDATE members SET name=$1, phone=$2, email=$3, tier=$4, discount_percent=$5 WHERE id=$6`,
		m.Name, m.Phone, m.Email, m.Tier, m.DiscountPercent, m.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update member", err)
	}

	return &m, nil
}

func (s *MemberService) Delete(ctx context.Context, id uuid.UUID) error {
	q := middleware.GetQuerier(ctx, s.db)

	result, err := q.Exec(ctx, `DELETE FROM members WHERE id = $1`, id)
	if err != nil {
		return apperror.Internal("failed to delete member", err)
	}
	if result.RowsAffected() == 0 {
		return apperror.NotFound("member not found")
	}
	return nil
}

func (s *MemberService) LookupByPhone(ctx context.Context, phone string) (*domain.Member, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var m domain.Member
	err := q.QueryRow(ctx,
		`SELECT `+memberSelectCols+` FROM members WHERE phone = $1`, phone).
		Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.TotalSpending,
			&m.ReferralCode, &m.ReferredByMemberID, &m.HasFirstTransaction, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("member not found")
		}
		return nil, apperror.Internal("failed to lookup member", err)
	}
	return &m, nil
}

func (s *MemberService) GetTransactionsByTenant(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.Transaction, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count transactions", err)
	}

	rows, err := q.Query(ctx,
		`SELECT id, tenant_id, outlet_id, local_order_number, customer_name, member_id, items, subtotal,
		        discount_amount, tax_amount, total_amount, payment_status, payments, status,
		        config_version_id, notes, created_by, created_at, synced_at
		 FROM transactions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, params.PerPage, params.Offset())
	if err != nil {
		return nil, 0, apperror.Internal("failed to list transactions", err)
	}
	defer rows.Close()

	var txns []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(&t.ID, &t.TenantID, &t.OutletID, &t.LocalOrderNumber, &t.CustomerName,
			&t.MemberID, &t.Items, &t.Subtotal, &t.DiscountAmount, &t.TaxAmount, &t.TotalAmount,
			&t.PaymentStatus, &t.Payments, &t.Status, &t.ConfigVersionID, &t.Notes,
			&t.CreatedBy, &t.CreatedAt, &t.SyncedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan transaction", err)
		}
		txns = append(txns, t)
	}

	if txns == nil {
		txns = []domain.Transaction{}
	}

	return txns, total, nil
}

func (s *MemberService) UpdateSpending(ctx context.Context, memberID uuid.UUID, amount int64) (*domain.Member, error) {
	q := middleware.GetQuerier(ctx, s.db)
	var m domain.Member
	err := q.QueryRow(ctx, `
		UPDATE members SET total_spending = total_spending + $1
		WHERE id = $2
		RETURNING `+memberSelectCols+`
	`, amount, memberID).Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
		&m.DiscountPercent, &m.TotalPoints, &m.TotalSpending,
		&m.ReferralCode, &m.ReferredByMemberID, &m.HasFirstTransaction, &m.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to update member spending", err)
	}

	// Auto-upgrade tier
	newTier, newDiscount := calculateTier(m.TotalSpending)
	if newTier != m.Tier {
		_, err = q.Exec(ctx, `UPDATE members SET tier = $1, discount_percent = $2 WHERE id = $3`, newTier, newDiscount, memberID)
		if err != nil {
			return nil, apperror.Internal("failed to upgrade tier", err)
		}
		m.Tier = newTier
		m.DiscountPercent = newDiscount
	}

	return &m, nil
}

// AwardReferralPoints awards points to the referrer on the referred member's first transaction.
func (s *MemberService) AwardReferralPoints(ctx context.Context, memberID uuid.UUID) {
	q := middleware.GetQuerier(ctx, s.db)

	var referredBy *uuid.UUID
	var hasFirst bool
	err := q.QueryRow(ctx,
		`SELECT referred_by_member_id, has_first_transaction FROM members WHERE id = $1`, memberID).
		Scan(&referredBy, &hasFirst)
	if err != nil || referredBy == nil || hasFirst {
		return
	}

	// Mark first transaction
	_, err = q.Exec(ctx, `UPDATE members SET has_first_transaction = true WHERE id = $1`, memberID)
	if err != nil {
		slog.Error("failed to mark first transaction", "member_id", memberID, "error", err)
		return
	}

	// Award 100 points to referrer
	_, err = q.Exec(ctx, `UPDATE members SET total_points = total_points + 100 WHERE id = $1`, *referredBy)
	if err != nil {
		slog.Error("failed to award referral points", "referrer_id", *referredBy, "error", err)
		return
	}

	slog.Info("referral points awarded", "referrer_id", *referredBy, "referred_member_id", memberID)
}

// PointsPerThousand defines the conversion rate: 100 points = Rp 1.000
const pointsPerThousandIDR = 100

// RedeemPoints deducts points from a member and returns the discount amount.
// Conversion: 100 points = Rp 1.000
func (s *MemberService) RedeemPoints(ctx context.Context, tenantID uuid.UUID, memberID uuid.UUID, points int) (discountAmount int64, remainingPoints int, err error) {
	q := middleware.GetQuerier(ctx, s.db)

	if points <= 0 {
		return 0, 0, apperror.Validation("points must be positive")
	}

	// Ensure redeemed in multiples of 100
	if points%pointsPerThousandIDR != 0 {
		return 0, 0, apperror.Validation("points must be in multiples of 100")
	}

	var currentPoints int
	err = q.QueryRow(ctx, `SELECT total_points FROM members WHERE id = $1`, memberID).Scan(&currentPoints)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, apperror.NotFound("member not found")
		}
		return 0, 0, apperror.Internal("failed to get member points", err)
	}

	if points > currentPoints {
		return 0, 0, apperror.Validation("insufficient points")
	}

	// 100 points = Rp 1.000
	discountAmount = int64(points / pointsPerThousandIDR * 1000)

	_, err = q.Exec(ctx,
		`UPDATE members SET total_points = total_points - $1 WHERE id = $2`,
		points, memberID)
	if err != nil {
		return 0, 0, apperror.Internal("failed to deduct points", err)
	}

	// Log the redemption
	_, err = q.Exec(ctx,
		`INSERT INTO point_redemptions (tenant_id, member_id, points_redeemed, discount_amount) VALUES ($1, $2, $3, $4)`,
		tenantID, memberID, points, discountAmount)
	if err != nil {
		slog.Error("failed to log point redemption", "error", err)
		// Non-fatal: points already deducted, just log the error
	}

	remainingPoints = currentPoints - points
	return discountAmount, remainingPoints, nil
}

// AwardPoints awards loyalty points based on transaction amount.
// Rate: 1 point per Rp 1.000 spent.
func (s *MemberService) AwardPoints(ctx context.Context, memberID uuid.UUID, transactionAmount int64) (pointsEarned int, err error) {
	q := middleware.GetQuerier(ctx, s.db)

	pointsEarned = int(transactionAmount / 1000) // 1 point per Rp 1.000
	if pointsEarned <= 0 {
		return 0, nil
	}

	_, err = q.Exec(ctx,
		`UPDATE members SET total_points = total_points + $1 WHERE id = $2`,
		pointsEarned, memberID)
	if err != nil {
		return 0, apperror.Internal("failed to award points", err)
	}

	return pointsEarned, nil
}

func generateReferralCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func calculateTier(totalSpending int64) (string, int) {
	switch {
	case totalSpending >= 5_000_000: // 5M → Platinum 15%
		return "platinum", 15
	case totalSpending >= 2_000_000: // 2M → Gold 10%
		return "gold", 10
	case totalSpending >= 500_000: // 500K → Silver 5%
		return "silver", 5
	default:
		return "bronze", 0
	}
}
