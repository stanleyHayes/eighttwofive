package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminRoleCRUD(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	// Create a custom role; its key is slugified from the name.
	create := doJSON(t, http.MethodPost, base+"/admin/roles",
		`{"name":"Photographer","description":"Shoots the lookbook","permissions":["catalogue:read"],"adminArea":true}`,
		admin)
	require.Equal(t, http.StatusCreated, create.status, "body: %s", create.body)
	assert.Contains(t, create.body, `"key":"photographer"`)
	assert.Contains(t, create.body, `"system":false`)
	assert.Contains(t, create.body, `"description":"Shoots the lookbook"`)

	// It appears in the listing.
	list := doJSON(t, http.MethodGet, base+"/admin/roles", "", admin)
	assert.Contains(t, list.body, `"key":"photographer"`)

	// Rename and retune it.
	upd := doJSON(t, http.MethodPut, base+"/admin/roles/photographer",
		`{"name":"Lookbook Photographer","permissions":["catalogue:read"],"adminArea":true}`, admin)
	require.Equal(t, http.StatusOK, upd.status, "body: %s", upd.body)
	assert.Contains(t, upd.body, `"name":"Lookbook Photographer"`)

	// An unknown permission is rejected.
	bad := doJSON(t, http.MethodPost, base+"/admin/roles",
		`{"name":"Bad","permissions":["not:a:permission"]}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)

	// Updating a role that doesn't exist is a 404.
	missing := doJSON(t, http.MethodPut, base+"/admin/roles/ghost", `{"name":"Ghost"}`, admin)
	assert.Equal(t, http.StatusNotFound, missing.status)

	// Built-in roles cannot be deleted.
	delSystem := doJSON(t, http.MethodDelete, base+"/admin/roles/staff", "", admin)
	assert.Equal(t, http.StatusConflict, delSystem.status)

	// The custom role can be deleted.
	del := doJSON(t, http.MethodDelete, base+"/admin/roles/photographer", "", admin)
	assert.Equal(t, http.StatusNoContent, del.status)

	gone := doJSON(t, http.MethodGet, base+"/admin/roles", "", admin)
	assert.NotContains(t, gone.body, `"key":"photographer"`)
}

// TestAdminCustomRoleAssignableAndEnforced is the end-to-end proof of Phase 3:
// a brand-new custom role (unknown to the static enum) can be assigned to a
// user and is then enforced by the middleware exactly as defined.
func TestAdminCustomRoleAssignableAndEnforced(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	// A custom admin-area role that edits the catalogue but not the team.
	create := doJSON(t, http.MethodPost, base+"/admin/roles",
		`{"name":"Photographer","permissions":["catalogue:read","catalogue:write"],"adminArea":true}`, admin)
	require.Equal(t, http.StatusCreated, create.status, "body: %s", create.body)

	// Assigning the custom role succeeds — SetUserRole validates against the
	// store, not the static enum.
	shooter := env.signInAs(t, admin, "shooter@e25.com", "photographer")

	// Its permissions are enforced: catalogue write yes, team read no.
	write := doJSON(t, http.MethodPost, base+"/admin/collections", `{"name":"Lookbook","note":""}`, shooter)
	assert.Equal(t, http.StatusCreated, write.status, "custom role grants catalogue:write: %s", write.body)

	team := doJSON(t, http.MethodGet, base+"/admin/users", "", shooter)
	assert.Equal(t, http.StatusForbidden, team.status, "custom role lacks team:read")
}

// TestAdminDeletedCustomRoleFailsSafe proves that deleting a role still held by
// a user denies that user (fail-safe via the Phase 2 fallback) rather than
// crashing or silently keeping access.
func TestAdminDeletedCustomRoleFailsSafe(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	create := doJSON(t, http.MethodPost, base+"/admin/roles",
		`{"name":"Photographer","permissions":["catalogue:read"],"adminArea":true}`, admin)
	require.Equal(t, http.StatusCreated, create.status, "body: %s", create.body)

	shooter := env.signInAs(t, admin, "shooter@e25.com", "photographer")

	// While the role exists the user can read the catalogue.
	before := doJSON(t, http.MethodGet, base+"/admin/collections", "", shooter)
	require.Equal(t, http.StatusOK, before.status, "body: %s", before.body)

	// Delete the role out from under the user.
	del := doJSON(t, http.MethodDelete, base+"/admin/roles/photographer", "", admin)
	require.Equal(t, http.StatusNoContent, del.status)

	// The user fails safe: a missing custom role resolves to no admin-area
	// access, so they are denied (403) — not granted, and not a 500.
	after := doJSON(t, http.MethodGet, base+"/admin/collections", "", shooter)
	assert.Equal(t, http.StatusForbidden, after.status, "deleted role must fail safe to deny: %s", after.body)
}

func TestAdminCreateRoleRejectsBlankName(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	reply := doJSON(t, http.MethodPost, base+"/admin/roles",
		`{"name":"   ","permissions":["catalogue:read"]}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, reply.status, "a blank name must be rejected: %s", reply.body)
}

func TestAdminRoleWriteRequiresTeamWrite(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	staff := env.signInAs(t, admin, "staff@e25.com", "staff")

	// Staff can read roles but not create them (team:write is admin-only).
	create := doJSON(t, http.MethodPost, base+"/admin/roles",
		`{"name":"X","permissions":["catalogue:read"]}`, staff)
	assert.Equal(t, http.StatusForbidden, create.status)
}

// TestAdminRoleProtectsAdmin proves an admin can't edit away their own ability
// to manage roles and the team: the admin role always keeps every permission
// and dashboard access, so there is no self-lockout.
func TestAdminRoleProtectsAdmin(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	// Try to strip the admin role down to read-only and out of the dashboard.
	upd := doJSON(t, http.MethodPut, base+"/admin/roles/admin",
		`{"name":"Admin","permissions":["catalogue:read"],"adminArea":false}`, admin)
	require.Equal(t, http.StatusOK, upd.status, "body: %s", upd.body)
	assert.Contains(t, upd.body, `"team:write"`, "admin keeps every permission")
	assert.Contains(t, upd.body, `"adminArea":true`, "admin keeps dashboard access")

	// And the admin can still manage the team afterwards.
	team := doJSON(t, http.MethodGet, base+"/admin/users", "", admin)
	assert.Equal(t, http.StatusOK, team.status)
}
