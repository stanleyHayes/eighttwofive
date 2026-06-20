package service

import (
	"context"
	"fmt"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// Roles exposes the editable role definitions and the fixed permission
// catalogue that the team-management UI lists toggles from.
type Roles struct {
	repo domain.RoleRepository
}

// NewRoles wires the roles service.
func NewRoles(repo domain.RoleRepository) *Roles {
	return &Roles{repo: repo}
}

// List returns every role definition (built-in and custom).
func (s *Roles) List(ctx context.Context) ([]domain.RoleDef, error) {
	roles, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	return roles, nil
}

// Resolve returns the stored definition for a role key. The transport layer
// uses it to enforce permissions from the editable roles instead of a static
// map, so an admin's role edit changes access immediately, with no redeploy.
// The store is tiny and admin traffic is low, so this reads through to the
// repository on each call rather than caching (which keeps edits instant and
// avoids any staleness window).
func (s *Roles) Resolve(ctx context.Context, key string) (*domain.RoleDef, error) {
	def, err := s.repo.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("resolve role %q: %w", key, err)
	}

	return def, nil
}

// Permissions returns the fixed catalogue of capabilities the app enforces.
func (s *Roles) Permissions() []domain.PermissionMeta {
	return domain.AllPermissionsMeta()
}
