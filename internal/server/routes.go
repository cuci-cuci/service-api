package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/handler"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/handler/admin"
	"github.com/bangun-ekosistem/service-api/internal/handler/pos"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

func (s *Server) RegisterRoutes() {
	// Global middleware
	s.Router.Use(chimw.RealIP)
	s.Router.Use(middleware.RequestLogging)
	s.Router.Use(middleware.Recovery)
	s.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   strings.Split(s.Config.CORSAllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Services
	authService := service.NewAuthService(s.DB, s.Config)
	tenantService := service.NewTenantService(s.DB)
	outletService := service.NewOutletService(s.DB)
	templateService := service.NewServiceTemplateService(s.DB)
	configService := service.NewConfigService(s.DB, templateService)
	syncService := service.NewSyncService(s.DB, configService, templateService)
	featureFlagService := service.NewFeatureFlagService(s.DB)
	auditService := service.NewAuditService(s.DB)
	analyticsService := service.NewAnalyticsService(s.DB)
	userService := service.NewUserService(s.DB)
	memberService := service.NewMemberService(s.DB)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, s.Validate)
	tenantHandler := admin.NewTenantHandler(tenantService, s.Validate)
	outletHandler := admin.NewOutletHandler(outletService, s.Validate)
	templateHandler := admin.NewServiceTemplateHandler(templateService, s.Validate)
	configHandler := admin.NewConfigHandler(configService, s.Validate)
	featureFlagHandler := admin.NewFeatureFlagHandler(featureFlagService, s.Validate)
	userHandler := admin.NewUserHandler(userService, s.Validate)
	analyticsHandler := admin.NewAnalyticsHandler(analyticsService)
	auditLogHandler := admin.NewAuditLogHandler(auditService)
	membershipHandler := admin.NewMembershipHandler(memberService, s.Validate)

	syncHandler := pos.NewSyncHandler(syncService, s.Validate)
	transactionHandler := pos.NewTransactionHandler(memberService)
	memberHandler := pos.NewMemberHandler(memberService)

	// Health check
	s.Router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// API v1
	s.Router.Route("/api/v1", func(r chi.Router) {
		// Auth (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshToken)
		})

		// Admin routes (authenticated)
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.JWTAuth(s.Config.JWTSecret))
			r.Use(middleware.TenantIsolation(s.DB))

			// Tenants (superadmin only)
			r.Route("/tenants", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", tenantHandler.List)
				r.Post("/", tenantHandler.Create)

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", tenantHandler.GetByID)
					r.Put("/", tenantHandler.Update)

					// Outlets for tenant
					r.Get("/outlets", outletHandler.ListByTenant)
					r.Post("/outlets", outletHandler.Create)

					// Config
					r.Get("/config", configHandler.GetCurrentConfig)
					r.Post("/config/push", configHandler.PushConfig)

					// Feature flags for tenant
					r.Get("/feature-flags", featureFlagHandler.GetTenantFlags)
					r.Put("/feature-flags", featureFlagHandler.SetTenantFlag)

					// Sync health
					r.Get("/sync-health", func(w http.ResponseWriter, req *http.Request) {
						tenantIDStr := chi.URLParam(req, "id")
						tid, err := parseUUID(tenantIDStr)
						if err != nil {
							response.Error(w, err)
							return
						}
						health, svcErr := syncService.GetSyncHealth(req.Context(), tid)
						if svcErr != nil {
							response.Error(w, svcErr)
							return
						}
						response.JSON(w, http.StatusOK, health)
					})
				})
			})

			// Outlets
			r.Route("/outlets/{id}", func(r chi.Router) {
				r.Use(middleware.RequireRole("superadmin", "tenant_owner"))
				r.Get("/", outletHandler.GetByID)
				r.Put("/", outletHandler.Update)
			})

			// Users (superadmin only)
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
			})

			// Service categories
			r.Route("/service-categories", func(r chi.Router) {
				r.Use(middleware.RequireRole("superadmin", "tenant_owner"))
				r.Get("/", templateHandler.ListCategories)
				r.Post("/", templateHandler.CreateCategory)
				r.Put("/{id}", templateHandler.UpdateCategory)
			})

			// Service templates
			r.Route("/service-templates", func(r chi.Router) {
				r.Use(middleware.RequireRole("superadmin", "tenant_owner"))
				r.Get("/", templateHandler.ListTemplates)
				r.Post("/", templateHandler.CreateTemplate)
				r.Put("/{id}", templateHandler.UpdateTemplate)
			})

			// Feature flags (global)
			r.Route("/feature-flags", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", featureFlagHandler.ListFlags)
				r.Post("/", featureFlagHandler.CreateFlag)
				r.Put("/{id}", featureFlagHandler.UpdateFlag)
			})

			// Membership
			r.Route("/membership", func(r chi.Router) {
				r.Use(middleware.RequireRole("superadmin", "tenant_owner"))
				r.Get("/", membershipHandler.List)
				r.Post("/", membershipHandler.Create)
				r.Put("/{id}", membershipHandler.Update)
			})

			// Analytics
			r.Route("/analytics", func(r chi.Router) {
				r.Use(middleware.RequireRole("superadmin", "tenant_owner"))
				r.Get("/revenue", analyticsHandler.Revenue)
				r.Get("/transactions", analyticsHandler.TransactionStats)
			})

			// Audit logs
			r.Route("/audit-logs", func(r chi.Router) {
				r.Use(middleware.RequireRole("superadmin", "tenant_owner"))
				r.Get("/", auditLogHandler.List)
			})

			// Config broadcast
			r.Route("/config", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Post("/broadcast", configHandler.BroadcastConfig)
			})
		})

		// POS routes (authenticated, tenant-scoped)
		r.Route("/pos", func(r chi.Router) {
			r.Use(middleware.JWTAuth(s.Config.JWTSecret))
			r.Use(middleware.TenantIsolation(s.DB))
			r.Use(middleware.RequireRole("tenant_owner", "cashier"))

			r.Post("/sync/upload", syncHandler.Upload)
			r.Get("/sync/download", syncHandler.Download)
			r.Get("/transactions", transactionHandler.List)
			r.Get("/members/lookup", memberHandler.Lookup)
		})
	})
}

func parseUUID(s string) (uuid.UUID, *apperror.AppError) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, apperror.NewAppError(400, "invalid UUID")
	}
	return id, nil
}
