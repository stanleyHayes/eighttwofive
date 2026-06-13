package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
	r.Use(middleware.Recoverer)
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

		r.Post("/orders", h.CreateOrder)
		r.Post("/orders/request", h.CreateCustomRequest)
		r.Post("/payments/webhook", h.HandlePaymentWebhook)
		r.Get("/slots", h.ListOpenSlots)

		// Booking claims a slot before any deposit is paid, so the public
		// endpoint is rate-limited to blunt inventory-exhaustion abuse.
		bookLimiter := newRateLimiter(bookRateLimit, bookRateWindow)
		r.With(rateLimitByIP(bookLimiter)).Post("/slots/{id}/book", h.BookSlot)

		r.Route("/auth", func(r chi.Router) {
			// The magic-link request sends an email and is unauthenticated, so
			// rate-limit it per client to blunt email-spam and enumeration abuse.
			loginLimiter := newRateLimiter(loginRateLimit, loginRateWindow)
			r.With(rateLimitByIP(loginLimiter)).Post("/request-link", h.RequestLoginLink)
			r.Post("/verify", h.VerifyLogin)
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
	return func(r chi.Router) {
		r.Use(h.RequireAuth, h.RequireAdmin)
		r.Get("/waitlist", h.ListWaitlist)
		r.Put("/settings", h.UpdateSettings)
		r.Post("/uploads/sign", h.SignUpload)
		r.Get("/analytics", h.AdminGetAnalytics)
		r.Get("/orders", h.AdminListOrders)
		r.Get("/orders/{ref}", h.AdminGetOrder)
		r.Put("/orders/{ref}/quote", h.AdminUpdateQuote)
		r.Post("/orders/{ref}/payment-link", h.AdminSendPaymentLink)
		r.Post("/orders/{ref}/mark-paid", h.AdminMarkPaid)
		r.Post("/orders/{ref}/status", h.AdminUpdateOrderStatus)

		r.Route("/slots", func(r chi.Router) {
			r.Get("/", h.AdminListSlots)
			r.Post("/", h.AdminCreateSlot)
			r.Post("/{id}/close", h.AdminCloseSlot)
			r.Post("/{id}/reopen", h.AdminReopenSlot)
		})

		r.Route("/visits", func(r chi.Router) {
			r.Get("/", h.AdminListVisits)
			r.Post("/{id}/reschedule", h.AdminRescheduleVisit)
			r.Post("/{id}/cancel", h.AdminCancelVisit)
		})

		r.Route("/collections", func(r chi.Router) {
			r.Get("/", h.AdminListCollections)
			r.Post("/", h.AdminCreateCollection)
			r.Put("/{id}", h.AdminUpdateCollection)
			r.Post("/{id}/retire", h.AdminRetireCollection)
			r.Post("/{id}/restore", h.AdminRestoreCollection)
			r.Delete("/{id}", h.AdminDeleteCollection)
		})

		r.Route("/designs", func(r chi.Router) {
			r.Get("/", h.AdminListDesigns)
			r.Get("/{id}", h.AdminGetDesign)
			r.Post("/", h.AdminCreateDesign)
			r.Post("/retire", h.AdminRetireDesigns)
			r.Post("/restore", h.AdminRestoreDesigns)
			r.Put("/{id}", h.AdminUpdateDesign)
			r.Delete("/{id}", h.AdminDeleteDesign)
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
