package httpapi

import (
	"net/http"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
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
