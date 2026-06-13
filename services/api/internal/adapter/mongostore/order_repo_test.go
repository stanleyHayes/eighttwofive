package mongostore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func newOrder(ref string, typ domain.OrderType) *domain.Order {
	statusChange := domain.StatusChange{
		Status: domain.OrderStatusPendingPayment,
		At:     time.Now().UTC(),
		By:     "customer",
	}

	return &domain.Order{
		Ref:            ref,
		CustomerID:     "000000000000000000000001",
		DesignID:       "000000000000000000000002",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "e25/blazer", PricePesewas: 50000},
		Type:           typ,
		Customisation:  domain.Customisation{SizeMode: "band", BandLabel: "8"},
		Delivery:       domain.Delivery{Mode: "pickup"},
		Payments:       nil,
		Status:         domain.OrderStatusPendingPayment,
		StatusHistory:  []domain.StatusChange{statusChange},
		CustomerPhone:  "+233200000000",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

func TestOrderRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewOrderRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	// Create and round-trip.
	order := newOrder("E25-ABC", domain.OrderTypeStandard)
	require.NoError(t, repo.Create(ctx, order))
	require.NotEmpty(t, order.ID)

	loaded, err := repo.GetByRef(ctx, "E25-ABC")
	require.NoError(t, err)
	assert.Equal(t, order.ID, loaded.ID)
	assert.Equal(t, domain.OrderTypeStandard, loaded.Type)
	assert.Equal(t, int64(50000), loaded.DesignSnapshot.PricePesewas)

	// Unique ref rejects duplicates.
	dup := newOrder("E25-ABC", domain.OrderTypeCustomSize)
	require.ErrorIs(t, repo.Create(ctx, dup), domain.ErrDuplicateRef)

	// Update status.
	loaded.Status = domain.OrderStatusBooked
	loaded.StatusHistory = append(loaded.StatusHistory, domain.StatusChange{
		Status: domain.OrderStatusBooked,
		At:     time.Now().UTC(),
		By:     "payment_webhook",
	})
	require.NoError(t, repo.Update(ctx, loaded))

	reloaded, err := repo.GetByID(ctx, loaded.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, reloaded.Status)
	require.Len(t, reloaded.StatusHistory, 2)

	// List by customer.
	require.NoError(t, repo.Create(ctx, newOrder("E25-DEF", domain.OrderTypeStandard)))

	customerOrders, err := repo.ListByCustomer(ctx, "000000000000000000000001")
	require.NoError(t, err)
	require.Len(t, customerOrders, 2)

	// Admin list sorted by type then createdAt desc.
	require.NoError(t, repo.Create(ctx, newOrder("E25-VIS", domain.OrderTypeVisit)))
	require.NoError(t, repo.Create(ctx, newOrder("E25-CUST", domain.OrderTypeCustomSize)))

	adminOrders, err := repo.List(ctx, domain.OrderFilter{})
	require.NoError(t, err)
	require.Len(t, adminOrders, 4)

	types := make([]domain.OrderType, 0, len(adminOrders))
	for _, o := range adminOrders {
		types = append(types, o.Type)
	}

	expectedTypes := []domain.OrderType{
		domain.OrderTypeCustomSize,
		domain.OrderTypeStandard,
		domain.OrderTypeStandard,
		domain.OrderTypeVisit,
	}

	assert.Equal(t, expectedTypes, types)
}

func TestOrderRepository_CountAndListPaged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewOrderRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	const seeded = 25
	for i := range seeded {
		order := newOrder(fmt.Sprintf("E25-PG%02d", i), domain.OrderTypeStandard)
		order.CreatedAt = time.Now().UTC().Add(time.Duration(i) * time.Second)
		require.NoError(t, repo.Create(ctx, order))
	}

	total, err := repo.Count(ctx, domain.OrderFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(seeded), total)

	// First page is full.
	first, err := repo.ListPaged(ctx, domain.OrderFilter{}, domain.NormalizePageParams(1, 10))
	require.NoError(t, err)
	require.Len(t, first, 10)

	// Pages don't overlap and cover the whole set.
	second, err := repo.ListPaged(ctx, domain.OrderFilter{}, domain.NormalizePageParams(2, 10))
	require.NoError(t, err)
	require.Len(t, second, 10)

	third, err := repo.ListPaged(ctx, domain.OrderFilter{}, domain.NormalizePageParams(3, 10))
	require.NoError(t, err)
	assert.Len(t, third, seeded-20)

	seen := map[string]bool{}

	for _, page := range [][]domain.Order{first, second, third} {
		for _, o := range page {
			assert.False(t, seen[o.Ref], "ref %s appeared on more than one page", o.Ref)
			seen[o.Ref] = true
		}
	}

	assert.Len(t, seen, seeded, "every order appears on exactly one page")
}

func TestOrderRepository_Update_VersionConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewOrderRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	require.NoError(t, repo.Create(ctx, newOrder("E25-CAS", domain.OrderTypeStandard)))

	// Two writers load the same version; the second write must lose.
	first, err := repo.GetByRef(ctx, "E25-CAS")
	require.NoError(t, err)

	second, err := repo.GetByRef(ctx, "E25-CAS")
	require.NoError(t, err)

	first.Status = domain.OrderStatusCancelled
	require.NoError(t, repo.Update(ctx, first))

	second.Status = domain.OrderStatusBooked
	err = repo.Update(ctx, second)
	require.ErrorIs(t, err, domain.ErrConflict)

	loaded, err := repo.GetByRef(ctx, "E25-CAS")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCancelled, loaded.Status, "the losing write must not overwrite the winner")

	// After reloading, the loser can update at the bumped version.
	second, err = repo.GetByRef(ctx, "E25-CAS")
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, second))
}

func TestOrderRepository_Update_LegacyDocWithoutVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewOrderRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	require.NoError(t, repo.Create(ctx, newOrder("E25-LEGACY", domain.OrderTypeStandard)))

	// Simulate a document written before versioning existed.
	_, err := db.Collection("orders").UpdateOne(ctx,
		bson.M{"ref": "E25-LEGACY"},
		bson.M{"$unset": bson.M{"version": ""}},
	)
	require.NoError(t, err)

	legacy, err := repo.GetByRef(ctx, "E25-LEGACY")
	require.NoError(t, err)
	require.Equal(t, int64(0), legacy.Version)

	legacy.Status = domain.OrderStatusCancelled
	require.NoError(t, repo.Update(ctx, legacy))

	loaded, err := repo.GetByRef(ctx, "E25-LEGACY")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCancelled, loaded.Status)
	assert.Equal(t, int64(1), loaded.Version)
}
