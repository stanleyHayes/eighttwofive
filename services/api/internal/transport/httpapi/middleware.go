package httpapi

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const sessionCookieName = "e25_session"

type contextKey int

const userContextKey contextKey = iota

func userFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)

	return user, ok
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
			respondError(w, http.StatusForbidden, "forbidden", "admin access required")

			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdminArea rejects users who may not enter the admin dashboard at all
// (i.e. customers). Must run after RequireAuth.
func (h *Handlers) RequireAdminArea(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok || !user.Role.IsAdminArea() {
			respondError(w, http.StatusForbidden, "forbidden", "admin access required")

			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequirePermission rejects users whose role lacks the given capability. Must
// run after RequireAuth.
func (h *Handlers) RequirePermission(permission domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := userFromContext(r.Context())
			if !ok || !user.Role.Has(permission) {
				respondError(w, http.StatusForbidden, "forbidden", "you don't have access to this")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
