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

// Permissions returns the fixed catalogue of capabilities the app enforces.
func (s *Roles) Permissions() []domain.PermissionMeta {
	return domain.AllPermissionsMeta()
}
