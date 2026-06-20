package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// AdminListUsers handles GET /api/v1/admin/users, paginated (page, pageSize).
func (h *Handlers) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pageQuery(r)

	result, err := h.auth.ListUsers(r.Context(), page, pageSize)
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	items := make([]userDTO, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, h.toUserDTO(r.Context(), &result.Items[i]))
	}

	respondJSON(w, http.StatusOK, pagedDTO[userDTO]{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

type inviteRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// AdminInvitePartner handles POST /api/v1/admin/invitations — provision a team
// member with a dashboard role and email them a sign-in link.
func (h *Handlers) AdminInvitePartner(w http.ResponseWriter, r *http.Request) {
	var req inviteRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, err := h.auth.InvitePartner(r.Context(), req.Email, req.Name, domain.Role(req.Role))

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrEmailSendFailed):
		logRequestError(r, err)
		respondError(w, http.StatusBadGateway, "email_unavailable",
			"We couldn't send the invite right now. Please try again in a moment.")
	case err != nil:
		respondInternal(w, r, err)
	default:
		respondJSON(w, http.StatusCreated, map[string]userDTO{"user": h.toUserDTO(r.Context(), user)})
	}
}

type setRoleRequest struct {
	Role string `json:"role"`
}

// AdminSetUserRole handles PUT /api/v1/admin/users/{id}/role.
func (h *Handlers) AdminSetUserRole(w http.ResponseWriter, r *http.Request) {
	var req setRoleRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, err := h.auth.SetUserRole(r.Context(), chi.URLParam(r, "id"), domain.Role(req.Role))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "not_found", "user not found")
		default:
			respondInternal(w, r, err)
		}

		return
	}

	respondJSON(w, http.StatusOK, map[string]userDTO{"user": h.toUserDTO(r.Context(), user)})
}
