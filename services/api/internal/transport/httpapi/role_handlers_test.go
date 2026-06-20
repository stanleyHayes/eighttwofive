package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminListRolesAndPermissions(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	// The four built-in roles are listed, flagged as system (protected).
	roles := doJSON(t, http.MethodGet, base+"/admin/roles", "", admin)
	require.Equal(t, http.StatusOK, roles.status, "body: %s", roles.body)
	assert.Contains(t, roles.body, `"key":"admin"`)
	assert.Contains(t, roles.body, `"key":"staff"`)
	assert.Contains(t, roles.body, `"key":"viewer"`)
	assert.Contains(t, roles.body, `"key":"customer"`)
	assert.Contains(t, roles.body, `"system":true`)

	// The permission catalogue exposes every enforced capability.
	perms := doJSON(t, http.MethodGet, base+"/admin/permissions", "", admin)
	require.Equal(t, http.StatusOK, perms.status)
	assert.Contains(t, perms.body, `"key":"team:write"`)
	assert.Contains(t, perms.body, `"key":"catalogue:delete"`)
	assert.Contains(t, perms.body, `"group":"Team"`)
}

func TestAdminListRoles_RequiresTeamRead(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	viewer := env.signInAs(t, admin, "viewer@e25.com", "viewer")

	// Viewer can enter the dashboard but lacks team:read, so the roles API is 403.
	reply := doJSON(t, http.MethodGet, base+"/admin/roles", "", viewer)
	assert.Equal(t, http.StatusForbidden, reply.status)
}
