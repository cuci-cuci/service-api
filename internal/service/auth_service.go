package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/bangun-ekosistem/service-api/internal/config"
	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
)

type AuthService struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewAuthService(db *pgxpool.Pool, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.TokenResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, apperror.Internal("failed to begin transaction", err)
	}
	defer tx.Rollback(ctx)

	// Check if slug already exists
	var slugExists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)", req.Slug).Scan(&slugExists)
	if err != nil {
		return nil, apperror.Internal("failed to check slug", err)
	}
	if slugExists {
		return nil, apperror.Validation("slug already taken")
	}

	// Check if email already exists
	var emailExists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&emailExists)
	if err != nil {
		return nil, apperror.Internal("failed to check email", err)
	}
	if emailExists {
		return nil, apperror.Validation("email already registered")
	}

	// Insert tenant
	tenantID := uuid.New()
	now := time.Now()
	_, err = tx.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, true, $4, $5)`,
		tenantID, req.BusinessName, req.Slug, now, now)
	if err != nil {
		slog.Error("failed to insert tenant", "error", err)
		return nil, apperror.Internal("failed to create tenant", err)
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal("failed to hash password", err)
	}

	// Insert user
	userID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8)`,
		userID, req.Email, req.OwnerName, string(passwordHash), "tenant_owner", tenantID, now, now)
	if err != nil {
		slog.Error("failed to insert user", "error", err)
		return nil, apperror.Internal("failed to create user", err)
	}

	// Create default outlet
	outletID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO outlets (id, tenant_id, name, address, phone, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, '', '', true, $4, $5)`,
		outletID, tenantID, req.BusinessName+" - Pusat", now, now)
	if err != nil {
		slog.Error("failed to insert default outlet", "error", err)
		return nil, apperror.Internal("failed to create default outlet", err)
	}

	// Create default payment methods
	paymentMethods := []struct {
		Name string
		Type string
	}{
		{"Tunai", "cash"},
		{"QRIS", "qris"},
		{"Transfer Bank", "bank_transfer"},
	}
	for i, pm := range paymentMethods {
		_, err = tx.Exec(ctx,
			`INSERT INTO payment_methods (id, tenant_id, name, type, is_active, sort_order, created_at)
			 VALUES ($1, $2, $3, $4, true, $5, $6)`,
			uuid.New(), tenantID, pm.Name, pm.Type, i+1, now)
		if err != nil {
			slog.Error("failed to insert payment method", "error", err, "name", pm.Name)
			return nil, apperror.Internal("failed to create default payment methods", err)
		}
	}

	// Copy all active service templates as tenant service prices (using base_price)
	_, err = tx.Exec(ctx,
		`INSERT INTO tenant_service_prices (id, tenant_id, service_template_id, price, is_active)
		 SELECT uuid_generate_v4(), $1, id, base_price, true
		 FROM service_templates WHERE is_active = true`,
		tenantID)
	if err != nil {
		slog.Error("failed to seed tenant service prices", "error", err)
		return nil, apperror.Internal("failed to create default service prices", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperror.Internal("failed to commit transaction", err)
	}

	user := domain.User{
		ID:           userID,
		Email:        req.Email,
		Name:         req.OwnerName,
		PasswordHash: string(passwordHash),
		Role:         "tenant_owner",
		TenantID:     &tenantID,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return s.generateTokenPair(user)
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenResponse, error) {
	var user domain.User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at
		 FROM users WHERE email = $1`, req.Email).
		Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role,
			&user.TenantID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.Unauthorized("invalid email or password")
		}
		slog.Error("failed to query user", "error", err)
		return nil, apperror.Internal("failed to query user", err)
	}

	if !user.IsActive {
		return nil, apperror.Unauthorized("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperror.Unauthorized("invalid email or password")
	}

	tokenResp, err := s.generateTokenPair(user)
	if err != nil {
		return nil, err
	}

	return tokenResp, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenResponse, error) {
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperror.Unauthorized("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, apperror.Unauthorized("invalid or expired refresh token")
	}

	var user domain.User
	err = s.db.QueryRow(ctx,
		`SELECT id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at
		 FROM users WHERE id = $1`, claims.UserID).
		Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role,
			&user.TenantID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.Unauthorized("user not found")
		}
		return nil, apperror.Internal("failed to query user", err)
	}

	if !user.IsActive {
		return nil, apperror.Unauthorized("account is deactivated")
	}

	return s.generateTokenPair(user)
}

func (s *AuthService) generateTokenPair(user domain.User) (*domain.TokenResponse, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)

	accessClaims := &middleware.Claims{
		UserID:   user.ID,
		Role:     user.Role,
		TenantID: user.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, apperror.Internal("failed to sign access token", err)
	}

	refreshExpiresAt := now.Add(time.Duration(s.cfg.JWTExpiryHours*7) * time.Hour)
	refreshClaims := &middleware.Claims{
		UserID:   user.ID,
		Role:     user.Role,
		TenantID: user.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			ID:        uuid.New().String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, apperror.Internal("failed to sign refresh token", err)
	}

	return &domain.TokenResponse{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    expiresAt,
		User:         domain.ToUserResponse(user),
	}, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
