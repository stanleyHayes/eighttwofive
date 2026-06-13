package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const sessionCookieMaxAge = 30 * 24 * time.Hour

type userDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role)}
}

type requestLinkRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// RequestLoginLink handles POST /api/v1/auth/request-link.
func (h *Handlers) RequestLoginLink(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req requestLinkRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "request body must be valid JSON")

		return
	}

	err = h.auth.RequestLink(r.Context(), req.Email, req.Name)

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case err != nil:
		respondError(w, http.StatusInternalServerError, "internal", "something went wrong")
	default:
		respondJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
	}
}

type verifyRequest struct {
	Token string `json:"token"`
}

// VerifyLogin handles POST /api/v1/auth/verify — exchanges a one-time link
// token for a session cookie.
func (h *Handlers) VerifyLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req verifyRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "request body must be valid JSON")

		return
	}

	sessionToken, user, err := h.auth.Verify(r.Context(), req.Token)

	switch {
	case errors.Is(err, domain.ErrTokenInvalid):
		respondError(w, http.StatusUnauthorized, "invalid_token", "this sign-in link is invalid or has expired")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "internal", "something went wrong")
	default:
		h.setSessionCookie(w, sessionToken, int(sessionCookieMaxAge.Seconds()))
		respondJSON(w, http.StatusOK, map[string]userDTO{"user": toUserDTO(user)})
	}
}

// Me handles GET /api/v1/auth/me (behind RequireAuth).
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

		return
	}

	respondJSON(w, http.StatusOK, map[string]userDTO{"user": toUserDTO(user)})
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		h.auth.Logout(r.Context(), cookie.Value)
	}

	h.setSessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}
