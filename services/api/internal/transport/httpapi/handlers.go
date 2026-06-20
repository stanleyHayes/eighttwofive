package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// healthCheckTimeout bounds the dependency ping behind /healthz so a wedged
// database makes the check fail fast instead of hanging the probe.
const healthCheckTimeout = 2 * time.Second

// HealthPinger verifies that a critical dependency (the database) is reachable.
// It is optional: when nil, /healthz reports liveness only.
type HealthPinger interface {
	Ping(ctx context.Context) error
}

// pagedDTO is the envelope data shape for every paginated admin listing.
type pagedDTO[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// pageQuery reads the 1-based page and pageSize query parameters, falling back
// to the shared defaults. Non-numeric or out-of-range values are clamped by the
// domain layer, so unparseable input simply yields the defaults.
func pageQuery(r *http.Request) (int, int) {
	page := parseIntQuery(r, "page", domain.DefaultPage)
	pageSize := parseIntQuery(r, "pageSize", domain.DefaultPageSize)

	return page, pageSize
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

// Handlers holds the use-cases exposed over HTTP.
type Handlers struct {
	waitlist      *service.Waitlist
	auth          *service.Auth
	settings      *service.StoreSettings
	catalog       *service.Catalog
	orders        *service.Order
	analytics     *service.Analytics
	roles         *service.Roles
	slots         *service.CalendarSlot
	visits        *service.CalendarVisit
	signer        domain.UploadSigner // nil when uploads are not configured
	cloudName     string
	secureCookies bool
	health        HealthPinger // nil when no readiness dependency is wired
}

// NewHandlers wires HTTP handlers to application services.
func NewHandlers(
	waitlist *service.Waitlist,
	auth *service.Auth,
	settings *service.StoreSettings,
	catalog *service.Catalog,
	orders *service.Order,
	analytics *service.Analytics,
	roles *service.Roles,
	slots *service.CalendarSlot,
	visits *service.CalendarVisit,
	signer domain.UploadSigner,
	cloudName string,
	secureCookies bool,
	health HealthPinger,
) *Handlers {
	return &Handlers{
		waitlist:      waitlist,
		auth:          auth,
		settings:      settings,
		catalog:       catalog,
		orders:        orders,
		analytics:     analytics,
		roles:         roles,
		slots:         slots,
		visits:        visits,
		signer:        signer,
		cloudName:     cloudName,
		secureCookies: secureCookies,
		health:        health,
	}
}

type subscriberDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func toSubscriberDTO(s domain.Subscriber) subscriberDTO {
	return subscriberDTO{ID: s.ID, Email: s.Email, Name: s.Name, CreatedAt: s.CreatedAt}
}

// Health reports readiness: it pings the database (when one is wired) so the
// platform routes traffic away from an instance that has lost its store.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if h.health != nil {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		err := h.health.Ping(ctx)
		if err != nil {
			respondError(w, http.StatusServiceUnavailable, "unavailable", "database unreachable")

			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type joinRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// JoinWaitlist handles POST /api/v1/waitlist.
func (h *Handlers) JoinWaitlist(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req joinRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "request body must be valid JSON")

		return
	}

	sub, err := h.waitlist.Join(r.Context(), req.Email, req.Name)

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrDuplicateEmail):
		respondError(w, http.StatusConflict, "conflict", "this email is already on the waitlist")
	case err != nil:
		respondInternal(w, r, err)
	default:
		respondJSON(w, http.StatusCreated, toSubscriberDTO(*sub))
	}
}

// ListWaitlist handles GET /api/v1/admin/waitlist, paginated (page, pageSize).
func (h *Handlers) ListWaitlist(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pageQuery(r)

	result, err := h.waitlist.ListPaged(r.Context(), page, pageSize)
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	dtos := make([]subscriberDTO, 0, len(result.Items))
	for _, s := range result.Items {
		dtos = append(dtos, toSubscriberDTO(s))
	}

	respondJSON(w, http.StatusOK, pagedDTO[subscriberDTO]{
		Items:    dtos,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// AdminDeleteSubscriber handles DELETE /api/v1/admin/waitlist/{id}.
func (h *Handlers) AdminDeleteSubscriber(w http.ResponseWriter, r *http.Request) {
	err := h.waitlist.Delete(r.Context(), chi.URLParam(r, "id"))

	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "subscriber not found")
	case err != nil:
		respondInternal(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

type signUploadResponse struct {
	CloudName string `json:"cloudName"`
	APIKey    string `json:"apiKey"`
	Timestamp int64  `json:"timestamp"`
	Folder    string `json:"folder"`
	Signature string `json:"signature"`
}

type signUploadRequest struct {
	Folder string `json:"folder"`
}

// SignUpload handles POST /api/v1/uploads/sign for direct-to-Cloudinary uploads.
func (h *Handlers) SignUpload(w http.ResponseWriter, r *http.Request) {
	if h.signer == nil {
		respondError(w, http.StatusServiceUnavailable, "not_configured", "uploads are not configured")

		return
	}

	var req signUploadRequest
	if !decodeBody(w, r, &req) {
		return
	}

	folder := strings.TrimSpace(req.Folder)
	if folder == "" {
		folder = "eightfivetwo"
	}

	sig, err := h.signer.SignUpload(folder)
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, signUploadResponse{
		CloudName: sig.CloudName,
		APIKey:    sig.APIKey,
		Timestamp: sig.Timestamp,
		Folder:    sig.Folder,
		Signature: sig.Signature,
	})
}
