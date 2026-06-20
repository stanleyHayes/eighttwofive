package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminDeleteSubscriber(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")

	// Add a subscriber through the public waitlist.
	require.Equal(t, http.StatusCreated,
		postJSON(t, base+"/waitlist", `{"email":"ada@example.com","name":"Ada"}`))

	// Find its id through the admin listing.
	list := doJSON(t, http.MethodGet, base+"/admin/waitlist", "", admin)
	require.Equal(t, http.StatusOK, list.status, "body: %s", list.body)

	var listed struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(list.body), &listed))
	require.Len(t, listed.Data.Items, 1)
	id := listed.Data.Items[0].ID

	// An admin removes it.
	del := doJSON(t, http.MethodDelete, base+"/admin/waitlist/"+id, "", admin)
	assert.Equal(t, http.StatusNoContent, del.status, "body: %s", del.body)

	// It is gone, so deleting again is a 404.
	again := doJSON(t, http.MethodDelete, base+"/admin/waitlist/"+id, "", admin)
	assert.Equal(t, http.StatusNotFound, again.status)
}

func TestAdminDeleteSubscriberRequiresWrite(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"
	admin := env.signIn(t, "boss@e25.com")
	viewer := env.signInAs(t, admin, "viewer@e25.com", "viewer")

	// A viewer can read the waitlist but not delete from it.
	reply := doJSON(t, http.MethodDelete, base+"/admin/waitlist/id-someone", "", viewer)
	assert.Equal(t, http.StatusForbidden, reply.status)
}
