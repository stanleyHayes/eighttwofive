package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	maxRoleName        = 60
	maxRoleDescription = 280
	maxRoleKeyRetries  = 50
)

// RoleInput is the editable shape of a role, used by Create and Update. The
// key is derived (Create) or taken from the path (Update), never from input.
type RoleInput struct {
	Name        string
	Description string
	Permissions []domain.Permission
	AdminArea   bool
}

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

// Create adds a custom role with a unique, slugified key.
func (s *Roles) Create(ctx context.Context, input RoleInput) (*domain.RoleDef, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	err := validateRoleInput(input)
	if err != nil {
		return nil, err
	}

	key, err := s.uniqueKey(ctx, input.Name)
	if err != nil {
		return nil, err
	}

	def := &domain.RoleDef{
		Key:         key,
		Name:        input.Name,
		Description: input.Description,
		Permissions: dedupePermissions(input.Permissions),
		System:      false,
		AdminArea:   input.AdminArea,
	}

	err = s.repo.Upsert(ctx, def)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	return def, nil
}

// Update retunes an existing role's name, description and permissions. A
// built-in role keeps its key, System flag and dashboard-access flag; the
// admin role additionally always keeps every permission, so an admin can never
// edit away their own ability to manage roles and the team (the recovery path).
func (s *Roles) Update(ctx context.Context, key string, input RoleInput) (*domain.RoleDef, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	err := validateRoleInput(input)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("load role: %w", err)
	}

	updated := &domain.RoleDef{
		Key:         existing.Key,
		Name:        input.Name,
		Description: input.Description,
		Permissions: dedupePermissions(input.Permissions),
		System:      existing.System,
		AdminArea:   input.AdminArea,
	}

	if existing.System {
		updated.AdminArea = existing.AdminArea
	}

	if existing.Key == string(domain.RoleAdmin) {
		updated.Permissions = domain.RoleAdmin.Permissions()
		updated.AdminArea = true
	}

	err = s.repo.Upsert(ctx, updated)
	if err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	return updated, nil
}

// Delete removes a custom role. Built-in roles are protected (ErrSystemRole).
// Users still holding a deleted role fall back to no access (fail-safe) until
// they are reassigned.
func (s *Roles) Delete(ctx context.Context, key string) error {
	existing, err := s.repo.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("load role: %w", err)
	}

	if existing.System {
		return domain.ErrSystemRole
	}

	err = s.repo.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}

	return nil
}

// uniqueKey slugifies the name and appends -2, -3, … until a free key is found.
func (s *Roles) uniqueKey(ctx context.Context, name string) (string, error) {
	base := slugify(name)

	for attempt := 1; attempt <= maxRoleKeyRetries; attempt++ {
		key := base
		if attempt > 1 {
			key = base + "-" + strconv.Itoa(attempt)
		}

		_, err := s.repo.Get(ctx, key)
		if errors.Is(err, domain.ErrNotFound) {
			return key, nil
		}

		if err != nil {
			return "", fmt.Errorf("check role key: %w", err)
		}
	}

	return "", fmt.Errorf("%w: no free key for %q", domain.ErrDuplicateRole, name)
}

func validateRoleInput(input RoleInput) error {
	if input.Name == "" || len(input.Name) > maxRoleName {
		return fmt.Errorf("%w: name must be 1-%d characters", domain.ErrInvalidInput, maxRoleName)
	}

	if len(input.Description) > maxRoleDescription {
		return fmt.Errorf("%w: description must be at most %d characters", domain.ErrInvalidInput, maxRoleDescription)
	}

	valid := make(map[domain.Permission]bool, len(domain.AllPermissionsMeta()))
	for _, meta := range domain.AllPermissionsMeta() {
		valid[meta.Key] = true
	}

	for _, perm := range input.Permissions {
		if !valid[perm] {
			return fmt.Errorf("%w: unknown permission %q", domain.ErrInvalidInput, perm)
		}
	}

	return nil
}

func dedupePermissions(perms []domain.Permission) []domain.Permission {
	seen := make(map[domain.Permission]bool, len(perms))
	out := make([]domain.Permission, 0, len(perms))

	for _, perm := range perms {
		if seen[perm] {
			continue
		}

		seen[perm] = true

		out = append(out, perm)
	}

	return out
}
