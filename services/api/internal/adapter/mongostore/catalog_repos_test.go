package mongostore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func newCollection(name, slug string) *domain.Collection {
	return &domain.Collection{
		ID:        "",
		Name:      name,
		Slug:      slug,
		Note:      "",
		Status:    domain.StatusLive,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		RetiredAt: nil,
	}
}

func newDesign(collectionID, name, slug string) *domain.Design {
	return &domain.Design{
		ID:           "",
		CollectionID: collectionID,
		Name:         name,
		Slug:         slug,
		Note:         "tailored office wear",
		Photos:       []domain.Photo{{PublicID: "e25/test", Order: 0}},
		SizeBands: []domain.SizeBand{
			{Label: "8", PricePesewas: 50000, Chart: map[string]string{"bust": "86 cm"}},
		},
		Status:    domain.StatusLive,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		RetiredAt: nil,
	}
}

func TestCatalogRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	collections := mongostore.NewCollectionRepository(db)
	designs := mongostore.NewDesignRepository(db)
	ctx := context.Background()
	require.NoError(t, collections.EnsureIndexes(ctx))
	require.NoError(t, designs.EnsureIndexes(ctx))

	// Collection create + slug uniqueness.
	velvet := newCollection("Velvet", "velvet")
	require.NoError(t, collections.Create(ctx, velvet))
	require.NotEmpty(t, velvet.ID)

	dup := newCollection("Velvet Again", "velvet")
	require.ErrorIs(t, collections.Create(ctx, dup), domain.ErrDuplicateSlug)

	// Design create + slug uniqueness + round-trip of bands/photos/chart.
	blazer := newDesign(velvet.ID, "Boardroom Blazer", "boardroom-blazer")
	require.NoError(t, designs.Create(ctx, blazer))
	require.NotEmpty(t, blazer.ID)

	dupDesign := newDesign(velvet.ID, "Other", "boardroom-blazer")
	require.ErrorIs(t, designs.Create(ctx, dupDesign), domain.ErrDuplicateSlug)

	loaded, err := designs.GetBySlug(ctx, "boardroom-blazer")
	require.NoError(t, err)
	assert.Equal(t, blazer.ID, loaded.ID)
	require.Len(t, loaded.SizeBands, 1)
	assert.Equal(t, int64(50000), loaded.SizeBands[0].PricePesewas)
	assert.Equal(t, "86 cm", loaded.SizeBands[0].Chart["bust"])
	require.Len(t, loaded.Photos, 1)
	assert.Equal(t, "e25/test", loaded.Photos[0].PublicID)

	// Text search finds by name; live filter hides retired.
	trousers := newDesign(velvet.ID, "Tailored Trousers", "tailored-trousers")
	require.NoError(t, designs.Create(ctx, trousers))

	found, err := designs.List(ctx, domain.DesignFilter{CollectionID: "", Query: "trousers", IncludeRetired: false})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "Tailored Trousers", found[0].Name)

	// Retire by collection cascades; public list goes empty.
	require.NoError(t, designs.SetStatusByCollection(ctx, velvet.ID, domain.StatusRetired, time.Now().UTC()))

	live, err := designs.List(ctx, domain.DesignFilter{CollectionID: velvet.ID, Query: "", IncludeRetired: false})
	require.NoError(t, err)
	assert.Empty(t, live)

	all, err := designs.List(ctx, domain.DesignFilter{CollectionID: velvet.ID, Query: "", IncludeRetired: true})
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.NotNil(t, all[0].RetiredAt)

	// Bulk restore clears retiredAt.
	require.NoError(t, designs.SetStatusBulk(ctx, []string{blazer.ID, trousers.ID}, domain.StatusLive, time.Now().UTC()))

	restored, err := designs.GetByID(ctx, blazer.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusLive, restored.Status)
	assert.Nil(t, restored.RetiredAt)

	// Collection status round-trip.
	require.NoError(t, collections.SetStatus(ctx, velvet.ID, domain.StatusRetired, time.Now().UTC()))

	liveCollections, err := collections.List(ctx, false)
	require.NoError(t, err)
	assert.Empty(t, liveCollections)

	allCollections, err := collections.List(ctx, true)
	require.NoError(t, err)
	require.Len(t, allCollections, 1)
	assert.NotNil(t, allCollections[0].RetiredAt)

	// Update text fields.
	require.NoError(t, collections.Update(ctx, velvet.ID, "Velvet II", "second run"))

	renamed, err := collections.GetByID(ctx, velvet.ID)
	require.NoError(t, err)
	assert.Equal(t, "Velvet II", renamed.Name)
	assert.Equal(t, "velvet", renamed.Slug, "slug never changes")

	// Permanent deletes.
	require.NoError(t, designs.DeleteByCollection(ctx, velvet.ID))

	gone, err := designs.List(ctx, domain.DesignFilter{CollectionID: velvet.ID, Query: "", IncludeRetired: true})
	require.NoError(t, err)
	assert.Empty(t, gone)

	require.NoError(t, collections.Delete(ctx, velvet.ID))

	_, err = collections.GetByID(ctx, velvet.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCatalogRepositories_CountAndListPaged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	collections := mongostore.NewCollectionRepository(db)
	designs := mongostore.NewDesignRepository(db)
	ctx := context.Background()
	require.NoError(t, collections.EnsureIndexes(ctx))
	require.NoError(t, designs.EnsureIndexes(ctx))

	col := newCollection("Paged", "paged")
	require.NoError(t, collections.Create(ctx, col))

	const seeded = 24

	base := time.Now().UTC().Truncate(time.Millisecond)

	for i := range seeded {
		design := newDesign(col.ID, fmt.Sprintf("Design %02d", i), fmt.Sprintf("design-%02d", i))
		design.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		require.NoError(t, designs.Create(ctx, design))
	}

	filter := domain.DesignFilter{CollectionID: col.ID, IncludeRetired: true}

	total, err := designs.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(seeded), total)

	// First page: full page, newest first.
	first, err := designs.ListPaged(ctx, filter, domain.NormalizePageParams(1, 10))
	require.NoError(t, err)
	require.Len(t, first, 10)
	assert.Equal(t, "Design 23", first[0].Name)

	// Last (partial) page.
	last, err := designs.ListPaged(ctx, filter, domain.NormalizePageParams(3, 10))
	require.NoError(t, err)
	assert.Len(t, last, seeded-20)

	// Past the end returns empty, not an error.
	beyond, err := designs.ListPaged(ctx, filter, domain.NormalizePageParams(99, 10))
	require.NoError(t, err)
	assert.Empty(t, beyond)

	// Collection paging + count.
	colTotal, err := collections.Count(ctx, true)
	require.NoError(t, err)
	assert.Equal(t, int64(1), colTotal)

	colPage, err := collections.ListPaged(ctx, true, domain.NormalizePageParams(1, 20))
	require.NoError(t, err)
	require.Len(t, colPage, 1)
	assert.Equal(t, "Paged", colPage[0].Name)
}
