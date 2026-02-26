package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/handler"
	"github.com/bangun-ekosistem/service-api/internal/handler/admin"
	"github.com/bangun-ekosistem/service-api/internal/handler/owner"
	"github.com/bangun-ekosistem/service-api/internal/handler/pos"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

func (s *Server) RegisterRoutes() {
	// Global middleware
	s.Router.Use(chimw.RealIP)
	s.Router.Use(middleware.RequestLogging)
	s.Router.Use(middleware.Recovery)
	s.Router.Use(middleware.MaxBodySize)

	// FIX 2: CORS - do not combine AllowCredentials with wildcard origin.
	origins := strings.Split(s.Config.CORSAllowedOrigins, ",")
	corsOpts := cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		MaxAge:           300,
	}

	// Only enable AllowCredentials when origins are explicitly specified (not wildcard).
	hasWildcard := false
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			hasWildcard = true
			break
		}
	}
	if !hasWildcard {
		corsOpts.AllowCredentials = true
	}

	s.Router.Use(cors.Handler(corsOpts))

	// Services
	authService := service.NewAuthService(s.DB, s.Config)
	tenantService := service.NewTenantService(s.DB)
	outletService := service.NewOutletService(s.DB)
	templateService := service.NewServiceTemplateService(s.DB)
	configService := service.NewConfigService(s.DB, templateService)
	orderService := service.NewOrderService(s.DB)
	shiftService := service.NewShiftService(s.DB)
	memberService := service.NewMemberService(s.DB)
	syncService := service.NewSyncService(s.DB, configService, templateService, orderService, memberService)
	featureFlagService := service.NewFeatureFlagService(s.DB)
	auditService := service.NewAuditService(s.DB)
	analyticsService := service.NewAnalyticsService(s.DB)
	userService := service.NewUserService(s.DB)
	dashboardService := service.NewDashboardService(s.DB)
	syncMonitorService := service.NewSyncMonitorService(s.DB)
	billingService := service.NewBillingService(s.DB)
	notificationService := service.NewNotificationService(s.DB, s.Config)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, billingService, s.Validate)
	tenantHandler := admin.NewTenantHandler(tenantService, auditService, s.Validate)
	outletHandler := admin.NewOutletHandler(outletService, s.Validate)
	templateHandler := admin.NewServiceTemplateHandler(templateService, s.Validate)
	configHandler := admin.NewConfigHandler(configService, auditService, s.Validate)
	featureFlagHandler := admin.NewFeatureFlagHandler(featureFlagService, s.Validate)
	userHandler := admin.NewUserHandler(userService, auditService, s.Validate)
	analyticsHandler := admin.NewAnalyticsHandler(analyticsService)
	auditLogHandler := admin.NewAuditLogHandler(auditService)
	membershipHandler := admin.NewMembershipHandler(memberService, s.Validate)
	dashboardHandler := admin.NewDashboardHandler(dashboardService)
	syncMonitorHandler := admin.NewSyncMonitorHandler(syncMonitorService)

	paymentMethodService := service.NewPaymentMethodService(s.DB)

	ownerOutletHandler := owner.NewOutletHandler(outletService, billingService, s.Validate)
	ownerServicePriceHandler := owner.NewServicePriceHandler(templateService, s.Validate)
	ownerPaymentMethodHandler := owner.NewPaymentMethodHandler(paymentMethodService, s.Validate)
	ownerCashierHandler := owner.NewCashierHandler(userService, s.Validate)
	ownerMemberHandler := owner.NewMemberHandler(memberService, s.Validate)
	ownerAnalyticsHandler := owner.NewAnalyticsHandler(analyticsService)
	ownerConfigHandler := owner.NewConfigHandler(configService)
	ownerBillingHandler := owner.NewBillingHandler(billingService)
	ownerNotificationHandler := owner.NewNotificationHandler(notificationService)

	trackingHandler := handler.NewTrackingHandler(orderService)

	syncHandler := pos.NewSyncHandler(syncService, billingService, s.Validate)
	transactionHandler := pos.NewTransactionHandler(memberService)
	memberHandler := pos.NewMemberHandler(memberService)
	posOutletHandler := pos.NewOutletHandler(outletService)
	orderHandler := pos.NewOrderHandler(orderService, notificationService, s.Validate)
	shiftHandler := pos.NewShiftHandler(shiftService, s.Validate)

	// Health check
	s.Router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// API v1
	s.Router.Route("/api/v1", func(r chi.Router) {
		// Auth (public, rate-limited)
		r.Route("/auth", func(r chi.Router) {
			r.Use(middleware.RateLimit(20, time.Minute))
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/register", authHandler.Register)
			r.Post("/forgot-password", authHandler.RequestPasswordReset)
			r.Post("/reset-password", authHandler.ResetPassword)
		})

		// Public: list subscription plans (no auth needed)
		r.Get("/plans", ownerBillingHandler.ListPlans)

		// Public: order tracking (no auth needed)
		r.Get("/track/{token}", trackingHandler.GetByToken)

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
					r.Delete("/", tenantHandler.Delete)

					// Outlets for tenant
					r.Get("/outlets", outletHandler.ListByTenant)
					r.Post("/outlets", outletHandler.Create)

					// Config
					r.Get("/config", configHandler.GetCurrentConfig)
					r.Post("/config/push", configHandler.PushConfig)

					// Feature flags for tenant
					r.Get("/feature-flags", featureFlagHandler.GetTenantFlags)
					r.Put("/feature-flags", featureFlagHandler.SetTenantFlag)

					// Analytics by outlet
					r.Get("/analytics/outlets", analyticsHandler.RevenueByOutlet)

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
				r.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleTenantOwner))
				r.Get("/", outletHandler.GetByID)
				r.Put("/", outletHandler.Update)
				r.Delete("/", outletHandler.Delete)
			})

			// Users (superadmin only)
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
				r.Put("/{id}", userHandler.Update)
				r.Delete("/{id}", userHandler.Delete)
			})

			// Service categories
			r.Route("/service-categories", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleTenantOwner))
				r.Get("/", templateHandler.ListCategories)
				r.Post("/", templateHandler.CreateCategory)
				r.Put("/{id}", templateHandler.UpdateCategory)
				r.Delete("/{id}", templateHandler.DeleteCategory)
			})

			// Service templates
			r.Route("/service-templates", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleTenantOwner))
				r.Get("/", templateHandler.ListTemplates)
				r.Post("/", templateHandler.CreateTemplate)
				r.Put("/{id}", templateHandler.UpdateTemplate)
				r.Delete("/{id}", templateHandler.DeleteTemplate)
			})

			// Feature flags (global)
			r.Route("/feature-flags", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", featureFlagHandler.ListFlags)
				r.Post("/", featureFlagHandler.CreateFlag)
				r.Put("/{id}", featureFlagHandler.UpdateFlag)
				r.Delete("/{id}", featureFlagHandler.DeleteFlag)
				r.Get("/{id}/tenants", featureFlagHandler.GetFlagTenantOverrides)
			})

			// Tenant feature flag toggle
			r.Post("/tenant-feature-flags", featureFlagHandler.ToggleTenantFlag)

			// Membership
			r.Route("/membership", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleTenantOwner))
				r.Get("/", membershipHandler.List)
				r.Post("/", membershipHandler.Create)
				r.Put("/{id}", membershipHandler.Update)
				r.Delete("/{id}", membershipHandler.Delete)
			})

			// Analytics
			r.Route("/analytics", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleTenantOwner))
				r.Get("/revenue", analyticsHandler.Revenue)
				r.Get("/transactions", analyticsHandler.TransactionStats)
			})

			// Audit logs
			r.Route("/audit-logs", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleTenantOwner))
				r.Get("/", auditLogHandler.List)
			})

			// Config broadcast
			r.Route("/config", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Post("/broadcast", configHandler.BroadcastConfig)
			})

			// Dashboard (superadmin only)
			r.Route("/dashboard", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/stats", dashboardHandler.Stats)
				r.Get("/revenue", dashboardHandler.Revenue)
			})

			// Transactions (admin view)
			r.Route("/transactions", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", dashboardHandler.TransactionList)
			})

			// Sync monitor (superadmin only)
			r.Route("/sync", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/health/outlets", syncMonitorHandler.OutletHealth)
				r.Get("/health", syncMonitorHandler.GlobalHealth)
				r.Get("/health/stream", syncMonitorHandler.StreamHealth)
				r.Get("/sessions", syncMonitorHandler.ListSessions)
			})

			// Configs (list all tenant configs)
			r.Route("/configs", func(r chi.Router) {
				r.Use(middleware.RequireSuperadmin())
				r.Get("/", configHandler.ListAllConfigs)
				r.Post("/broadcast", configHandler.BroadcastConfig)
			})
		})

		// Owner routes (authenticated, tenant-scoped, tenant_owner only)
		r.Route("/owner", func(r chi.Router) {
			r.Use(middleware.JWTAuth(s.Config.JWTSecret))
			r.Use(middleware.TenantIsolation(s.DB))
			r.Use(middleware.RequireRole(domain.RoleTenantOwner))

			// Outlets
			r.Get("/outlets", ownerOutletHandler.List)
			r.Post("/outlets", ownerOutletHandler.Create)
			r.Put("/outlets/{id}", ownerOutletHandler.Update)

			// Services & Pricing
			r.Get("/services", ownerServicePriceHandler.ListServicesWithPrices)
			r.Put("/service-prices/{templateId}", ownerServicePriceHandler.SetPrice)
			r.Put("/service-prices/{templateId}/quick-add", ownerServicePriceHandler.SetQuickAdd)
			r.Put("/service-prices/bulk", ownerServicePriceHandler.BulkSetPrices)

			// Payment Methods
			r.Get("/payment-methods", ownerPaymentMethodHandler.List)
			r.Post("/payment-methods", ownerPaymentMethodHandler.Create)
			r.Put("/payment-methods/{id}", ownerPaymentMethodHandler.Update)
			r.Delete("/payment-methods/{id}", ownerPaymentMethodHandler.Delete)

			// Cashiers
			r.Get("/cashiers", ownerCashierHandler.List)
			r.Post("/cashiers", ownerCashierHandler.Create)
			r.Put("/cashiers/{id}", ownerCashierHandler.Update)

			// Members
			r.Get("/members", ownerMemberHandler.List)
			r.Post("/members", ownerMemberHandler.Create)
			r.Put("/members/{id}", ownerMemberHandler.Update)

			// Store settings (config)
			r.Get("/store-settings", ownerConfigHandler.GetStoreSettings)
			r.Put("/store-settings", ownerConfigHandler.UpdateStoreSettings)

			// Subscription & billing
			r.Get("/subscription", ownerBillingHandler.GetSubscription)

			// Notification settings (WhatsApp)
			r.Get("/notification-settings", ownerNotificationHandler.GetSettings)
			r.Put("/notification-settings", ownerNotificationHandler.UpdateSettings)

			// Analytics
			r.Get("/analytics/summary", ownerAnalyticsHandler.Summary)
			r.Get("/analytics/outlets", ownerAnalyticsHandler.RevenueByOutlet)
			r.Get("/analytics/daily-revenue", ownerAnalyticsHandler.DailyRevenue)
			r.Get("/analytics/by-service", ownerAnalyticsHandler.RevenueByService)
			r.Get("/analytics/by-payment-method", ownerAnalyticsHandler.RevenueByPaymentMethod)
		})

		// POS routes (authenticated, tenant-scoped)
		r.Route("/pos", func(r chi.Router) {
			r.Use(middleware.JWTAuth(s.Config.JWTSecret))
			r.Use(middleware.TenantIsolation(s.DB))
			r.Use(middleware.RequireRole(domain.RoleTenantOwner, domain.RoleCashier))

			r.Post("/sync/upload", syncHandler.Upload)
			r.Get("/sync/download", syncHandler.Download)
			r.Get("/transactions", transactionHandler.List)
			r.Get("/members/lookup", memberHandler.Lookup)
			r.Get("/members/search", memberHandler.Search)
			r.Post("/members", memberHandler.Register)
			r.Get("/outlets", posOutletHandler.List)

			r.Get("/orders", orderHandler.List)
			r.Get("/orders/active", orderHandler.ListActive)
			r.Post("/orders", orderHandler.Create)
			r.Route("/orders/{id}", func(r chi.Router) {
				r.Get("/", orderHandler.GetByID)
				r.Put("/status", orderHandler.UpdateStatus)
			})

			r.Post("/shifts/open", shiftHandler.Open)
			r.Post("/shifts/close", shiftHandler.Close)
			r.Get("/shifts/current", shiftHandler.GetCurrent)
			r.Get("/shifts", shiftHandler.List)
			r.Get("/shifts/{id}/summary", shiftHandler.GetSummary)
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
