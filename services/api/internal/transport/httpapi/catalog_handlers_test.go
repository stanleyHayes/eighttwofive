package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const designBody = `{
	"collectionId": "%COL%",
	"name": "Boardroom Blazer",
	"note": "tailored",
	"photos": [{"publicId": "e25/x", "order": 0}],
	"sizeBands": [{"label": "8", "pricePesewas": 50000, "chart": {"bust": "86 cm"}}]
}`

func createCollection(t *testing.T, env *testEnv, admin *http.Cookie, name string) string {
	t.Helper()

	reply := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/admin/collections",
		`{"name":"`+name+`","note":""}`, admin)
	require.Equal(t, http.StatusCreated, reply.status, reply.body)

	var payload struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}

	require.NoError(t, json.Unmarshal([]byte(reply.body), &payload))

	return payload.Data.ID
}

func TestCatalogAdminCRUDAndPublicVisibility(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	// Admin routes are guarded.
	denied := doJSON(t, http.MethodPost, base+"/admin/collections", `{"name":"X"}`, nil)
	assert.Equal(t, http.StatusUnauthorized, denied.status)

	// Create a collection and a design in it.
	collectionID := createCollection(t, env, admin, "Velvet")
	body := strings.ReplaceAll(designBody, "%COL%", collectionID)

	created := doJSON(t, http.MethodPost, base+"/admin/designs", body, admin)
	require.Equal(t, http.StatusCreated, created.status, created.body)
	assert.Contains(t, created.body, `"slug":"boardroom-blazer"`)

	var design struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}

	require.NoError(t, json.Unmarshal([]byte(created.body), &design))

	// Public storefront sees the live design and collection.
	public := doJSON(t, http.MethodGet, base+"/designs/"+design.Data.Slug, "", nil)
	assert.Equal(t, http.StatusOK, public.status)
	assert.Contains(t, public.body, `"pricePesewas":50000`)

	publicList := doJSON(t, http.MethodGet, base+"/collections", "", nil)
	assert.Equal(t, http.StatusOK, publicList.status)
	assert.Contains(t, publicList.body, "Velvet")

	// Validation errors surface as 422.
	invalidBody := strings.ReplaceAll(body, `"pricePesewas": 50000`, `"pricePesewas": 0`)
	invalidBody = strings.ReplaceAll(invalidBody, "Boardroom Blazer", "Other Name")
	invalid := doJSON(t, http.MethodPost, base+"/admin/designs", invalidBody, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, invalid.status)

	// The admin single-design endpoint serves any status.
	adminGet := doJSON(t, http.MethodGet, base+"/admin/designs/"+design.Data.ID, "", admin)
	assert.Equal(t, http.StatusOK, adminGet.status)
	assert.Contains(t, adminGet.body, `"boardroom-blazer"`)

	// Retire the collection — design disappears from the storefront.
	retire := doJSON(t, http.MethodPost, base+"/admin/collections/"+collectionID+"/retire", "", admin)
	require.Equal(t, http.StatusOK, retire.status)

	hidden := doJSON(t, http.MethodGet, base+"/designs/"+design.Data.Slug, "", nil)
	assert.Equal(t, http.StatusNotFound, hidden.status)

	hiddenList := doJSON(t, http.MethodGet, base+"/collections", "", nil)
	assert.NotContains(t, hiddenList.body, "Velvet")

	// Restoring the design alone is blocked while its collection is retired.
	blocked := doJSON(t, http.MethodPost, base+"/admin/designs/restore",
		`{"ids":["`+design.Data.ID+`"]}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, blocked.status)
	assert.Contains(t, blocked.body, "restore the collection first")

	// Restore the collection — everything is public again.
	restore := doJSON(t, http.MethodPost, base+"/admin/collections/"+collectionID+"/restore", "", admin)
	require.Equal(t, http.StatusOK, restore.status)

	visible := doJSON(t, http.MethodGet, base+"/designs/"+design.Data.Slug, "", nil)
	assert.Equal(t, http.StatusOK, visible.status)

	// Permanent delete removes the design for good.
	deleted := doJSON(t, http.MethodDelete, base+"/admin/designs/"+design.Data.ID, "", admin)
	require.Equal(t, http.StatusNoContent, deleted.status)

	gone := doJSON(t, http.MethodGet, base+"/designs/"+design.Data.Slug, "", nil)
	assert.Equal(t, http.StatusNotFound, gone.status)
}

func TestCatalogSlugConflictSuffix(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	first := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/admin/collections", `{"name":"Velvet"}`, admin)
	require.Equal(t, http.StatusCreated, first.status)
	assert.Contains(t, first.body, `"slug":"velvet"`)

	second := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/admin/collections", `{"name":"Velvet"}`, admin)
	require.Equal(t, http.StatusCreated, second.status)
	assert.Contains(t, second.body, `"slug":"velvet-2"`)
}

func TestAdminUpdateCollection_ReturnsUpdatedResource(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	id := createCollection(t, env, admin, "Velvet")
	updated := doJSON(t, http.MethodPut, env.srv.URL+"/api/v1/admin/collections/"+id,
		`{"name":"Velvet Edit","note":" refreshed note "}`, admin)
	require.Equal(t, http.StatusOK, updated.status, updated.body)
	assert.Contains(t, updated.body, `"name":"Velvet Edit"`)
	assert.Contains(t, updated.body, `"note":"refreshed note"`)
	assert.Contains(t, updated.body, `"slug":"velvet"`)
}
