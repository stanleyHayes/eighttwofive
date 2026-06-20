package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// recoverer recovers panics, logs them through the structured logger (with the
// request id + stack) instead of chi's colorized stderr dump, and returns 500.
func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				logger.ErrorContext(ctx, "panic recovered",
					"error", fmt.Sprintf("%v", rec),
					"path", r.URL.Path,
					"request_id", chimw.GetReqID(ctx),
					"stack", string(debug.Stack()),
				)
				respondError(w, http.StatusInternalServerError, "internal", "something went wrong")
			}()

			next.ServeHTTP(w, r)
		})
	}
}

const sessionCookieName = "e25_session"

// msgAdminAccessRequired is the 403 message for users barred from the dashboard.
const msgAdminAccessRequired = "admin access required"

// msgNoAccess is the 403 message for a user whose role lacks a capability.
const msgNoAccess = "you don't have access to this"

type contextKey int

const (
	userContextKey contextKey = iota
	roleDefContextKey
)

func userFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)

	return user, ok
}

func roleDefFromContext(ctx context.Context) (*domain.RoleDef, bool) {
	def, ok := ctx.Value(roleDefContextKey).(*domain.RoleDef)

	return def, ok
}

// staticRoleDef builds a role definition from the built-in static matrix. It is
// the fallback when the editable store has no entry for a role key — a fresh
// database before seeding, or a deleted custom role — so the built-in roles
// keep working and an unknown key resolves to no permissions and no admin-area
// access (fails safe: deny).
func staticRoleDef(role domain.Role) *domain.RoleDef {
	return &domain.RoleDef{
		Key:         string(role),
		Permissions: role.Permissions(),
		AdminArea:   role.IsAdminArea(),
	}
}

// roleDef resolves a user's effective role definition from the editable store,
// so an admin's permission edit takes effect with no redeploy. A missing role
// falls back to the built-in static definition; an unexpected store error is
// returned so security-critical callers can fail closed rather than guess.
func (h *Handlers) roleDef(ctx context.Context, user *domain.User) (*domain.RoleDef, error) {
	if h.roles == nil {
		return staticRoleDef(user.Role), nil
	}

	def, err := h.roles.Resolve(ctx, string(user.Role))

	switch {
	case errors.Is(err, domain.ErrNotFound):
		return staticRoleDef(user.Role), nil
	case err != nil:
		return nil, fmt.Errorf("role lookup: %w", err)
	default:
		return def, nil
	}
}

// RequireAuth rejects requests without a valid session and injects the user
// into the request context for downstream handlers.
func (h *Handlers) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

			return
		}

		user, err := h.auth.UserFromSession(r.Context(), cookie.Value)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}

const (
	// bookRateLimit and bookRateWindow bound how many bookings a single client
	// may attempt per window: slots are claimed before the deposit is paid, so
	// the unauthenticated book endpoint must not allow inventory exhaustion.
	bookRateLimit  = 10
	bookRateWindow = time.Minute
	// loginRateLimit and loginRateWindow bound how many magic-link requests a
	// single client may trigger per window. The endpoint sends an email and is
	// unauthenticated, so this blunts email-spam abuse and login enumeration.
	loginRateLimit  = 5
	loginRateWindow = 10 * time.Minute
	// checkoutRateLimit and checkoutRateWindow bound the unauthenticated order /
	// custom-request endpoints, which each create DB documents and a Paystack
	// transaction, to blunt abuse.
	checkoutRateLimit  = 15
	checkoutRateWindow = time.Minute
	// waitlistRateLimit and waitlistRateWindow bound the unauthenticated waitlist
	// signup, which persists a subscriber and sends a welcome email, so it can't
	// be used for subscriber spam or to email-bomb a third-party address.
	waitlistRateLimit  = 10
	waitlistRateWindow = time.Minute
	// maxRateEntries caps the limiter map; expired windows are pruned when the
	// cap is reached so memory stays bounded under address churn.
	maxRateEntries = 10_000
)

type rateWindow struct {
	start time.Time
	count int
}

// rateLimiter is a small fixed-window per-key limiter for unauthenticated
// endpoints with real-world side effects.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	seen   map[string]*rateWindow
	now    func() time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		mu:     sync.Mutex{},
		limit:  limit,
		window: window,
		seen:   map[string]*rateWindow{},
		now:    time.Now,
	}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	current, ok := l.seen[key]
	if !ok || now.Sub(current.start) >= l.window {
		if len(l.seen) >= maxRateEntries {
			l.prune(now)
		}

		l.seen[key] = &rateWindow{start: now, count: 1}

		return true
	}

	current.count++

	return current.count <= l.limit
}

func (l *rateLimiter) prune(now time.Time) {
	for key, window := range l.seen {
		if now.Sub(window.start) >= l.window {
			delete(l.seen, key)
		}
	}
}

// rateLimitByIP rejects requests beyond the per-client budget with 429.
func rateLimitByIP(limiter *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}

			if !limiter.allow(host) {
				respondError(w, http.StatusTooManyRequests, "rate_limited", "too many booking attempts, slow down")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin rejects non-admin users. Must run after RequireAuth.
func (h *Handlers) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok || user.Role != domain.RoleAdmin {
			respondError(w, http.StatusForbidden, "forbidden", msgAdminAccessRequired)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdminArea rejects users who may not enter the admin dashboard at all
// (i.e. customers). It resolves the role from the editable store and stashes
// the definition in the request context so the per-route RequirePermission
// guard reuses it instead of resolving again. Must run after RequireAuth.
func (h *Handlers) RequireAdminArea(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok {
			respondError(w, http.StatusForbidden, "forbidden", msgAdminAccessRequired)

			return
		}

		def, err := h.roleDef(r.Context(), user)
		if err != nil {
			// Fail closed: an unexpected role-store error must not grant access.
			logRequestError(r, err)
			respondError(w, http.StatusForbidden, "forbidden", msgAdminAccessRequired)

			return
		}

		if !def.AdminArea {
			respondError(w, http.StatusForbidden, "forbidden", msgAdminAccessRequired)

			return
		}

		ctx := context.WithValue(r.Context(), roleDefContextKey, def)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission rejects users whose role lacks the given capability,
// resolved from the editable role store (reusing the definition stashed by
// RequireAdminArea when present). Must run after RequireAuth.
func (h *Handlers) RequirePermission(permission domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := userFromContext(r.Context())
			if !ok {
				respondError(w, http.StatusForbidden, "forbidden", msgNoAccess)

				return
			}

			// RequireAdminArea resolves and stashes the role for the admin group;
			// fall back to resolving here for any route guarded standalone.
			def, hasDef := roleDefFromContext(r.Context())
			if !hasDef {
				resolved, err := h.roleDef(r.Context(), user)
				if err != nil {
					logRequestError(r, err)
					respondError(w, http.StatusForbidden, "forbidden", msgNoAccess)

					return
				}

				def = resolved
			}

			if !def.Has(permission) {
				respondError(w, http.StatusForbidden, "forbidden", msgNoAccess)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
