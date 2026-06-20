package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type roleDTO struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	System      bool     `json:"system"`
	AdminArea   bool     `json:"adminArea"`
}

func toRoleDTO(r domain.RoleDef) roleDTO {
	perms := make([]string, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, string(p))
	}

	return roleDTO{
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
		System:      r.System,
		AdminArea:   r.AdminArea,
	}
}

type permissionDTO struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Group       string `json:"group"`
}

// AdminListRoles handles GET /api/v1/admin/roles — every role definition.
func (h *Handlers) AdminListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.roles.List(r.Context())
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	dtos := make([]roleDTO, 0, len(roles))
	for _, role := range roles {
		dtos = append(dtos, toRoleDTO(role))
	}

	respondJSON(w, http.StatusOK, dtos)
}

// AdminListPermissions handles GET /api/v1/admin/permissions — the fixed
// catalogue of capabilities a role can grant.
func (h *Handlers) AdminListPermissions(w http.ResponseWriter, _ *http.Request) {
	metas := h.roles.Permissions()

	dtos := make([]permissionDTO, 0, len(metas))
	for _, m := range metas {
		dtos = append(dtos, permissionDTO{
			Key:         string(m.Key),
			Label:       m.Label,
			Description: m.Description,
			Group:       m.Group,
		})
	}

	respondJSON(w, http.StatusOK, dtos)
}

type roleWriteRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	AdminArea   bool     `json:"adminArea"`
}

func (req roleWriteRequest) toInput() service.RoleInput {
	perms := make([]domain.Permission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, domain.Permission(p))
	}

	return service.RoleInput{
		Name:        req.Name,
		Description: req.Description,
		Permissions: perms,
		AdminArea:   req.AdminArea,
	}
}

// AdminCreateRole handles POST /api/v1/admin/roles — create a custom role.
func (h *Handlers) AdminCreateRole(w http.ResponseWriter, r *http.Request) {
	var req roleWriteRequest
	if !decodeBody(w, r, &req) {
		return
	}

	role, err := h.roles.Create(r.Context(), req.toInput())
	if err != nil {
		respondRoleError(w, r, err)

		return
	}

	respondJSON(w, http.StatusCreated, toRoleDTO(*role))
}

// AdminUpdateRole handles PUT /api/v1/admin/roles/{key} — retune a role.
func (h *Handlers) AdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	var req roleWriteRequest
	if !decodeBody(w, r, &req) {
		return
	}

	role, err := h.roles.Update(r.Context(), chi.URLParam(r, "key"), req.toInput())
	if err != nil {
		respondRoleError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toRoleDTO(*role))
}

// AdminDeleteRole handles DELETE /api/v1/admin/roles/{key} — remove a custom
// role (built-in roles are protected).
func (h *Handlers) AdminDeleteRole(w http.ResponseWriter, r *http.Request) {
	err := h.roles.Delete(r.Context(), chi.URLParam(r, "key"))
	if err != nil {
		respondRoleError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func respondRoleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "role not found")
	case errors.Is(err, domain.ErrSystemRole):
		respondError(w, http.StatusConflict, "system_role", "built-in roles cannot be deleted")
	case errors.Is(err, domain.ErrDuplicateRole):
		respondError(w, http.StatusConflict, "conflict", "a role with that name already exists")
	default:
		respondInternal(w, r, err)
	}
}
