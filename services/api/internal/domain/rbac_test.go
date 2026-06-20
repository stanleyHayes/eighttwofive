package domain_test

import (
	"testing"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func TestRolePermissions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role domain.Role
		perm domain.Permission
		want bool
	}{
		{domain.RoleAdmin, domain.PermSettingsWrite, true},
		{domain.RoleAdmin, domain.PermTeamWrite, true},
		{domain.RoleStaff, domain.PermOrdersWrite, true},
		{domain.RoleStaff, domain.PermCatalogueWrite, true},
		{domain.RoleStaff, domain.PermCatalogueDelete, false},
		{domain.RoleAdmin, domain.PermCatalogueDelete, true},
		{domain.RoleStaff, domain.PermSettingsWrite, false},
		{domain.RoleStaff, domain.PermTeamRead, false},
		{domain.RoleViewer, domain.PermOrdersRead, true},
		{domain.RoleViewer, domain.PermOrdersWrite, false},
		{domain.RoleViewer, domain.PermAnalyticsRead, true},
		{domain.RoleCustomer, domain.PermAnalyticsRead, false},
	}

	for _, c := range cases {
		got := c.role.Has(c.perm)
		if got != c.want {
			t.Errorf("%s.Has(%s) = %v, want %v", c.role, c.perm, got, c.want)
		}
	}
}

func TestRoleIsAdminArea(t *testing.T) {
	t.Parallel()

	if domain.RoleCustomer.IsAdminArea() {
		t.Error("customer must not be admin-area")
	}

	for _, r := range []domain.Role{domain.RoleViewer, domain.RoleStaff, domain.RoleAdmin} {
		if !r.IsAdminArea() {
			t.Errorf("%s must be admin-area", r)
		}
	}
}
