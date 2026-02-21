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
