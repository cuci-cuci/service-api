package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
)

type UserService struct {
	db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

func (s *UserService) List(ctx context.Context, params pagination.Params) ([]domain.UserResponse, int, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var total int
	err := q.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		slog.Error("failed to count users", "error", err)
		return nil, 0, apperror.Internal("failed to count users", err)
	}

	rows, err := q.Query(ctx,
		`SELECT id, email, name, role, tenant_id, is_active, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		params.PerPage, params.Offset())
	if err != nil {
		return nil, 0, apperror.Internal("failed to list users", err)
	}
	defer rows.Close()

	var users []domain.UserResponse
	for rows.Next() {
		var u domain.UserResponse
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, apperror.Internal("failed to scan user", err)
		}
		users = append(users, u)
	}

	if users == nil {
		users = []domain.UserResponse{}
	}

	return users, total, nil
}

func (s *UserService) Create(ctx context.Context, req domain.CreateUserRequest) (*domain.UserResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var exists bool
	err := q.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, apperror.Internal("failed to check email", err)
	}
	if exists {
		return nil, apperror.Conflict("user with this email already exists")
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, apperror.Internal("failed to hash password", err)
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, apperror.Validation("invalid tenant_id")
	}

	now := time.Now()
	user := domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: hash,
		Role:         req.Role,
		TenantID:     &tenantID,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = q.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.ID, user.Email, user.Name, user.PasswordHash, user.Role, user.TenantID, user.IsActive, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, apperror.Internal("failed to create user", err)
	}

	resp := domain.ToUserResponse(user)
	return &resp, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.UserResponse, error) {
	q := middleware.GetQuerier(ctx, s.db)

	var u domain.UserResponse
	err := q.QueryRow(ctx,
		`SELECT id, email, name, role, tenant_id, is_active, created_at, updated_at
		 FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, apperror.NotFound("user not found")
	}
	return &u, nil
}
