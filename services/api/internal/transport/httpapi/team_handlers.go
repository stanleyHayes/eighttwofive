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
		respondError(w, http.StatusInternalServerError, "internal", "something went wrong")

		return
	}

	items := make([]userDTO, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, h.toUserDTO(&result.Items[i]))
	}

	respondJSON(w, http.StatusOK, pagedDTO[userDTO]{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
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
			respondError(w, http.StatusInternalServerError, "internal", "something went wrong")
		}

		return
	}

	respondJSON(w, http.StatusOK, map[string]userDTO{"user": h.toUserDTO(user)})
}
