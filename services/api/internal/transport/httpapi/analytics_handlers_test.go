package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminGetAnalytics_RequiresAdmin(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")

	// Unauthenticated request is rejected.
	reply := doJSON(t, http.MethodGet, env.srv.URL+"/api/v1/admin/analytics", "", nil)
	require.Equal(t, http.StatusUnauthorized, reply.status)

	// Non-admin customer is rejected.
	customer := env.signIn(t, "customer@example.com")
	reply = doJSON(t, http.MethodGet, env.srv.URL+"/api/v1/admin/analytics", "", customer)
	require.Equal(t, http.StatusForbidden, reply.status)
}

func TestAdminGetAnalytics_ReturnsShape(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	reply := doJSON(t, http.MethodGet, env.srv.URL+"/api/v1/admin/analytics", "", admin)
	require.Equal(t, http.StatusOK, reply.status)

	var body struct {
		Data struct {
			WaitlistCount   int64            `json:"waitlistCount"`
			CustomerCount   int64            `json:"customerCount"`
			OrdersByStatus  map[string]int64 `json:"ordersByStatus"`
			OrdersByType    map[string]int64 `json:"ordersByType"`
			RevenuePesewas  int64            `json:"revenuePesewas"`
			CollectionViews int64            `json:"collectionViews"`
		} `json:"data"`
	}

	require.NoError(t, json.Unmarshal([]byte(reply.body), &body))

	assert.Equal(t, int64(3), body.Data.WaitlistCount)
	assert.Equal(t, int64(5), body.Data.CustomerCount)
	assert.Equal(t, int64(2), body.Data.OrdersByStatus["booked"])
	assert.Equal(t, int64(2), body.Data.OrdersByType["standard"])
	assert.Equal(t, int64(250_000), body.Data.RevenuePesewas)
}

func TestAdminGetAnalytics_ExposesEnrichedFields(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	reply := doJSON(t, http.MethodGet, env.srv.URL+"/api/v1/admin/analytics", "", admin)
	require.Equal(t, http.StatusOK, reply.status)

	var body struct {
		Data struct {
			RevenuePesewas           int64 `json:"revenuePesewas"`
			OrderCount               int64 `json:"orderCount"`
			BookedRevenuePesewas     int64 `json:"bookedRevenuePesewas"`
			AverageOrderValuePesewas int64 `json:"averageOrderValuePesewas"`
			Comparison               struct {
				RevenueChangeBps    int64 `json:"revenueChangeBps"`
				OrderCountChangeBps int64 `json:"orderCountChangeBps"`
			} `json:"comparison"`
			RevenueSeries []struct {
				Label          string `json:"label"`
				RevenuePesewas int64  `json:"revenuePesewas"`
				OrderCount     int64  `json:"orderCount"`
			} `json:"revenueSeries"`
			TopDesigns []struct {
				DesignID       string `json:"designId"`
				Name           string `json:"name"`
				OrderCount     int64  `json:"orderCount"`
				RevenuePesewas int64  `json:"revenuePesewas"`
			} `json:"topDesigns"`
			TopCollections []struct {
				CollectionID   string `json:"collectionId"`
				Name           string `json:"name"`
				RevenuePesewas int64  `json:"revenuePesewas"`
			} `json:"topCollections"`
			RecentOrders []struct {
				Ref          string `json:"ref"`
				Type         string `json:"type"`
				Status       string `json:"status"`
				TotalPesewas int64  `json:"totalPesewas"`
			} `json:"recentOrders"`
		} `json:"data"`
	}

	require.NoError(t, json.Unmarshal([]byte(reply.body), &body))

	// The shared fake repo carries the headline revenue; assert it still
	// surfaces and that the enriched arrays decode to non-nil collections.
	assert.Equal(t, int64(250_000), body.Data.RevenuePesewas)
	assert.NotNil(t, body.Data.RevenueSeries)
	assert.NotNil(t, body.Data.TopDesigns)
	assert.NotNil(t, body.Data.TopCollections)
	assert.NotNil(t, body.Data.RecentOrders)
}
