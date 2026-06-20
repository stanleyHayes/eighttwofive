package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// userID looks up a user's id by email through the admin users listing.
func (e *testEnv) userID(t *testing.T, admin *http.Cookie, email string) string {
	t.Helper()

	reply := doJSON(t, http.MethodGet, e.srv.URL+"/api/v1/admin/users", "", admin)
	require.Equal(t, http.StatusOK, reply.status, "body: %s", reply.body)

	var resp struct {
		Data struct {
			Items []struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(reply.body), &resp))

	for _, item := range resp.Data.Items {
		if item.Email == email {
			return item.ID
		}
	}

	t.Fatalf("user %q not found in admin listing", email)

	return ""
}

// signInAs signs a fresh user in (created as a customer) and has the admin
// promote them to the given role, returning that user's session cookie. The
// session loads the user fresh on each request, so the new role takes effect
// immediately on the returned cookie.
func (e *testEnv) signInAs(t *testing.T, admin *http.Cookie, email, role string) *http.Cookie {
	t.Helper()

	cookie := e.signIn(t, email)
	id := e.userID(t, admin, email)

	reply := doJSON(t, http.MethodPut,
		e.srv.URL+"/api/v1/admin/users/"+id+"/role", `{"role":"`+role+`"}`, admin)
	require.Equal(t, http.StatusOK, reply.status, "promote to %s: %s", role, reply.body)

	return cookie
}

func TestRBAC_StaffPermissions(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	staff := env.signInAs(t, admin, "staff@e25.com", "staff")

	// Staff may read and write the catalogue.
	read := doJSON(t, http.MethodGet, base+"/admin/collections", "", staff)
	assert.Equal(t, http.StatusOK, read.status, "staff reads catalogue")

	create := doJSON(t, http.MethodPost, base+"/admin/collections",
		`{"name":"Harmattan","note":""}`, staff)
	assert.Equal(t, http.StatusCreated, create.status, "staff writes catalogue: %s", create.body)

	// Staff may NOT delete the catalogue (admin-only). The guard rejects before
	// the handler, so a non-existent id still surfaces the permission boundary.
	del := doJSON(t, http.MethodDelete, base+"/admin/collections/anything", "", staff)
	assert.Equal(t, http.StatusForbidden, del.status, "staff cannot delete catalogue")

	// Staff may NOT touch settings or the team.
	settings := doJSON(t, http.MethodPut, base+"/admin/settings", `{"depositPesewas":1}`, staff)
	assert.Equal(t, http.StatusForbidden, settings.status, "staff cannot write settings")

	teamRead := doJSON(t, http.MethodGet, base+"/admin/users", "", staff)
	assert.Equal(t, http.StatusForbidden, teamRead.status, "staff cannot read the team")

	teamWrite := doJSON(t, http.MethodPut, base+"/admin/users/x/role", `{"role":"viewer"}`, staff)
	assert.Equal(t, http.StatusForbidden, teamWrite.status, "staff cannot assign roles")
}

func TestRBAC_ViewerPermissions(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	viewer := env.signInAs(t, admin, "viewer@e25.com", "viewer")

	// Viewer may read across the dashboard.
	orders := doJSON(t, http.MethodGet, base+"/admin/orders", "", viewer)
	assert.Equal(t, http.StatusOK, orders.status, "viewer reads orders")

	collections := doJSON(t, http.MethodGet, base+"/admin/collections", "", viewer)
	assert.Equal(t, http.StatusOK, collections.status, "viewer reads catalogue")

	// Viewer may NOT write anything.
	create := doJSON(t, http.MethodPost, base+"/admin/collections",
		`{"name":"Nope","note":""}`, viewer)
	assert.Equal(t, http.StatusForbidden, create.status, "viewer cannot write catalogue")

	status := doJSON(t, http.MethodPost, base+"/admin/orders/E25-X/status",
		`{"status":"booked"}`, viewer)
	assert.Equal(t, http.StatusForbidden, status.status, "viewer cannot write orders")

	slot := doJSON(t, http.MethodPost, base+"/admin/slots", createSlotPayload(24*time.Hour, 25*time.Hour), viewer)
	assert.Equal(t, http.StatusForbidden, slot.status, "viewer cannot create slots")
}

func TestRBAC_CustomerBlockedFromAdminArea(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	customer := env.signIn(t, "ama@example.com")

	// A customer cannot enter the admin area at all, even read-only routes.
	reply := doJSON(t, http.MethodGet, base+"/admin/collections", "", customer)
	assert.Equal(t, http.StatusForbidden, reply.status)
}

func TestAdminSetUserRole(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	// Create a user (as customer) then promote them.
	env.signIn(t, "kofi@e25.com")
	id := env.userID(t, admin, "kofi@e25.com")

	promote := doJSON(t, http.MethodPut, base+"/admin/users/"+id+"/role", `{"role":"staff"}`, admin)
	require.Equal(t, http.StatusOK, promote.status, "body: %s", promote.body)
	assert.Contains(t, promote.body, `"role":"staff"`)

	// An unknown role is rejected.
	bad := doJSON(t, http.MethodPut, base+"/admin/users/"+id+"/role", `{"role":"wizard"}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)

	// An unknown user is a 404.
	missing := doJSON(t, http.MethodPut, base+"/admin/users/nobody/role", `{"role":"viewer"}`, admin)
	assert.Equal(t, http.StatusNotFound, missing.status)
}

// TestRBAC_DynamicRoleEnforcement proves the HTTP layer enforces permissions
// from the editable role store, not the static enum: editing a role's stored
// permissions changes access on the very next request, with no restart. This
// is the contract Phase 3's role editor relies on.
func TestRBAC_DynamicRoleEnforcement(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	staff := env.signInAs(t, admin, "staff@e25.com", "staff")

	// Baseline mirrors the seeded built-in staff matrix: writes the catalogue
	// (has catalogue:write) but cannot read the team (lacks team:read).
	create := doJSON(t, http.MethodPost, base+"/admin/collections", `{"name":"Baseline","note":""}`, staff)
	require.Equal(t, http.StatusCreated, create.status, "staff writes catalogue by default: %s", create.body)

	teamBefore := doJSON(t, http.MethodGet, base+"/admin/users", "", staff)
	require.Equal(t, http.StatusForbidden, teamBefore.status, "staff cannot read the team by default")

	// Persist an edited staff role — exactly what a Phase 3 admin edit stores:
	// revoke catalogue:write, grant team:read.
	require.NoError(t, env.roleStore.Upsert(t.Context(), &domain.RoleDef{
		Key: "staff", Name: "Staff", AdminArea: true,
		Permissions: []domain.Permission{
			domain.PermAnalyticsRead, domain.PermOrdersRead, domain.PermOrdersWrite,
			domain.PermSlotsRead, domain.PermSlotsWrite, domain.PermSubscribersRead,
			domain.PermCatalogueRead, // catalogue:write revoked
			domain.PermTeamRead,      // team:read granted
		},
	}))

	// Enforcement follows the store on the next request, with no restart.
	writeNow := doJSON(t, http.MethodPost, base+"/admin/collections", `{"name":"Blocked","note":""}`, staff)
	assert.Equal(t, http.StatusForbidden, writeNow.status, "the DB edit must revoke staff's catalogue write")

	teamNow := doJSON(t, http.MethodGet, base+"/admin/users", "", staff)
	assert.Equal(t, http.StatusOK, teamNow.status, "the DB edit must grant staff team:read: %s", teamNow.body)
}

// TestRBAC_MissingRoleFallsBackToStatic proves a role absent from the store
// (a cold database before seeding, or a removed entry) falls back to the
// built-in static matrix, so a built-in role is never locked out.
func TestRBAC_MissingRoleFallsBackToStatic(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	staff := env.signInAs(t, admin, "staff@e25.com", "staff")

	// Drop the staff role from the store entirely.
	require.NoError(t, env.roleStore.Delete(t.Context(), "staff"))

	// Staff still reaches read routes through the static fallback.
	read := doJSON(t, http.MethodGet, base+"/admin/collections", "", staff)
	assert.Equal(t, http.StatusOK, read.status, "a missing role must fall back to the built-in matrix: %s", read.body)
}
