package httpapi_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	base := env.srv.URL + "/api/v1"

	// Request a sign-in link.
	status := postJSON(t, base+"/auth/request-link", `{"email":"Ama@Example.com","name":"Ama"}`)
	require.Equal(t, http.StatusAccepted, status)
	require.Contains(t, env.sender.lastLink, "http://test.local/auth/verify?token=")

	// Exchange the token for a session.
	token := env.sender.tokenFromLink(t)
	verify := doJSON(t, http.MethodPost, base+"/auth/verify", `{"token":"`+token+`"}`, nil)
	require.Equal(t, http.StatusOK, verify.status)
	assert.Contains(t, verify.body, `"ama@example.com"`, "email must be normalized")
	assert.Contains(t, verify.body, `"customer"`)

	var session *http.Cookie

	for _, cookie := range verify.cookies {
		if cookie.Name == "e25_session" {
			session = cookie
		}
	}

	require.NotNil(t, session, "verify must set the session cookie")
	assert.True(t, session.HttpOnly)

	// The session authenticates /auth/me.
	me := doJSON(t, http.MethodGet, base+"/auth/me", "", session)
	assert.Equal(t, http.StatusOK, me.status)
	assert.Contains(t, me.body, `"ama@example.com"`)

	// A login token is single-use.
	reuse := doJSON(t, http.MethodPost, base+"/auth/verify", `{"token":"`+token+`"}`, nil)
	assert.Equal(t, http.StatusUnauthorized, reuse.status)

	// Logout revokes the session.
	logout := doJSON(t, http.MethodPost, base+"/auth/logout", "", session)
	require.Equal(t, http.StatusNoContent, logout.status)

	after := doJSON(t, http.MethodGet, base+"/auth/me", "", session)
	assert.Equal(t, http.StatusUnauthorized, after.status)
}

func TestAuthMe_Unauthenticated(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	reply := doJSON(t, http.MethodGet, env.srv.URL+"/api/v1/auth/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, reply.status)
}

func TestAuthVerify_InvalidToken(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	reply := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/verify", `{"token":"bogus"}`, nil)
	assert.Equal(t, http.StatusUnauthorized, reply.status)
}

func TestRequestLink_InvalidEmail(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	status := postJSON(t, env.srv.URL+"/api/v1/auth/request-link", `{"email":"nope","name":"X"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestRequestLink_EmailUnavailable(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.sender.loginErr = errEmailDown

	// A delivery failure reports a distinct 502/email_unavailable so the customer
	// is told to retry, not shown a generic crash.
	reply := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/request-link",
		`{"email":"ama@example.com","name":"Ama"}`, nil)
	assert.Equal(t, http.StatusBadGateway, reply.status)
	assert.Contains(t, reply.body, `"code":"email_unavailable"`)
}

var errEmailDown = errors.New("resend unavailable")

func TestRequestLink_EmptyName(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	status := postJSON(t, env.srv.URL+"/api/v1/auth/request-link", `{"email":"ama@example.com","name":"  "}`)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestAdminAccess(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	// A customer session cannot reach admin routes.
	customer := env.signIn(t, "ama@example.com")
	denied := doJSON(t, http.MethodGet, base+"/admin/waitlist", "", customer)
	assert.Equal(t, http.StatusForbidden, denied.status)

	// No session at all is unauthorized.
	anonymous := doJSON(t, http.MethodGet, base+"/admin/waitlist", "", nil)
	assert.Equal(t, http.StatusUnauthorized, anonymous.status)

	// The allowlisted email signs in as admin.
	admin := env.signIn(t, "boss@e25.com")
	allowed := doJSON(t, http.MethodGet, base+"/admin/waitlist", "", admin)
	assert.Equal(t, http.StatusOK, allowed.status)

	// Admin updates settings; the public endpoint reflects it.
	updated := doJSON(t, http.MethodPut, base+"/admin/settings", `{
		"depositPesewas": 60000,
		"whatsappNumber": "+233200000000",
		"visitLocation": "Osu, Accra",
		"deliveryRates": [{"area": "Accra", "ratePesewas": 1000}]
	}`, admin)
	require.Equal(t, http.StatusOK, updated.status)

	public := doJSON(t, http.MethodGet, base+"/settings", "", nil)
	require.Equal(t, http.StatusOK, public.status)
	assert.Contains(t, public.body, `"depositPesewas":60000`)
	assert.Contains(t, public.body, "Osu, Accra")
	assert.Contains(t, public.body, `"area":"Accra"`)

	// Customers cannot update settings.
	forbidden := doJSON(t, http.MethodPut, base+"/admin/settings", `{"depositPesewas":1}`, customer)
	assert.Equal(t, http.StatusForbidden, forbidden.status)
}

func TestSettings_Defaults(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	reply := doJSON(t, http.MethodGet, env.srv.URL+"/api/v1/settings", "", nil)
	require.Equal(t, http.StatusOK, reply.status)
	assert.Contains(t, reply.body, `"depositPesewas":50000`, "default deposit is GHS 500")
	assert.Contains(t, reply.body, `"deliveryRates":[]`)
}

func TestSettings_Update_RejectsDuplicateArea(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	admin := env.signIn(t, "boss@e25.com")
	reply := doJSON(t, http.MethodPut, base+"/admin/settings", `{
		"depositPesewas": 50000,
		"whatsappNumber": "+233200000000",
		"visitLocation": "Accra",
		"deliveryRates": [
			{"area": "Accra", "ratePesewas": 1000},
			{"area": "Accra", "ratePesewas": 2000}
		]
	}`, admin)

	require.Equal(t, http.StatusConflict, reply.status)
	assert.Contains(t, reply.body, "duplicate_area")
}

func TestSettings_Update_RejectsInvalidDeliveryRate(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	admin := env.signIn(t, "boss@e25.com")
	reply := doJSON(t, http.MethodPut, base+"/admin/settings", `{
		"depositPesewas": 50000,
		"whatsappNumber": "+233200000000",
		"visitLocation": "Accra",
		"deliveryRates": [{"area": "", "ratePesewas": 1000}]
	}`, admin)

	require.Equal(t, http.StatusUnprocessableEntity, reply.status)
	assert.Contains(t, reply.body, "invalid_input")
}
