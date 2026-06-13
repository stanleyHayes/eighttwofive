package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// pagedEnvelope mirrors the {items,total,page,pageSize} data shape returned by
// every paginated admin listing.
type pagedEnvelope struct {
	Data struct {
		Items    []json.RawMessage `json:"items"`
		Total    int64             `json:"total"`
		Page     int               `json:"page"`
		PageSize int               `json:"pageSize"`
	} `json:"data"`
}

func decodePaged(t *testing.T, body string) pagedEnvelope {
	t.Helper()

	var env pagedEnvelope
	require.NoError(t, json.Unmarshal([]byte(body), &env), body)

	return env
}

func TestAdminListCollections_PaginatedEnvelope(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	const seeded = 25
	for i := range seeded {
		createCollection(t, env, admin, fmt.Sprintf("Collection %02d", i))
	}

	// Default page: 20 items, total reflects every collection.
	first := doJSON(t, http.MethodGet, base+"/admin/collections", "", admin)
	require.Equal(t, http.StatusOK, first.status, first.body)

	page1 := decodePaged(t, first.body)
	assert.Equal(t, int64(seeded), page1.Data.Total)
	assert.Equal(t, 1, page1.Data.Page)
	assert.Equal(t, 20, page1.Data.PageSize)
	assert.Len(t, page1.Data.Items, 20)

	// Second page carries the remainder.
	second := doJSON(t, http.MethodGet, base+"/admin/collections?page=2", "", admin)
	require.Equal(t, http.StatusOK, second.status, second.body)

	page2 := decodePaged(t, second.body)
	assert.Equal(t, 2, page2.Data.Page)
	assert.Len(t, page2.Data.Items, seeded-20)
}

func TestAdminListCollections_PageSizeAndInvalidParams(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	for i := range 5 {
		createCollection(t, env, admin, fmt.Sprintf("Coll %02d", i))
	}

	// Explicit pageSize is honored.
	sized := doJSON(t, http.MethodGet, base+"/admin/collections?page=1&pageSize=2", "", admin)
	require.Equal(t, http.StatusOK, sized.status, sized.body)

	page := decodePaged(t, sized.body)
	assert.Equal(t, 2, page.Data.PageSize)
	assert.Len(t, page.Data.Items, 2)

	// Garbage params fall back to the defaults rather than erroring.
	bad := doJSON(t, http.MethodGet, base+"/admin/collections?page=abc&pageSize=-9", "", admin)
	require.Equal(t, http.StatusOK, bad.status, bad.body)

	fallback := decodePaged(t, bad.body)
	assert.Equal(t, 1, fallback.Data.Page)
	assert.Equal(t, 20, fallback.Data.PageSize)

	// pageSize over the cap is clamped to 100.
	capped := doJSON(t, http.MethodGet, base+"/admin/collections?pageSize=500", "", admin)
	require.Equal(t, http.StatusOK, capped.status, capped.body)
	assert.Equal(t, 100, decodePaged(t, capped.body).Data.PageSize)
}

func TestAdminListDesigns_PaginatedAndFilterable(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	collectionID := createCollection(t, env, admin, "Velvet")

	const seeded = 22
	for i := range seeded {
		body := strings.ReplaceAll(designBody, "%COL%", collectionID)
		body = strings.Replace(body, "Boardroom Blazer", fmt.Sprintf("Design %02d", i), 1)

		created := doJSON(t, http.MethodPost, base+"/admin/designs", body, admin)
		require.Equal(t, http.StatusCreated, created.status, created.body)
	}

	first := doJSON(t, http.MethodGet, base+"/admin/designs?page=1&pageSize=20", "", admin)
	require.Equal(t, http.StatusOK, first.status, first.body)

	page1 := decodePaged(t, first.body)
	assert.Equal(t, int64(seeded), page1.Data.Total)
	assert.Len(t, page1.Data.Items, 20)

	// Filtering by collection still returns the paginated envelope.
	filtered := doJSON(t, http.MethodGet, base+"/admin/designs?collection="+collectionID, "", admin)
	require.Equal(t, http.StatusOK, filtered.status, filtered.body)
	assert.Equal(t, int64(seeded), decodePaged(t, filtered.body).Data.Total)
}

func TestAdminListWaitlist_PaginatedEnvelope(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	for i := range 3 {
		status := postJSON(t, base+"/waitlist",
			fmt.Sprintf(`{"email":"sub%d@example.com","name":"Sub %d"}`, i, i))
		require.Equal(t, http.StatusCreated, status)
	}

	reply := doJSON(t, http.MethodGet, base+"/admin/waitlist", "", admin)
	require.Equal(t, http.StatusOK, reply.status, reply.body)

	page := decodePaged(t, reply.body)
	assert.Equal(t, int64(3), page.Data.Total)
	assert.Equal(t, 1, page.Data.Page)
	assert.Equal(t, 20, page.Data.PageSize)
	assert.Len(t, page.Data.Items, 3)
}

func TestAdminListOrders_PaginatedEnvelope(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	const seeded = 23
	for i := range seeded {
		createdAt := time.Now().UTC().Add(time.Duration(i) * time.Second)
		err := env.orders.Create(t.Context(), &domain.Order{
			Ref:            fmt.Sprintf("E25-PAGE-%02d", i),
			CustomerID:     "user-1",
			DesignID:       "000000000000000000000002",
			DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PricePesewas: 50000},
			Type:           domain.OrderTypeStandard,
			Customisation:  domain.Customisation{SizeMode: "band", BandLabel: "8"},
			Delivery:       domain.Delivery{Mode: "pickup"},
			Status:         domain.OrderStatusBooked,
			CustomerPhone:  "+233200000000",
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
		})
		require.NoError(t, err)
	}

	first := doJSON(t, http.MethodGet, base+"/admin/orders?page=1&pageSize=20", "", admin)
	require.Equal(t, http.StatusOK, first.status, first.body)

	page1 := decodePaged(t, first.body)
	assert.Equal(t, int64(seeded), page1.Data.Total)
	assert.Equal(t, 20, page1.Data.PageSize)
	assert.Len(t, page1.Data.Items, 20)

	second := doJSON(t, http.MethodGet, base+"/admin/orders?page=2&pageSize=20", "", admin)
	require.Equal(t, http.StatusOK, second.status, second.body)
	assert.Len(t, decodePaged(t, second.body).Data.Items, seeded-20)
}
