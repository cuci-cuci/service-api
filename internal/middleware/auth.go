package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyUserRole contextKey = "user_role"
	ContextKeyTenantID contextKey = "tenant_id"
	ContextKeyClaims   contextKey = "claims"
)

type Claims struct {
	UserID   uuid.UUID  `json:"user_id"`
	Role     string     `json:"role"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, apperror.Unauthorized("missing authorization header"))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				response.Error(w, apperror.Unauthorized("invalid authorization header format"))
				return
			}

			tokenStr := parts[1]
			claims := &Claims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, apperror.Unauthorized("unexpected signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				response.Error(w, apperror.Unauthorized("invalid or expired token"))
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyUserRole, claims.Role)
			ctx = context.WithValue(ctx, ContextKeyTenantID, claims.TenantID)
			ctx = context.WithValue(ctx, ContextKeyClaims, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(ContextKeyUserID).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func GetUserRole(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyUserRole).(string); ok {
		return v
	}
	return ""
}

func GetTenantID(ctx context.Context) *uuid.UUID {
	if v, ok := ctx.Value(ContextKeyTenantID).(*uuid.UUID); ok {
		return v
	}
	return nil
}

func GetClaims(ctx context.Context) *Claims {
	if v, ok := ctx.Value(ContextKeyClaims).(*Claims); ok {
		return v
	}
	return nil
}
