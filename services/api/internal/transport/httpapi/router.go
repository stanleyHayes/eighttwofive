package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	requestTimeout    = 30 * time.Second
	corsMaxAgeSeconds = 300
)

// NewRouter assembles the HTTP routing tree with standard middleware.
func NewRouter(h *Handlers, logger *slog.Logger, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(recoverer(logger))
	r.Use(middleware.Timeout(requestTimeout))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		MaxAge:           corsMaxAgeSeconds,
		AllowCredentials: true,
	}))

	// Bare liveness probe for Render health checks.
	r.Get("/healthz", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", h.Health)
		r.Post("/waitlist", h.JoinWaitlist)
		r.Get("/settings", h.GetSettings)
		r.Get("/collections", h.ListCollections)
		r.Get("/collections/{slug}", h.GetCollection)
		r.Get("/designs", h.ListDesigns)
		r.Get("/designs/{slug}", h.GetDesign)

		// Unauthenticated checkout creates DB documents + a Paystack transaction,
		// so rate-limit it per client.
		checkoutLimiter := newRateLimiter(checkoutRateLimit, checkoutRateWindow)
		r.With(rateLimitByIP(checkoutLimiter)).Post("/orders", h.CreateOrder)
		r.With(rateLimitByIP(checkoutLimiter)).Post("/orders/request", h.CreateCustomRequest)
		r.Post("/payments/webhook", h.HandlePaymentWebhook)
		r.Get("/slots", h.ListOpenSlots)

		// Booking claims a slot before any deposit is paid, so the public
		// endpoint is rate-limited to blunt inventory-exhaustion abuse.
		bookLimiter := newRateLimiter(bookRateLimit, bookRateWindow)
		r.With(rateLimitByIP(bookLimiter)).Post("/slots/{id}/book", h.BookSlot)

		r.Route("/auth", func(r chi.Router) {
			// The magic-link request and verify endpoints are unauthenticated and
			// security-sensitive, so rate-limit both per client (email-spam,
			// enumeration, and token brute-force).
			loginLimiter := newRateLimiter(loginRateLimit, loginRateWindow)
			r.With(rateLimitByIP(loginLimiter)).Post("/request-link", h.RequestLoginLink)
			r.With(rateLimitByIP(loginLimiter)).Post("/verify", h.VerifyLogin)
			r.Post("/logout", h.Logout)

			r.Group(func(r chi.Router) {
				r.Use(h.RequireAuth)
				r.Get("/me", h.Me)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(h.RequireAuth)
			r.Get("/orders", h.ListCustomerOrders)
			r.Get("/orders/{ref}", h.GetOrder)
		})

		r.Route("/admin", adminRoutes(h))
	})

	return r
}

func adminRoutes(h *Handlers) func(chi.Router) {
	// Per-permission guards keep each route's required capability explicit.
	read := func(p domain.Permission) func(http.Handler) http.Handler { return h.RequirePermission(p) }

	return func(r chi.Router) {
		// Any admin-area role (viewer/staff/admin) may enter; each route then
		// requires its specific capability.
		r.Use(h.RequireAuth, h.RequireAdminArea)

		r.With(read(domain.PermSubscribersRead)).Get("/waitlist", h.ListWaitlist)
		r.With(read(domain.PermSettingsWrite)).Put("/settings", h.UpdateSettings)
		r.With(read(domain.PermCatalogueWrite)).Post("/uploads/sign", h.SignUpload)
		r.With(read(domain.PermAnalyticsRead)).Get("/analytics", h.AdminGetAnalytics)

		r.With(read(domain.PermOrdersRead)).Get("/orders", h.AdminListOrders)
		r.With(read(domain.PermOrdersRead)).Get("/orders/{ref}", h.AdminGetOrder)
		r.With(read(domain.PermOrdersWrite)).Put("/orders/{ref}/quote", h.AdminUpdateQuote)
		r.With(read(domain.PermOrdersWrite)).Post("/orders/{ref}/payment-link", h.AdminSendPaymentLink)
		r.With(read(domain.PermOrdersWrite)).Post("/orders/{ref}/mark-paid", h.AdminMarkPaid)
		r.With(read(domain.PermOrdersWrite)).Post("/orders/{ref}/status", h.AdminUpdateOrderStatus)

		r.Route("/users", func(r chi.Router) {
			r.With(read(domain.PermTeamRead)).Get("/", h.AdminListUsers)
			r.With(read(domain.PermTeamWrite)).Put("/{id}/role", h.AdminSetUserRole)
		})

		r.Route("/slots", func(r chi.Router) {
			r.With(read(domain.PermSlotsRead)).Get("/", h.AdminListSlots)
			r.With(read(domain.PermSlotsWrite)).Post("/", h.AdminCreateSlot)
			r.With(read(domain.PermSlotsWrite)).Post("/{id}/close", h.AdminCloseSlot)
			r.With(read(domain.PermSlotsWrite)).Post("/{id}/reopen", h.AdminReopenSlot)
		})

		r.Route("/visits", func(r chi.Router) {
			r.With(read(domain.PermSlotsRead)).Get("/", h.AdminListVisits)
			r.With(read(domain.PermSlotsWrite)).Post("/{id}/reschedule", h.AdminRescheduleVisit)
			r.With(read(domain.PermSlotsWrite)).Post("/{id}/cancel", h.AdminCancelVisit)
		})

		r.Route("/collections", func(r chi.Router) {
			r.With(read(domain.PermCatalogueRead)).Get("/", h.AdminListCollections)
			r.With(read(domain.PermCatalogueWrite)).Post("/", h.AdminCreateCollection)
			r.With(read(domain.PermCatalogueWrite)).Put("/{id}", h.AdminUpdateCollection)
			r.With(read(domain.PermCatalogueWrite)).Post("/{id}/retire", h.AdminRetireCollection)
			r.With(read(domain.PermCatalogueWrite)).Post("/{id}/restore", h.AdminRestoreCollection)
			r.With(read(domain.PermCatalogueDelete)).Delete("/{id}", h.AdminDeleteCollection)
		})

		r.Route("/designs", func(r chi.Router) {
			r.With(read(domain.PermCatalogueRead)).Get("/", h.AdminListDesigns)
			r.With(read(domain.PermCatalogueRead)).Get("/{id}", h.AdminGetDesign)
			r.With(read(domain.PermCatalogueWrite)).Post("/", h.AdminCreateDesign)
			r.With(read(domain.PermCatalogueWrite)).Post("/retire", h.AdminRetireDesigns)
			r.With(read(domain.PermCatalogueWrite)).Post("/restore", h.AdminRestoreDesigns)
			r.With(read(domain.PermCatalogueWrite)).Put("/{id}", h.AdminUpdateDesign)
			r.With(read(domain.PermCatalogueDelete)).Delete("/{id}", h.AdminDeleteDesign)
		})
	}
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapped, r)
			logger.InfoContext(
				r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"bytes", wrapped.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
