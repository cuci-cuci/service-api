package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/db"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
)

type contextKeyDB string

const ContextKeyDBTx contextKeyDB = "db_tx"

func TenantIsolation(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetUserRole(r.Context())
			tenantID := GetTenantID(r.Context())

			conn, err := pool.Acquire(r.Context())
			if err != nil {
				slog.Error("failed to acquire db connection", "error", err)
				response.Error(w, apperror.Internal("failed to acquire database connection", err))
				return
			}
			defer conn.Release()

			tx, err := conn.Begin(r.Context())
			if err != nil {
				slog.Error("failed to begin transaction", "error", err)
				response.Error(w, apperror.Internal("failed to begin transaction", err))
				return
			}

			// Use fmt.Sprintf for SET LOCAL because PostgreSQL SET does not support
			// parameterized placeholders ($1). These values come from JWT claims
			// (trusted, server-side generated), NOT from user input.
			if role == "superadmin" {
				if _, err := tx.Exec(r.Context(), `SET LOCAL "app.current_role" = 'superadmin'`); err != nil {
					_ = tx.Rollback(r.Context())
					slog.Error("failed to set role", "error", err)
					response.Error(w, apperror.Internal("failed to set session role", err))
					return
				}
			} else {
				if tenantID == nil {
					_ = tx.Rollback(r.Context())
					response.Error(w, apperror.Forbidden("tenant context required"))
					return
				}
				if _, err := tx.Exec(r.Context(), fmt.Sprintf(`SET LOCAL "app.current_tenant_id" = '%s'`, tenantID.String())); err != nil {
					_ = tx.Rollback(r.Context())
					slog.Error("failed to set tenant_id", "error", err)
					response.Error(w, apperror.Internal("failed to set tenant context", err))
					return
				}
				if _, err := tx.Exec(r.Context(), fmt.Sprintf(`SET LOCAL "app.current_role" = '%s'`, role)); err != nil {
					_ = tx.Rollback(r.Context())
					slog.Error("failed to set role", "error", err)
					response.Error(w, apperror.Internal("failed to set session role", err))
					return
				}
			}

			ctx := context.WithValue(r.Context(), ContextKeyDBTx, tx)
			next.ServeHTTP(w, r.WithContext(ctx))

			if err := tx.Commit(r.Context()); err != nil {
				slog.Error("failed to commit transaction", "error", err)
			}
		})
	}
}

// GetQuerier returns the transaction from context if available (set by
// TenantIsolation middleware), otherwise returns the provided fallback
// querier (typically the pool). This allows services to transparently
// use RLS-scoped transactions when the middleware is active.
func GetQuerier(ctx context.Context, fallback db.Querier) db.Querier {
	if tx, ok := ctx.Value(ContextKeyDBTx).(pgx.Tx); ok {
		return tx
	}
	return fallback
}
