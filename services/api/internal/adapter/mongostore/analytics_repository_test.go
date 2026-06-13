package mongostore_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func TestAnalyticsRepository_GetStoreAnalytics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	ctx := context.Background()

	orders := mongostore.NewOrderRepository(db)
	users := mongostore.NewUserRepository(db)
	subscribers := mongostore.NewSubscriberRepository(db)
	analytics := mongostore.NewAnalyticsRepository(db)

	require.NoError(t, orders.EnsureIndexes(ctx))
	require.NoError(t, users.EnsureIndexes(ctx))
	require.NoError(t, subscribers.EnsureIndexes(ctx))
	require.NoError(t, analytics.EnsureIndexes(ctx))

	// Seed waitlist subscribers.
	for i, name := range []string{"Ama", "Esi", "Naa"} {
		err := subscribers.Create(ctx, &domain.Subscriber{
			Email:     name + "@example.com",
			Name:      name,
			CreatedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour),
		})
		require.NoError(t, err)
	}

	// Seed customers and one admin.
	for _, email := range []string{"customer1@example.com", "customer2@example.com"} {
		require.NoError(t, users.Upsert(ctx, &domain.User{
			Email: email,
			Name:  "Customer",
			Role:  domain.RoleCustomer,
		}))
	}

	require.NoError(t, users.Upsert(ctx, &domain.User{
		Email: "admin@example.com",
		Name:  "Admin",
		Role:  domain.RoleAdmin,
	}))

	// Seed orders: one booked standard order and one requested custom request.
	require.NoError(t, orders.Create(ctx, &domain.Order{
		Ref:            "E25-BOOKED",
		CustomerID:     "000000000000000000000001",
		DesignID:       "000000000000000000000002",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "e25/blazer", PricePesewas: 100_000},
		Type:           domain.OrderTypeStandard,
		Customisation:  domain.Customisation{SizeMode: "band", BandLabel: "8"},
		Delivery:       domain.Delivery{Mode: "pickup"},
		Status:         domain.OrderStatusBooked,
		StatusHistory:  []domain.StatusChange{{Status: domain.OrderStatusBooked, At: time.Now().UTC(), By: "system"}},
		CustomerPhone:  "+233200000000",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}))

	require.NoError(t, orders.Create(ctx, &domain.Order{
		Ref:            "E25-REQUEST",
		CustomerID:     "000000000000000000000001",
		DesignID:       "000000000000000000000002",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "e25/blazer", PricePesewas: 0},
		Type:           domain.OrderTypeCustomSize,
		Customisation:  domain.Customisation{SizeMode: "self", Measurements: map[string]string{"bust": "90 cm"}},
		Delivery:       domain.Delivery{Mode: "pickup"},
		Status:         domain.OrderStatusRequested,
		StatusHistory:  []domain.StatusChange{{Status: domain.OrderStatusRequested, At: time.Now().UTC(), By: "customer"}},
		CustomerPhone:  "+233200000000",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}))

	got, err := analytics.GetStoreAnalytics(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(3), got.WaitlistCount)
	assert.Equal(t, int64(2), got.CustomerCount)
	assert.Equal(t, int64(1), got.OrdersByStatus[string(domain.OrderStatusBooked)])
	assert.Equal(t, int64(1), got.OrdersByStatus[string(domain.OrderStatusRequested)])
	assert.Equal(t, int64(1), got.OrdersByType[string(domain.OrderTypeStandard)])
	assert.Equal(t, int64(1), got.OrdersByType[string(domain.OrderTypeCustomSize)])
	assert.Equal(t, int64(100_000), got.RevenuePesewas)
	assert.Equal(t, int64(100_000), got.BookedRevenuePesewas)
	assert.Equal(t, int64(1), got.OrderCount)
	assert.Equal(t, int64(100_000), got.AverageOrderValuePesewas)
	assert.Equal(t, int64(0), got.CollectionViews)
	assert.Len(t, got.RevenueSeries, 12)
	assert.Len(t, got.RecentOrders, 2)
}

// seededCatalog records the collection and design IDs created for the enriched
// analytics test so orders can reference them by the IDs Mongo assigned.
type seededCatalog struct {
	heritageCollectionID string
	studioCollectionID   string
	blazerDesignID       string
	dressDesignID        string
}

func seedCatalog(ctx context.Context, t *testing.T, db *mongo.Database) seededCatalog {
	t.Helper()

	collections := mongostore.NewCollectionRepository(db)
	designs := mongostore.NewDesignRepository(db)

	heritage := &domain.Collection{
		Name: "Heritage", Slug: "heritage", Status: domain.StatusLive, CreatedAt: time.Now().UTC(),
	}
	studio := &domain.Collection{
		Name: "Studio", Slug: "studio", Status: domain.StatusLive, CreatedAt: time.Now().UTC(),
	}

	require.NoError(t, collections.Create(ctx, heritage))
	require.NoError(t, collections.Create(ctx, studio))

	blazer := &domain.Design{
		CollectionID: heritage.ID,
		Name:         "Blazer",
		Slug:         "blazer",
		Status:       domain.StatusLive,
		SizeBands:    []domain.SizeBand{{Label: "8", PricePesewas: 100_000}},
		CreatedAt:    time.Now().UTC(),
	}
	dress := &domain.Design{
		CollectionID: studio.ID,
		Name:         "Dress",
		Slug:         "dress",
		Status:       domain.StatusLive,
		SizeBands:    []domain.SizeBand{{Label: "10", PricePesewas: 200_000}},
		CreatedAt:    time.Now().UTC(),
	}

	require.NoError(t, designs.Create(ctx, blazer))
	require.NoError(t, designs.Create(ctx, dress))

	return seededCatalog{
		heritageCollectionID: heritage.ID,
		studioCollectionID:   studio.ID,
		blazerDesignID:       blazer.ID,
		dressDesignID:        dress.ID,
	}
}

func bookedOrder(ref, designID, name string, price int64, at time.Time) *domain.Order {
	return &domain.Order{
		Ref:            ref,
		CustomerID:     "000000000000000000000001",
		DesignID:       designID,
		DesignSnapshot: domain.DesignSnapshot{Name: name, PhotoPublicID: "e25/" + name, PricePesewas: price},
		Type:           domain.OrderTypeStandard,
		Customisation:  domain.Customisation{SizeMode: "band", BandLabel: "8"},
		Delivery:       domain.Delivery{Mode: "pickup"},
		Status:         domain.OrderStatusBooked,
		StatusHistory:  []domain.StatusChange{{Status: domain.OrderStatusBooked, At: at, By: "system"}},
		CustomerPhone:  "+233200000000",
		CreatedAt:      at,
		UpdatedAt:      at,
	}
}

func TestAnalyticsRepository_EnrichedAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	ctx := context.Background()

	orders := mongostore.NewOrderRepository(db)
	analytics := mongostore.NewAnalyticsRepository(db)

	require.NoError(t, orders.EnsureIndexes(ctx))
	require.NoError(t, analytics.EnsureIndexes(ctx))

	cat := seedCatalog(ctx, t, db)
	now := time.Now().UTC()
	hour := time.Hour
	day := 24 * time.Hour

	// Blazer (Heritage): three booked orders, two recent, one in the prior
	// 30-day window. Dress (Studio): one recent booked order at a higher price.
	b1 := bookedOrder("E25-B1", cat.blazerDesignID, "Blazer", 100_000, now.Add(-1*hour))
	b2 := bookedOrder("E25-B2", cat.blazerDesignID, "Blazer", 100_000, now.Add(-2*hour))
	b3 := bookedOrder("E25-B3", cat.blazerDesignID, "Blazer", 100_000, now.Add(-40*day))
	d1 := bookedOrder("E25-D1", cat.dressDesignID, "Dress", 200_000, now.Add(-3*hour))

	require.NoError(t, orders.Create(ctx, b1))
	require.NoError(t, orders.Create(ctx, b2))
	require.NoError(t, orders.Create(ctx, b3))
	require.NoError(t, orders.Create(ctx, d1))

	got, err := analytics.GetStoreAnalytics(ctx)
	require.NoError(t, err)

	// Headline: four booked orders totalling GH₵5,000 (500_000 pesewas).
	assert.Equal(t, int64(4), got.OrderCount)
	assert.Equal(t, int64(500_000), got.BookedRevenuePesewas)
	assert.Equal(t, int64(125_000), got.AverageOrderValuePesewas)

	// Time series spans 12 weekly buckets with the recent revenue in the last.
	require.Len(t, got.RevenueSeries, 12)
	assert.Equal(t, int64(400_000), got.RevenueSeries[11].RevenuePesewas)
	assert.Equal(t, int64(3), got.RevenueSeries[11].OrderCount)

	// Comparison: three recent booked orders vs one in the prior window.
	assert.Equal(t, int64(400_000), got.Comparison.CurrentRevenuePesewas)
	assert.Equal(t, int64(100_000), got.Comparison.PriorRevenuePesewas)
	assert.Equal(t, int64(3), got.Comparison.CurrentOrderCount)
	assert.Equal(t, int64(1), got.Comparison.PriorOrderCount)
	assert.Equal(t, int64(30_000), got.Comparison.RevenueChangeBps)
	assert.Equal(t, int64(20_000), got.Comparison.OrderCountChangeBps)

	// Top designs: Blazer leads by order count (3), Dress follows (1).
	require.Len(t, got.TopDesigns, 2)
	assert.Equal(t, "Blazer", got.TopDesigns[0].Name)
	assert.Equal(t, int64(3), got.TopDesigns[0].OrderCount)
	assert.Equal(t, int64(300_000), got.TopDesigns[0].RevenuePesewas)
	assert.Equal(t, "Dress", got.TopDesigns[1].Name)

	// Top collections: Heritage leads by revenue (GH₵3,000) over Studio.
	require.Len(t, got.TopCollections, 2)
	assert.Equal(t, "Heritage", got.TopCollections[0].Name)
	assert.Equal(t, int64(300_000), got.TopCollections[0].RevenuePesewas)
	assert.Equal(t, "Studio", got.TopCollections[1].Name)
	assert.Equal(t, int64(200_000), got.TopCollections[1].RevenuePesewas)

	// Recent orders are newest-first and capped at eight.
	require.Len(t, got.RecentOrders, 4)
	assert.Equal(t, "E25-B1", got.RecentOrders[0].Ref)
	assert.Equal(t, int64(100_000), got.RecentOrders[0].TotalPesewas)
}
