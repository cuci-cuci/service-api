package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/bangun-ekosistem/service-api/internal/config"
	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
)

// --- Pure logic tests for auth_service ---

func TestHashPassword_ReturnsValidBcryptHash(t *testing.T) {
	password := "secureP@ssword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	// Verify the hash is valid bcrypt and matches the password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Errorf("bcrypt.CompareHashAndPassword failed: %v", err)
	}
}

func TestHashPassword_DifferentPasswordsProduceDifferentHashes(t *testing.T) {
	hash1, err := HashPassword("password1")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	hash2, err := HashPassword("password2")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash1 == hash2 {
		t.Error("expected different hashes for different passwords")
	}
}

func TestHashPassword_SamePasswordProducesDifferentHashes(t *testing.T) {
	// Due to bcrypt salt, same password should produce different hashes
	hash1, err := HashPassword("samePassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	hash2, err := HashPassword("samePassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash1 == hash2 {
		t.Error("expected different hashes due to bcrypt salt")
	}
}

func TestGenerateTokenPair_ValidTokens(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:            "test-secret-key-for-jwt-signing",
		JWTExpiryMinutes:     15,
		JWTRefreshExpiryDays: 7,
	}
	svc := NewAuthService(nil, cfg) // db is nil since we're testing pure logic

	tenantID := uuid.New()
	user := domain.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "$2a$10$fakehash",
		Role:         domain.RoleTenantOwner,
		TenantID:     &tenantID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	resp, err := svc.generateTokenPair(user)
	if err != nil {
		t.Fatalf("generateTokenPair returned error: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.AccessToken == resp.RefreshToken {
		t.Error("access token and refresh token should be different")
	}

	// Verify access token claims
	accessClaims := &middleware.Claims{}
	accessToken, err := jwt.ParseWithClaims(resp.AccessToken, accessClaims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse access token: %v", err)
	}
	if !accessToken.Valid {
		t.Error("access token should be valid")
	}
	if accessClaims.UserID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, accessClaims.UserID)
	}
	if accessClaims.Role != user.Role {
		t.Errorf("expected role %s, got %s", user.Role, accessClaims.Role)
	}
	if accessClaims.TenantID == nil || *accessClaims.TenantID != tenantID {
		t.Errorf("expected tenant ID %v, got %v", tenantID, accessClaims.TenantID)
	}

	// Verify refresh token claims
	refreshClaims := &middleware.Claims{}
	refreshToken, err := jwt.ParseWithClaims(resp.RefreshToken, refreshClaims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse refresh token: %v", err)
	}
	if !refreshToken.Valid {
		t.Error("refresh token should be valid")
	}
	if refreshClaims.UserID != user.ID {
		t.Errorf("expected user ID %v in refresh token, got %v", user.ID, refreshClaims.UserID)
	}

	// Verify expiration times
	if resp.ExpiresAt.Before(time.Now()) {
		t.Error("access token expiry should be in the future")
	}
	expectedExpiry := time.Now().Add(time.Duration(cfg.JWTExpiryMinutes) * time.Minute)
	diff := resp.ExpiresAt.Sub(expectedExpiry)
	if diff > 2*time.Second || diff < -2*time.Second {
		t.Errorf("access token expiry should be ~%d minutes from now, got diff=%v", cfg.JWTExpiryMinutes, diff)
	}

	// Verify user response
	if resp.User.ID != user.ID {
		t.Errorf("expected user response ID %v, got %v", user.ID, resp.User.ID)
	}
	if resp.User.Email != user.Email {
		t.Errorf("expected user response email %s, got %s", user.Email, resp.User.Email)
	}
	if resp.User.Role != user.Role {
		t.Errorf("expected user response role %s, got %s", user.Role, resp.User.Role)
	}
}

func TestGenerateTokenPair_SuperadminWithoutTenant(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:            "test-secret-key-for-jwt-signing",
		JWTExpiryMinutes:     15,
		JWTRefreshExpiryDays: 7,
	}
	svc := NewAuthService(nil, cfg)

	user := domain.User{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		Name:         "Super Admin",
		PasswordHash: "$2a$10$fakehash",
		Role:         domain.RoleSuperadmin,
		TenantID:     nil,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	resp, err := svc.generateTokenPair(user)
	if err != nil {
		t.Fatalf("generateTokenPair returned error: %v", err)
	}

	// Verify claims have nil tenant
	claims := &middleware.Claims{}
	_, err = jwt.ParseWithClaims(resp.AccessToken, claims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if claims.TenantID != nil {
		t.Errorf("expected nil tenant ID for superadmin, got %v", claims.TenantID)
	}
	if claims.Role != domain.RoleSuperadmin {
		t.Errorf("expected role superadmin, got %s", claims.Role)
	}
}

func TestRefreshToken_ExpiredTokenRejected(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:            "test-secret-key-for-jwt-signing",
		JWTExpiryMinutes:     15,
		JWTRefreshExpiryDays: 7,
	}
	svc := NewAuthService(nil, cfg)

	// Create an already-expired refresh token
	userID := uuid.New()
	tenantID := uuid.New()
	claims := &middleware.Claims{
		UserID:   userID,
		Role:     domain.RoleTenantOwner,
		TenantID: &tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired 1 hour ago
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// RefreshToken should reject the expired token
	_, err = svc.RefreshToken(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for expired refresh token, got nil")
	}

	// Verify the error message indicates token invalidity
	if err.Error() != "invalid or expired refresh token" {
		t.Errorf("expected 'invalid or expired refresh token', got: %s", err.Error())
	}
}

func TestRefreshToken_InvalidSignatureRejected(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:            "correct-secret",
		JWTExpiryMinutes:     15,
		JWTRefreshExpiryDays: 7,
	}
	svc := NewAuthService(nil, cfg)

	// Create a token signed with a different secret
	userID := uuid.New()
	claims := &middleware.Claims{
		UserID: userID,
		Role:   domain.RoleTenantOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("wrong-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = svc.RefreshToken(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for token with invalid signature, got nil")
	}
}

func TestLogin_PasswordVerification(t *testing.T) {
	// Test the password comparison logic used in Login
	// This tests the same bcrypt.CompareHashAndPassword call the Login method uses
	password := "correctpassword"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Correct password should match
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		t.Errorf("expected correct password to match, got error: %v", err)
	}

	// Wrong password should not match
	err = bcrypt.CompareHashAndPassword(hash, []byte("wrongpassword"))
	if err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestGenerateTokenPair_AccessTokenExpiresBeforeRefresh(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:            "test-secret",
		JWTExpiryMinutes:     15,
		JWTRefreshExpiryDays: 7,
	}
	svc := NewAuthService(nil, cfg)

	user := domain.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Name:         "Test",
		PasswordHash: "$2a$10$fakehash",
		Role:         domain.RoleCashier,
		TenantID:     ptrUUID(uuid.New()),
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	resp, err := svc.generateTokenPair(user)
	if err != nil {
		t.Fatalf("generateTokenPair returned error: %v", err)
	}

	// Parse both tokens to compare expiry
	accessClaims := &middleware.Claims{}
	jwt.ParseWithClaims(resp.AccessToken, accessClaims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})

	refreshClaims := &middleware.Claims{}
	jwt.ParseWithClaims(resp.RefreshToken, refreshClaims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})

	accessExpiry := accessClaims.ExpiresAt.Time
	refreshExpiry := refreshClaims.ExpiresAt.Time

	if !accessExpiry.Before(refreshExpiry) {
		t.Errorf("access token expiry (%v) should be before refresh token expiry (%v)", accessExpiry, refreshExpiry)
	}
}

// ptrUUID is a helper to get a pointer to a UUID.
func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}
