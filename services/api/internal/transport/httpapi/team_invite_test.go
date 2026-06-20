package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminInvitePartner proves the invite flow end to end: an admin invites a
// partner with a dashboard role, and following the emailed link signs that
// partner in with the assigned role.
func TestAdminInvitePartner(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	inv := doJSON(t, http.MethodPost, base+"/admin/invitations",
		`{"email":"partner@e25.com","name":"Partner","role":"staff"}`, admin)
	require.Equal(t, http.StatusCreated, inv.status, "body: %s", inv.body)
	assert.Contains(t, inv.body, `"role":"staff"`)

	// The invite emailed a sign-in link; following it signs the partner in as staff.
	verify := doJSON(t, http.MethodPost, base+"/auth/verify",
		`{"token":"`+env.sender.tokenFromLink(t)+`"}`, nil)
	require.Equal(t, http.StatusOK, verify.status, "body: %s", verify.body)
	assert.Contains(t, verify.body, `"role":"staff"`)

	// A storefront-only role cannot be invited as a partner.
	bad := doJSON(t, http.MethodPost, base+"/admin/invitations",
		`{"email":"shopper@e25.com","name":"Shopper","role":"customer"}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)
}

func TestAdminInvitePartnerRequiresTeamWrite(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	viewer := env.signInAs(t, admin, "viewer@e25.com", "viewer")

	reply := doJSON(t, http.MethodPost, base+"/admin/invitations",
		`{"email":"partner@e25.com","name":"Partner","role":"staff"}`, viewer)
	assert.Equal(t, http.StatusForbidden, reply.status, "viewer lacks team:write")
}
