package service_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

// catalogPageSlice returns the window of items for a normalized page request.
func catalogPageSlice[T any](items []T, params domain.PageParams) []T {
	skip := int(params.Skip())
	if skip >= len(items) {
		return []T{}
	}

	end := min(skip+params.PageSize, len(items))

	return items[skip:end]
}

type memCatalogCollections struct {
	byID   map[string]*domain.Collection
	nextID int
}

func newMemCatalogCollections() *memCatalogCollections {
	return &memCatalogCollections{byID: map[string]*domain.Collection{}, nextID: 1}
}

func (m *memCatalogCollections) Create(_ context.Context, c *domain.Collection) error {
	for _, existing := range m.byID {
		if existing.Slug == c.Slug {
			return domain.ErrDuplicateSlug
		}
	}

	c.ID = "col-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *c
	m.byID[c.ID] = &clone

	return nil
}

func (m *memCatalogCollections) Update(_ context.Context, id, name, note string) error {
	collection, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}

	collection.Name, collection.Note = name, note

	return nil
}

func (m *memCatalogCollections) GetByID(_ context.Context, id string) (*domain.Collection, error) {
	collection, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}

	clone := *collection

	return &clone, nil
}

func (m *memCatalogCollections) GetBySlug(_ context.Context, slug string) (*domain.Collection, error) {
	for _, collection := range m.byID {
		if collection.Slug == slug {
			clone := *collection

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (m *memCatalogCollections) List(_ context.Context, includeRetired bool) ([]domain.Collection, error) {
	out := make([]domain.Collection, 0, len(m.byID))

	for _, collection := range m.byID {
		if !includeRetired && collection.Status != domain.StatusLive {
			continue
		}

		out = append(out, *collection)
	}

	return out, nil
}

func (m *memCatalogCollections) Count(ctx context.Context, includeRetired bool) (int64, error) {
	out, err := m.List(ctx, includeRetired)

	return int64(len(out)), err
}

func (m *memCatalogCollections) ListPaged(
	ctx context.Context, includeRetired bool, params domain.PageParams,
) ([]domain.Collection, error) {
	out, err := m.List(ctx, includeRetired)
	if err != nil {
		return nil, err
	}

	return catalogPageSlice(out, params), nil
}

func (m *memCatalogCollections) SetStatus(_ context.Context, id string, status domain.Status, at time.Time) error {
	collection, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}

	collection.Status = status

	collection.RetiredAt = nil
	if status == domain.StatusRetired {
		collection.RetiredAt = &at
	}

	return nil
}

func (m *memCatalogCollections) Delete(_ context.Context, id string) error {
	delete(m.byID, id)

	return nil
}

type memCatalogDesigns struct {
	byID   map[string]*domain.Design
	nextID int
}

func newMemCatalogDesigns() *memCatalogDesigns {
	return &memCatalogDesigns{byID: map[string]*domain.Design{}, nextID: 1}
}

func (m *memCatalogDesigns) Create(_ context.Context, d *domain.Design) error {
	for _, existing := range m.byID {
		if existing.Slug == d.Slug {
			return domain.ErrDuplicateSlug
		}
	}

	d.ID = "des-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *d
	m.byID[d.ID] = &clone

	return nil
}

func (m *memCatalogDesigns) Update(_ context.Context, d *domain.Design) error {
	if _, ok := m.byID[d.ID]; !ok {
		return domain.ErrNotFound
	}

	clone := *d
	m.byID[d.ID] = &clone

	return nil
}

func (m *memCatalogDesigns) GetByID(_ context.Context, id string) (*domain.Design, error) {
	design, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}

	clone := *design

	return &clone, nil
}

func (m *memCatalogDesigns) GetBySlug(_ context.Context, slug string) (*domain.Design, error) {
	for _, design := range m.byID {
		if design.Slug == slug {
			clone := *design

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (m *memCatalogDesigns) List(_ context.Context, filter domain.DesignFilter) ([]domain.Design, error) {
	out := make([]domain.Design, 0, len(m.byID))

	for _, design := range m.byID {
		if !filter.IncludeRetired && design.Status != domain.StatusLive {
			continue
		}

		if filter.CollectionID != "" && design.CollectionID != filter.CollectionID {
			continue
		}

		out = append(out, *design)
	}

	return out, nil
}

func (m *memCatalogDesigns) Count(ctx context.Context, filter domain.DesignFilter) (int64, error) {
	out, err := m.List(ctx, filter)

	return int64(len(out)), err
}

func (m *memCatalogDesigns) ListPaged(
	ctx context.Context, filter domain.DesignFilter, params domain.PageParams,
) ([]domain.Design, error) {
	out, err := m.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return catalogPageSlice(out, params), nil
}

func (m *memCatalogDesigns) SetStatusBulk(_ context.Context, ids []string, status domain.Status, at time.Time) error {
	for _, id := range ids {
		design, ok := m.byID[id]
		if !ok {
			return domain.ErrNotFound
		}

		design.Status = status

		design.RetiredAt = nil
		if status == domain.StatusRetired {
			design.RetiredAt = &at
		}
	}

	return nil
}

func (m *memCatalogDesigns) SetStatusByCollection(
	_ context.Context, collectionID string, status domain.Status, at time.Time,
) error {
	for _, design := range m.byID {
		if design.CollectionID != collectionID {
			continue
		}

		design.Status = status

		design.RetiredAt = nil
		if status == domain.StatusRetired {
			design.RetiredAt = &at
		}
	}

	return nil
}

func (m *memCatalogDesigns) Delete(_ context.Context, id string) error {
	delete(m.byID, id)

	return nil
}

func (m *memCatalogDesigns) DeleteByCollection(_ context.Context, collectionID string) error {
	for id, design := range m.byID {
		if design.CollectionID == collectionID {
			delete(m.byID, id)
		}
	}

	return nil
}

// --- tests --------------------------------------------------------------------

func newCatalog() (*service.Catalog, *memCatalogCollections, *memCatalogDesigns) {
	collections := newMemCatalogCollections()
	designs := newMemCatalogDesigns()

	return service.NewCatalog(collections, designs), collections, designs
}

func band(label string, price int64) domain.SizeBand {
	return domain.SizeBand{Label: label, PricePesewas: price, Chart: map[string]string{"bust": "86 cm"}}
}

func validDesignInput(collectionID string) service.DesignInput {
	return service.DesignInput{
		CollectionID: collectionID,
		Name:         "The Boardroom Blazer",
		Note:         "",
		Photos:       nil,
		SizeBands:    []domain.SizeBand{band("8", 50000), band("10", 52000)},
	}
}

func TestCreateCollection_Slugifies(t *testing.T) {
	t.Parallel()

	catalog, _, _ := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "  The Boardroom Edit!  ", "First drop")
	require.NoError(t, err)
	assert.Equal(t, "the-boardroom-edit", collection.Slug)
	assert.Equal(t, domain.StatusLive, collection.Status)
	assert.NotEmpty(t, collection.ID)
}

func TestCreateCollection_SlugConflictGetsSuffix(t *testing.T) {
	t.Parallel()

	catalog, _, _ := newCatalog()

	first, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	second, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	assert.Equal(t, "velvet", first.Slug)
	assert.Equal(t, "velvet-2", second.Slug)
}

func TestCreateDesign_Validation(t *testing.T) {
	t.Parallel()

	catalog, _, _ := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	cases := []struct {
		name  string
		mutil func(*service.DesignInput)
	}{
		{"empty name", func(in *service.DesignInput) { in.Name = "  " }},
		{"no bands", func(in *service.DesignInput) { in.SizeBands = nil }},
		{"zero price", func(in *service.DesignInput) { in.SizeBands = []domain.SizeBand{band("8", 0)} }},
		{"duplicate band", func(in *service.DesignInput) {
			in.SizeBands = []domain.SizeBand{band("8", 1000), band("8", 2000)}
		}},
		{"unknown collection", func(in *service.DesignInput) { in.CollectionID = "missing" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := validDesignInput(collection.ID)
			tc.mutil(&input)

			_, err := catalog.CreateDesign(t.Context(), input)
			require.ErrorIs(t, err, domain.ErrInvalidInput)
		})
	}
}

func TestRetireCollection_CascadesAndBlocksDesignRestore(t *testing.T) {
	t.Parallel()

	catalog, _, designs := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	design, err := catalog.CreateDesign(t.Context(), validDesignInput(collection.ID))
	require.NoError(t, err)

	require.NoError(t, catalog.RetireCollection(t.Context(), collection.ID))

	retired, err := designs.GetByID(t.Context(), design.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRetired, retired.Status, "retiring a collection retires its designs")

	err = catalog.RestoreDesigns(t.Context(), []string{design.ID})
	require.ErrorIs(t, err, domain.ErrInvalidInput, "design restore is blocked while its collection is retired")

	require.NoError(t, catalog.RestoreCollection(t.Context(), collection.ID))

	restored, err := designs.GetByID(t.Context(), design.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusLive, restored.Status, "restoring a collection restores its designs")
}

func TestDeleteCollection_DeletesDesigns(t *testing.T) {
	t.Parallel()

	catalog, collections, designs := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	_, err = catalog.CreateDesign(t.Context(), validDesignInput(collection.ID))
	require.NoError(t, err)

	require.NoError(t, catalog.DeleteCollection(t.Context(), collection.ID))
	assert.Empty(t, collections.byID)
	assert.Empty(t, designs.byID)
}

func TestGetLiveDesignBySlug_HidesRetired(t *testing.T) {
	t.Parallel()

	catalog, _, _ := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	design, err := catalog.CreateDesign(t.Context(), validDesignInput(collection.ID))
	require.NoError(t, err)

	found, err := catalog.GetLiveDesignBySlug(t.Context(), design.Slug)
	require.NoError(t, err)
	assert.Equal(t, design.ID, found.ID)

	require.NoError(t, catalog.RetireDesigns(t.Context(), []string{design.ID}))

	_, err = catalog.GetLiveDesignBySlug(t.Context(), design.Slug)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateDesign_KeepsSlugAndStatus(t *testing.T) {
	t.Parallel()

	catalog, _, _ := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "Velvet", "")
	require.NoError(t, err)

	design, err := catalog.CreateDesign(t.Context(), validDesignInput(collection.ID))
	require.NoError(t, err)

	input := validDesignInput(collection.ID)
	input.Name = "Completely New Name"

	updated, err := catalog.UpdateDesign(t.Context(), design.ID, input)
	require.NoError(t, err)
	assert.Equal(t, "Completely New Name", updated.Name)
	assert.Equal(t, design.Slug, updated.Slug, "slug is immutable — shared links stay stable")
	assert.Equal(t, domain.StatusLive, updated.Status)
}

func TestSlugify_TransliteratesAccentsAndStripsSymbols(t *testing.T) {
	t.Parallel()

	catalog, _, _ := newCatalog()

	collection, err := catalog.CreateCollection(t.Context(), "Été 2026 — Linen & Silk", "")
	require.NoError(t, err)
	assert.Equal(t, "ete-2026-linen-silk", collection.Slug)
	assert.NotContains(t, collection.Slug, " ")
	assert.Empty(t, strings.Trim(collection.Slug, "abcdefghijklmnopqrstuvwxyz0123456789-"),
		"slug %q must be lowercase alnum and dashes", collection.Slug)
}
