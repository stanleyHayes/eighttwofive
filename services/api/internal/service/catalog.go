package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	maxCatalogName = 120
	maxNote        = 2000
	maxPhotos      = 12
	maxSlugRetries = 5
)

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

// Catalog implements collection and design management (scope §05, §06).
type Catalog struct {
	collections domain.CollectionRepository
	designs     domain.DesignRepository
	now         func() time.Time
}

// NewCatalog wires the catalog service.
func NewCatalog(collections domain.CollectionRepository, designs domain.DesignRepository) *Catalog {
	return &Catalog{collections: collections, designs: designs, now: time.Now}
}

// --- collections ------------------------------------------------------------

// CreateCollection creates a live collection with a unique slug.
func (c *Catalog) CreateCollection(ctx context.Context, name, note string) (*domain.Collection, error) {
	name = strings.TrimSpace(name)

	err := validateCatalogText(name, note)
	if err != nil {
		return nil, err
	}

	collection := &domain.Collection{
		ID:        "",
		Name:      name,
		Slug:      "",
		Note:      strings.TrimSpace(note),
		Status:    domain.StatusLive,
		CreatedAt: c.now().UTC(),
		RetiredAt: nil,
	}

	err = withUniqueSlug(name, func(slug string) error {
		collection.Slug = slug

		return c.collections.Create(ctx, collection)
	})
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}

	return collection, nil
}

// UpdateCollection renames or re-notes a collection. The slug never changes.
func (c *Catalog) UpdateCollection(ctx context.Context, id, name, note string) (*domain.Collection, error) {
	name = strings.TrimSpace(name)

	err := validateCatalogText(name, note)
	if err != nil {
		return nil, err
	}

	err = c.collections.Update(ctx, id, name, strings.TrimSpace(note))
	if err != nil {
		return nil, fmt.Errorf("update collection: %w", err)
	}

	collection, err := c.collections.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load updated collection: %w", err)
	}

	return collection, nil
}

// ListCollections lists collections, optionally including retired ones.
func (c *Catalog) ListCollections(ctx context.Context, includeRetired bool) ([]domain.Collection, error) {
	collections, err := c.collections.List(ctx, includeRetired)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}

	return collections, nil
}

// ListCollectionsPaged returns one page of collections (newest first) with the
// total count, for the admin collections table.
func (c *Catalog) ListCollectionsPaged(
	ctx context.Context, includeRetired bool, page, pageSize int,
) (domain.Page[domain.Collection], error) {
	params := domain.NormalizePageParams(page, pageSize)

	total, err := c.collections.Count(ctx, includeRetired)
	if err != nil {
		return domain.Page[domain.Collection]{}, fmt.Errorf("count collections: %w", err)
	}

	collections, err := c.collections.ListPaged(ctx, includeRetired, params)
	if err != nil {
		return domain.Page[domain.Collection]{}, fmt.Errorf("list collections: %w", err)
	}

	return domain.NewPage(collections, total, params), nil
}

// GetCollectionBySlug returns a live collection and its live designs.
func (c *Catalog) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, []domain.Design, error) {
	collection, err := c.collections.GetBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("get collection: %w", err)
	}

	if collection.Status != domain.StatusLive {
		return nil, nil, domain.ErrNotFound
	}

	designs, err := c.designs.List(ctx, domain.DesignFilter{
		CollectionID:   collection.ID,
		Query:          "",
		IncludeRetired: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list collection designs: %w", err)
	}

	return collection, designs, nil
}

// RetireCollection retires a collection and every design in it.
func (c *Catalog) RetireCollection(ctx context.Context, id string) error {
	return c.setCollectionStatus(ctx, id, domain.StatusRetired)
}

// RestoreCollection brings a collection and its designs back to the shop.
func (c *Catalog) RestoreCollection(ctx context.Context, id string) error {
	return c.setCollectionStatus(ctx, id, domain.StatusLive)
}

// DeleteCollection permanently deletes a collection AND its designs — the
// separate, deliberate action the scope distinguishes from retiring (§05).
func (c *Catalog) DeleteCollection(ctx context.Context, id string) error {
	err := c.designs.DeleteByCollection(ctx, id)
	if err != nil {
		return fmt.Errorf("delete collection designs: %w", err)
	}

	err = c.collections.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}

	return nil
}

// --- designs -----------------------------------------------------------------

// DesignInput carries everything needed to create or update a design.
type DesignInput struct {
	CollectionID string
	Name         string
	Note         string
	Photos       []domain.Photo
	SizeBands    []domain.SizeBand
}

// CreateDesign creates a live design with a unique slug.
func (c *Catalog) CreateDesign(ctx context.Context, input DesignInput) (*domain.Design, error) {
	err := c.validateDesignInput(ctx, &input)
	if err != nil {
		return nil, err
	}

	design := &domain.Design{
		ID:           "",
		CollectionID: input.CollectionID,
		Name:         input.Name,
		Slug:         "",
		Note:         input.Note,
		Photos:       input.Photos,
		SizeBands:    input.SizeBands,
		Status:       domain.StatusLive,
		CreatedAt:    c.now().UTC(),
		RetiredAt:    nil,
	}

	err = withUniqueSlug(input.Name, func(slug string) error {
		design.Slug = slug

		return c.designs.Create(ctx, design)
	})
	if err != nil {
		return nil, fmt.Errorf("create design: %w", err)
	}

	return design, nil
}

// UpdateDesign replaces a design's editable fields. Slug and status stay.
func (c *Catalog) UpdateDesign(ctx context.Context, id string, input DesignInput) (*domain.Design, error) {
	err := c.validateDesignInput(ctx, &input)
	if err != nil {
		return nil, err
	}

	design, err := c.designs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load design: %w", err)
	}

	design.CollectionID = input.CollectionID
	design.Name = input.Name
	design.Note = input.Note
	design.Photos = input.Photos
	design.SizeBands = input.SizeBands

	err = c.designs.Update(ctx, design)
	if err != nil {
		return nil, fmt.Errorf("update design: %w", err)
	}

	return design, nil
}

// ListDesigns lists designs for the storefront or the dashboard.
func (c *Catalog) ListDesigns(ctx context.Context, filter domain.DesignFilter) ([]domain.Design, error) {
	designs, err := c.designs.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list designs: %w", err)
	}

	return designs, nil
}

// ListDesignsPaged returns one page of designs (newest first) matching the
// filter, with the total count, for the admin designs table.
func (c *Catalog) ListDesignsPaged(
	ctx context.Context, filter domain.DesignFilter, page, pageSize int,
) (domain.Page[domain.Design], error) {
	params := domain.NormalizePageParams(page, pageSize)

	total, err := c.designs.Count(ctx, filter)
	if err != nil {
		return domain.Page[domain.Design]{}, fmt.Errorf("count designs: %w", err)
	}

	designs, err := c.designs.ListPaged(ctx, filter, params)
	if err != nil {
		return domain.Page[domain.Design]{}, fmt.Errorf("list designs: %w", err)
	}

	return domain.NewPage(designs, total, params), nil
}

// GetLiveDesignBySlug returns a design for the storefront; retired is 404.
func (c *Catalog) GetLiveDesignBySlug(ctx context.Context, slug string) (*domain.Design, error) {
	design, err := c.designs.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get design: %w", err)
	}

	if design.Status != domain.StatusLive {
		return nil, domain.ErrNotFound
	}

	return design, nil
}

// GetDesignByID returns a design regardless of status (dashboard use).
func (c *Catalog) GetDesignByID(ctx context.Context, id string) (*domain.Design, error) {
	design, err := c.designs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get design: %w", err)
	}

	return design, nil
}

// RetireDesigns retires one or several designs at once (scope §05).
func (c *Catalog) RetireDesigns(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("%w: no design ids given", domain.ErrInvalidInput)
	}

	err := c.designs.SetStatusBulk(ctx, ids, domain.StatusRetired, c.now().UTC())
	if err != nil {
		return fmt.Errorf("retire designs: %w", err)
	}

	return nil
}

// RestoreDesigns brings designs back, provided their collection is live.
func (c *Catalog) RestoreDesigns(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("%w: no design ids given", domain.ErrInvalidInput)
	}

	for _, id := range ids {
		design, err := c.designs.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("load design: %w", err)
		}

		collection, err := c.collections.GetByID(ctx, design.CollectionID)
		if err != nil {
			return fmt.Errorf("load collection: %w", err)
		}

		if collection.Status != domain.StatusLive {
			return fmt.Errorf("%w: restore the collection first", domain.ErrInvalidInput)
		}
	}

	err := c.designs.SetStatusBulk(ctx, ids, domain.StatusLive, c.now().UTC())
	if err != nil {
		return fmt.Errorf("restore designs: %w", err)
	}

	return nil
}

// DeleteDesign permanently deletes a single design.
func (c *Catalog) DeleteDesign(ctx context.Context, id string) error {
	err := c.designs.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete design: %w", err)
	}

	return nil
}

// --- internals -----------------------------------------------------------------

func (c *Catalog) setCollectionStatus(ctx context.Context, id string, status domain.Status) error {
	at := c.now().UTC()

	err := c.collections.SetStatus(ctx, id, status, at)
	if err != nil {
		return fmt.Errorf("set collection status: %w", err)
	}

	err = c.designs.SetStatusByCollection(ctx, id, status, at)
	if err != nil {
		return fmt.Errorf("set designs status: %w", err)
	}

	return nil
}

// --- validation & slugs --------------------------------------------------------

func (c *Catalog) validateDesignInput(ctx context.Context, input *DesignInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Note = strings.TrimSpace(input.Note)

	err := validateCatalogText(input.Name, input.Note)
	if err != nil {
		return err
	}

	if len(input.Photos) > maxPhotos {
		return fmt.Errorf("%w: at most %d photos", domain.ErrInvalidInput, maxPhotos)
	}

	if len(input.SizeBands) == 0 {
		return fmt.Errorf("%w: at least one size band is required", domain.ErrInvalidInput)
	}

	seen := map[string]bool{}

	for _, band := range input.SizeBands {
		if strings.TrimSpace(band.Label) == "" {
			return fmt.Errorf("%w: size band label is required", domain.ErrInvalidInput)
		}

		if band.PricePesewas <= 0 {
			return fmt.Errorf("%w: size band %q needs a positive price", domain.ErrInvalidInput, band.Label)
		}

		if seen[band.Label] {
			return fmt.Errorf("%w: duplicate size band %q", domain.ErrInvalidInput, band.Label)
		}

		seen[band.Label] = true
	}

	_, err = c.collections.GetByID(ctx, input.CollectionID)
	if err != nil {
		return fmt.Errorf("%w: unknown collection", domain.ErrInvalidInput)
	}

	return nil
}

func validateCatalogText(name, note string) error {
	if name == "" || len(name) > maxCatalogName {
		return fmt.Errorf("%w: name must be 1-%d characters", domain.ErrInvalidInput, maxCatalogName)
	}

	if len(note) > maxNote {
		return fmt.Errorf("%w: note must be at most %d characters", domain.ErrInvalidInput, maxNote)
	}

	return nil
}

// withUniqueSlug tries slugified candidates (name, name-2, …) until create
// succeeds or attempts run out.
func withUniqueSlug(name string, create func(slug string) error) error {
	base := slugify(name)

	for attempt := 1; attempt <= maxSlugRetries; attempt++ {
		slug := base
		if attempt > 1 {
			slug = base + "-" + strconv.Itoa(attempt)
		}

		err := create(slug)
		if err == nil {
			return nil
		}

		if !errors.Is(err, domain.ErrDuplicateSlug) {
			return err
		}
	}

	return fmt.Errorf("%w: could not find a free slug for %q", domain.ErrDuplicateSlug, name)
}

func slugify(name string) string {
	// Decompose accented characters (NFD) so combining marks can be stripped,
	// turning e.g. "é" into "e" instead of removing the letter entirely.
	decomposed := norm.NFD.String(strings.ToLower(name))
	cleaned := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}

		return r
	}, decomposed)

	slug := strings.Trim(slugStrip.ReplaceAllString(cleaned, "-"), "-")
	if slug == "" {
		slug = "item"
	}

	return slug
}
