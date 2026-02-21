package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type MemberService struct {
	db *pgxpool.Pool
}

func NewMemberService(db *pgxpool.Pool) *MemberService {
	return &MemberService{db: db}
}

func (s *MemberService) List(ctx context.Context, tenantID *uuid.UUID, params pagination.Params) ([]domain.Member, int, error) {
	var total int
	countQuery := "SELECT COUNT(*) FROM members"
	listQuery := `SELECT id, tenant_id, name, phone, email, tier, discount_percent, total_points, created_at FROM members`
	var args []any

	if tenantID != nil {
		countQuery += " WHERE tenant_id = $1 OR tenant_id IS NULL"
		listQuery += " WHERE tenant_id = $1 OR tenant_id IS NULL"
		args = append(args, *tenantID)
	}

	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		slog.Error("failed to count members", "error", err)
		return nil, 0, apperror.Internal("failed to count members", err)
	}

	argIdx := len(args) + 1
	listQuery += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := s.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list members", err)
	}
	defer rows.Close()

	var members []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.CreatedAt); err != nil {
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
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM members WHERE phone = $1)", req.Phone).Scan(&exists)
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

	m := domain.Member{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            req.Name,
		Phone:           req.Phone,
		Email:           req.Email,
		Tier:            tier,
		DiscountPercent: req.DiscountPercent,
		TotalPoints:     0,
		CreatedAt:       time.Now(),
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO members (id, tenant_id, name, phone, email, tier, discount_percent, total_points, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		m.ID, m.TenantID, m.Name, m.Phone, m.Email, m.Tier, m.DiscountPercent, m.TotalPoints, m.CreatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create member", err)
	}

	return &m, nil
}

func (s *MemberService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateMemberRequest) (*domain.Member, error) {
	var m domain.Member
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, phone, email, tier, discount_percent, total_points, created_at
		 FROM members WHERE id = $1`, id).
		Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.CreatedAt)
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

	_, err = s.db.Exec(ctx,
		`UPDATE members SET name=$1, phone=$2, email=$3, tier=$4, discount_percent=$5 WHERE id=$6`,
		m.Name, m.Phone, m.Email, m.Tier, m.DiscountPercent, m.ID)
	if err != nil {
		return nil, apperror.Internal("failed to update member", err)
	}

	return &m, nil
}

func (s *MemberService) LookupByPhone(ctx context.Context, phone string) (*domain.Member, error) {
	var m domain.Member
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, phone, email, tier, discount_percent, total_points, created_at
		 FROM members WHERE phone = $1`, phone).
		Scan(&m.ID, &m.TenantID, &m.Name, &m.Phone, &m.Email, &m.Tier,
			&m.DiscountPercent, &m.TotalPoints, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.NotFound("member not found")
		}
		return nil, apperror.Internal("failed to lookup member", err)
	}
	return &m, nil
}

func (s *MemberService) GetTransactionsByTenant(ctx context.Context, tenantID uuid.UUID, params pagination.Params) ([]domain.Transaction, int, error) {
	var total int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count transactions", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, outlet_id, local_order_number, customer_name, items, subtotal,
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
			&t.Items, &t.Subtotal, &t.DiscountAmount, &t.TaxAmount, &t.TotalAmount,
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
