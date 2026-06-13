package domain

import (
	"context"
	"errors"
	"time"
)

// ErrDuplicateSlug is returned when a slug already exists.
var ErrDuplicateSlug = errors.New("slug already exists")

// Status is the lifecycle state of catalog items. Retired items leave the
// shop but stay in the dashboard and can be brought back (scope §05).
type Status string

// Catalog lifecycle states.
const (
	StatusLive    Status = "live"
	StatusRetired Status = "retired"
)

// Collection is a themed release of around ten designs (scope §1.2).
type Collection struct {
	ID        string
	Name      string
	Slug      string // immutable after create — shareable links stay stable
	Note      string
	Status    Status
	CreatedAt time.Time
	RetiredAt *time.Time
}

// Photo is a Cloudinary asset attached to a design.
type Photo struct {
	PublicID string
	Order    int
}

// SizeBand is one standard size with its own chart and set price (scope §4.2).
type SizeBand struct {
	Label        string
	PricePesewas int64
	Chart        map[string]string // e.g. {"bust": "86 cm", "waist": "66 cm"}
}

// Design is a single garment design inside a collection.
type Design struct {
	ID           string
	CollectionID string
	Name         string
	Slug         string // immutable after create
	Note         string
	Photos       []Photo
	SizeBands    []SizeBand
	Status       Status
	CreatedAt    time.Time
	RetiredAt    *time.Time
}

// CollectionRepository is the persistence port for collections.
type CollectionRepository interface {
	Create(ctx context.Context, c *Collection) error
	Update(ctx context.Context, id, name, note string) error
	GetByID(ctx context.Context, id string) (*Collection, error)
	GetBySlug(ctx context.Context, slug string) (*Collection, error)
	List(ctx context.Context, includeRetired bool) ([]Collection, error)
	// Count returns the number of collections matching includeRetired.
	Count(ctx context.Context, includeRetired bool) (int64, error)
	// ListPaged returns one page of collections, newest first.
	ListPaged(ctx context.Context, includeRetired bool, params PageParams) ([]Collection, error)
	SetStatus(ctx context.Context, id string, status Status, at time.Time) error
	Delete(ctx context.Context, id string) error
}

// DesignFilter narrows design listings.
type DesignFilter struct {
	CollectionID   string
	Query          string
	IncludeRetired bool
}

// DesignRepository is the persistence port for designs.
type DesignRepository interface {
	Create(ctx context.Context, d *Design) error
	Update(ctx context.Context, d *Design) error
	GetByID(ctx context.Context, id string) (*Design, error)
	GetBySlug(ctx context.Context, slug string) (*Design, error)
	List(ctx context.Context, filter DesignFilter) ([]Design, error)
	// Count returns the number of designs matching the filter.
	Count(ctx context.Context, filter DesignFilter) (int64, error)
	// ListPaged returns one page of designs matching the filter, newest first.
	ListPaged(ctx context.Context, filter DesignFilter, params PageParams) ([]Design, error)
	SetStatusBulk(ctx context.Context, ids []string, status Status, at time.Time) error
	SetStatusByCollection(ctx context.Context, collectionID string, status Status, at time.Time) error
	Delete(ctx context.Context, id string) error
	DeleteByCollection(ctx context.Context, collectionID string) error
}
