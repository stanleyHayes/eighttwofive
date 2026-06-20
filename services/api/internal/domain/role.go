package domain

import (
	"context"
	"errors"
	"slices"
)

// ErrSystemRole is returned when an operation would delete or re-key a built-in
// role, which the dashboard must never allow.
var ErrSystemRole = errors.New("built-in role cannot be modified that way")

// ErrDuplicateRole is returned when a new role's key collides with an existing
// one and no free variant could be derived.
var ErrDuplicateRole = errors.New("role key already exists")

// RoleDef is a named, editable bundle of permissions. The four built-in roles
// (customer/viewer/staff/admin) are seeded as System roles — their key can't
// change and they can't be deleted — while admins may add custom roles with any
// permission set.
type RoleDef struct {
	Key         string       // stable identifier, e.g. "admin" or "photographer"
	Name        string       // display label
	Description string       // shown in the team UI
	Permissions []Permission // the capabilities this role grants
	System      bool         // built-in; cannot be deleted, its key is fixed
	AdminArea   bool         // may enter the admin dashboard at all
}

// Has reports whether the role definition grants a permission.
func (r *RoleDef) Has(p Permission) bool {
	return slices.Contains(r.Permissions, p)
}

// RoleRepository is the persistence port for role definitions.
type RoleRepository interface {
	List(ctx context.Context) ([]RoleDef, error)
	Get(ctx context.Context, key string) (*RoleDef, error)
	Upsert(ctx context.Context, r *RoleDef) error
	Delete(ctx context.Context, key string) error
}

// PermissionMeta describes a permission for the role-management UI: a stable
// key plus a human label, a one-line description, and a grouping.
type PermissionMeta struct {
	Key         Permission
	Label       string
	Description string
	Group       string
}

// AllPermissionsMeta is the fixed catalogue of capabilities the app enforces.
// Roles compose these; new entries appear only when the code adds an
// enforcement point, so this is the single source the UI lists toggles from.
func AllPermissionsMeta() []PermissionMeta {
	return []PermissionMeta{
		{PermOrdersRead, "View orders", "See orders and their details", "Orders"},
		{PermOrdersWrite, "Manage orders", "Quote, mark paid, change status, send links", "Orders"},
		{PermSlotsRead, "View visits", "See booking slots and home visits", "Visits"},
		{PermSlotsWrite, "Manage visits", "Open/close slots, reschedule and cancel visits", "Visits"},
		{PermCatalogueRead, "View catalogue", "See collections and designs", "Catalogue"},
		{PermCatalogueWrite, "Edit catalogue", "Create and edit collections and designs", "Catalogue"},
		{PermCatalogueDelete, "Delete catalogue", "Permanently delete collections and designs", "Catalogue"},
		{PermAnalyticsRead, "View analytics", "See the store dashboard and metrics", "Insights"},
		{PermSubscribersRead, "View subscribers", "See and export the waitlist", "Insights"},
		{PermSubscribersWrite, "Manage subscribers", "Remove people from the waitlist", "Insights"},
		{PermSettingsWrite, "Edit settings", "Change deposit, delivery rates and contact details", "Store"},
		{PermTeamRead, "View team", "See team members and their roles", "Team"},
		{PermTeamWrite, "Manage team", "Invite members, assign roles, edit roles & permissions", "Team"},
	}
}

// BuiltInRoles are the seeded, protected role definitions. They mirror the
// static defaults exactly, so seeding them changes no existing behaviour.
func BuiltInRoles() []RoleDef {
	return []RoleDef{
		{
			Key:         string(RoleAdmin),
			Name:        "Admin",
			Description: "Full access, including settings and team management.",
			Permissions: RoleAdmin.Permissions(),
			System:      true,
			AdminArea:   true,
		},
		{
			Key:         string(RoleStaff),
			Name:        "Staff",
			Description: "Day-to-day operations: orders, visits and the catalogue.",
			Permissions: RoleStaff.Permissions(),
			System:      true,
			AdminArea:   true,
		},
		{
			Key:         string(RoleViewer),
			Name:        "Viewer",
			Description: "Read-only access across the dashboard.",
			Permissions: RoleViewer.Permissions(),
			System:      true,
			AdminArea:   true,
		},
		{
			Key:         string(RoleCustomer),
			Name:        "Customer",
			Description: "Storefront only — no dashboard access.",
			Permissions: RoleCustomer.Permissions(),
			System:      true,
			AdminArea:   false,
		},
	}
}
