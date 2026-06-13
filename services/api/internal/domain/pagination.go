package domain

// Pagination defaults and bounds shared by every paged admin listing.
const (
	// DefaultPage is the page returned when none is requested (1-based).
	DefaultPage = 1
	// DefaultPageSize is the page size used when none is requested.
	DefaultPageSize = 20
	// MaxPageSize caps how many rows a single page may return.
	MaxPageSize = 100
)

// PageParams is a 1-based page request. Use NormalizePageParams to clamp
// caller-supplied values into the supported range before querying a repo.
type PageParams struct {
	Page     int
	PageSize int
}

// NormalizePageParams clamps a page request into the supported range: page is
// at least 1, and pageSize falls back to the default when non-positive and is
// capped at MaxPageSize.
func NormalizePageParams(page, pageSize int) PageParams {
	if page < 1 {
		page = DefaultPage
	}

	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return PageParams{Page: page, PageSize: pageSize}
}

// Skip is the number of documents to skip for this page.
func (p PageParams) Skip() int64 {
	return int64(p.Page-1) * int64(p.PageSize)
}

// Limit is the maximum number of documents this page may contain.
func (p PageParams) Limit() int64 {
	return int64(p.PageSize)
}

// Page is one page of a larger result set, carrying enough metadata for a
// client to render pagination controls.
type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}

// NewPage builds a Page from its items, the total match count, and the page
// request that produced it.
func NewPage[T any](items []T, total int64, params PageParams) Page[T] {
	return Page[T]{
		Items:    items,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}
}
