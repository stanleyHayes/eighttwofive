package domain

import (
	"context"
	"errors"
	"slices"
	"time"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// Role is a user's access level. Customers use only the storefront; the three
// admin-area roles (viewer, staff, admin) have escalating dashboard access.
type Role string

const (
	// RoleCustomer can access only the public storefront.
	RoleCustomer Role = "customer"
	// RoleViewer is read-only across the admin dashboard.
	RoleViewer Role = "viewer"
	// RoleStaff handles day-to-day operations (orders, slots) but not settings,
	// catalogue changes, or team management.
	RoleStaff Role = "staff"
	// RoleAdmin has full access, including settings and team management.
	RoleAdmin Role = "admin"
)

// Permission is a single capability checked by the transport layer.
type Permission string

// Permission identifiers checked per route by RequirePermission.
const (
	PermAnalyticsRead   Permission = "analytics:read"
	PermOrdersRead      Permission = "orders:read"
	PermOrdersWrite     Permission = "orders:write"
	PermSlotsRead       Permission = "slots:read"
	PermSlotsWrite      Permission = "slots:write"
	PermCatalogueRead   Permission = "catalogue:read"
	PermCatalogueWrite  Permission = "catalogue:write"
	PermCatalogueDelete Permission = "catalogue:delete"
	PermSubscribersRead Permission = "subscribers:read"
	PermSettingsWrite   Permission = "settings:write"
	PermTeamRead        Permission = "team:read"
	PermTeamWrite       Permission = "team:write"
)

// allPermissions is the full capability set, granted to admins.
func allPermissions() []Permission {
	return []Permission{
		PermAnalyticsRead, PermOrdersRead, PermOrdersWrite, PermSlotsRead, PermSlotsWrite,
		PermCatalogueRead, PermCatalogueWrite, PermCatalogueDelete, PermSubscribersRead,
		PermSettingsWrite, PermTeamRead, PermTeamWrite,
	}
}

// Permissions returns the capabilities granted to this role.
func (r Role) Permissions() []Permission {
	switch r {
	case RoleAdmin:
		return allPermissions()
	case RoleStaff:
		return []Permission{
			PermAnalyticsRead, PermOrdersRead, PermOrdersWrite,
			PermSlotsRead, PermSlotsWrite, PermSubscribersRead,
			PermCatalogueRead, PermCatalogueWrite,
		}
	case RoleViewer:
		return []Permission{
			PermAnalyticsRead, PermOrdersRead, PermSlotsRead, PermSubscribersRead, PermCatalogueRead,
		}
	case RoleCustomer:
		return nil
	default:
		return nil
	}
}

// Has reports whether the role grants the given permission.
func (r Role) Has(p Permission) bool {
	return slices.Contains(r.Permissions(), p)
}

// IsAdminArea reports whether the role may enter the admin dashboard at all.
func (r Role) IsAdminArea() bool {
	return r == RoleAdmin || r == RoleStaff || r == RoleViewer
}

// User is a person with an account — created lightly, at the last step of
// completing an order (scope §4.8), or on first sign-in.
type User struct {
	ID        string
	Email     string
	Name      string
	Role      Role
	CreatedAt time.Time
}

// UserRepository is the persistence port for users.
type UserRepository interface {
	// Upsert creates the user if the email is new and backfills ID, Role,
	// Name and CreatedAt from the stored document otherwise. An admin role
	// on the input promotes an existing user; it never demotes.
	Upsert(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	// Count returns the total number of users.
	Count(ctx context.Context) (int64, error)
	// ListPaged returns one page of users, newest first.
	ListPaged(ctx context.Context, params PageParams) ([]User, error)
	// UpdateRole sets a user's role unconditionally (team management).
	UpdateRole(ctx context.Context, id string, role Role) error
}
