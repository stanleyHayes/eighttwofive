package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

// --- DTOs --------------------------------------------------------------------

type collectionDTO struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Note      string     `json:"note"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	RetiredAt *time.Time `json:"retiredAt,omitempty"`
}

func toCollectionDTO(c domain.Collection) collectionDTO {
	return collectionDTO{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		Note:      c.Note,
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt,
		RetiredAt: c.RetiredAt,
	}
}

type photoDTO struct {
	PublicID string `json:"publicId"`
	Order    int    `json:"order"`
}

type sizeBandDTO struct {
	Label        string            `json:"label"`
	PricePesewas int64             `json:"pricePesewas"`
	Chart        map[string]string `json:"chart"`
}

type designDTO struct {
	ID           string        `json:"id"`
	CollectionID string        `json:"collectionId"`
	Name         string        `json:"name"`
	Slug         string        `json:"slug"`
	Note         string        `json:"note"`
	Photos       []photoDTO    `json:"photos"`
	SizeBands    []sizeBandDTO `json:"sizeBands"`
	Status       string        `json:"status"`
	CreatedAt    time.Time     `json:"createdAt"`
	RetiredAt    *time.Time    `json:"retiredAt,omitempty"`
}

func toDesignDTO(d domain.Design) designDTO {
	photos := make([]photoDTO, 0, len(d.Photos))
	for _, p := range d.Photos {
		photos = append(photos, photoDTO{PublicID: p.PublicID, Order: p.Order})
	}

	bands := make([]sizeBandDTO, 0, len(d.SizeBands))
	for _, b := range d.SizeBands {
		bands = append(bands, sizeBandDTO{Label: b.Label, PricePesewas: b.PricePesewas, Chart: b.Chart})
	}

	return designDTO{
		ID:           d.ID,
		CollectionID: d.CollectionID,
		Name:         d.Name,
		Slug:         d.Slug,
		Note:         d.Note,
		Photos:       photos,
		SizeBands:    bands,
		Status:       string(d.Status),
		CreatedAt:    d.CreatedAt,
		RetiredAt:    d.RetiredAt,
	}
}

func toDesignDTOs(designs []domain.Design) []designDTO {
	dtos := make([]designDTO, 0, len(designs))
	for _, d := range designs {
		dtos = append(dtos, toDesignDTO(d))
	}

	return dtos
}

type designRequest struct {
	CollectionID string        `json:"collectionId"`
	Name         string        `json:"name"`
	Note         string        `json:"note"`
	Photos       []photoDTO    `json:"photos"`
	SizeBands    []sizeBandDTO `json:"sizeBands"`
}

func (req designRequest) toInput() service.DesignInput {
	photos := make([]domain.Photo, 0, len(req.Photos))
	for _, p := range req.Photos {
		photos = append(photos, domain.Photo{PublicID: p.PublicID, Order: p.Order})
	}

	bands := make([]domain.SizeBand, 0, len(req.SizeBands))
	for _, b := range req.SizeBands {
		bands = append(bands, domain.SizeBand{Label: b.Label, PricePesewas: b.PricePesewas, Chart: b.Chart})
	}

	return service.DesignInput{
		CollectionID: req.CollectionID,
		Name:         req.Name,
		Note:         req.Note,
		Photos:       photos,
		SizeBands:    bands,
	}
}

// --- shared helpers ---------------------------------------------------------

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	err := json.NewDecoder(r.Body).Decode(dst)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "request body must be valid JSON")

		return false
	}

	return true
}

func respondCatalogError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, domain.ErrDuplicateSlug):
		respondError(w, http.StatusConflict, "conflict", "an item with a very similar name already exists")
	default:
		respondInternal(w, r, err)
	}
}

// --- public storefront endpoints ----------------------------------------------

// ListCollections handles GET /api/v1/collections (live only).
func (h *Handlers) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.catalog.ListCollections(r.Context(), false)
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	dtos := make([]collectionDTO, 0, len(collections))
	for _, c := range collections {
		dtos = append(dtos, toCollectionDTO(c))
	}

	respondJSON(w, http.StatusOK, dtos)
}

type collectionWithDesignsDTO struct {
	Collection collectionDTO `json:"collection"`
	Designs    []designDTO   `json:"designs"`
}

// GetCollection handles GET /api/v1/collections/{slug} (live only).
func (h *Handlers) GetCollection(w http.ResponseWriter, r *http.Request) {
	collection, designs, err := h.catalog.GetCollectionBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, collectionWithDesignsDTO{
		Collection: toCollectionDTO(*collection),
		Designs:    toDesignDTOs(designs),
	})
}

// ListDesigns handles GET /api/v1/designs?collection={id}&q={term} (live only).
func (h *Handlers) ListDesigns(w http.ResponseWriter, r *http.Request) {
	designs, err := h.catalog.ListDesigns(r.Context(), domain.DesignFilter{
		CollectionID:   r.URL.Query().Get("collection"),
		Query:          r.URL.Query().Get("q"),
		IncludeRetired: false,
	})
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toDesignDTOs(designs))
}

// GetDesign handles GET /api/v1/designs/{slug} (live only; retired is 404).
func (h *Handlers) GetDesign(w http.ResponseWriter, r *http.Request) {
	design, err := h.catalog.GetLiveDesignBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toDesignDTO(*design))
}

// --- admin endpoints -------------------------------------------------------------

type collectionRequest struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

func toCollectionDTOs(collections []domain.Collection) []collectionDTO {
	dtos := make([]collectionDTO, 0, len(collections))
	for _, c := range collections {
		dtos = append(dtos, toCollectionDTO(c))
	}

	return dtos
}

// AdminListCollections handles GET /api/v1/admin/collections (retired included),
// paginated (page, pageSize).
func (h *Handlers) AdminListCollections(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pageQuery(r)

	result, err := h.catalog.ListCollectionsPaged(r.Context(), true, page, pageSize)
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, pagedDTO[collectionDTO]{
		Items:    toCollectionDTOs(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// AdminCreateCollection handles POST /api/v1/admin/collections.
func (h *Handlers) AdminCreateCollection(w http.ResponseWriter, r *http.Request) {
	var req collectionRequest
	if !decodeBody(w, r, &req) {
		return
	}

	collection, err := h.catalog.CreateCollection(r.Context(), req.Name, req.Note)
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusCreated, toCollectionDTO(*collection))
}

// AdminUpdateCollection handles PUT /api/v1/admin/collections/{id}.
func (h *Handlers) AdminUpdateCollection(w http.ResponseWriter, r *http.Request) {
	var req collectionRequest
	if !decodeBody(w, r, &req) {
		return
	}

	collection, err := h.catalog.UpdateCollection(r.Context(), chi.URLParam(r, "id"), req.Name, req.Note)
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toCollectionDTO(*collection))
}

// AdminRetireCollection handles POST /api/v1/admin/collections/{id}/retire.
func (h *Handlers) AdminRetireCollection(w http.ResponseWriter, r *http.Request) {
	h.collectionStatusChange(w, r, h.catalog.RetireCollection)
}

// AdminRestoreCollection handles POST /api/v1/admin/collections/{id}/restore.
func (h *Handlers) AdminRestoreCollection(w http.ResponseWriter, r *http.Request) {
	h.collectionStatusChange(w, r, h.catalog.RestoreCollection)
}

func (h *Handlers) collectionStatusChange(
	w http.ResponseWriter,
	r *http.Request,
	change func(ctx context.Context, id string) error,
) {
	err := change(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AdminDeleteCollection handles DELETE /api/v1/admin/collections/{id} —
// permanent, removes the collection and its designs.
func (h *Handlers) AdminDeleteCollection(w http.ResponseWriter, r *http.Request) {
	err := h.catalog.DeleteCollection(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AdminListDesigns handles GET
// /api/v1/admin/designs?collection={id}&q={term}&page={n}&pageSize={n},
// paginated, with filters preserved.
func (h *Handlers) AdminListDesigns(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pageQuery(r)

	result, err := h.catalog.ListDesignsPaged(r.Context(), domain.DesignFilter{
		CollectionID:   r.URL.Query().Get("collection"),
		Query:          r.URL.Query().Get("q"),
		IncludeRetired: true,
	}, page, pageSize)
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, pagedDTO[designDTO]{
		Items:    toDesignDTOs(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// AdminGetDesign handles GET /api/v1/admin/designs/{id} (any status).
func (h *Handlers) AdminGetDesign(w http.ResponseWriter, r *http.Request) {
	design, err := h.catalog.GetDesignByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toDesignDTO(*design))
}

// AdminCreateDesign handles POST /api/v1/admin/designs.
func (h *Handlers) AdminCreateDesign(w http.ResponseWriter, r *http.Request) {
	var req designRequest
	if !decodeBody(w, r, &req) {
		return
	}

	design, err := h.catalog.CreateDesign(r.Context(), req.toInput())
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusCreated, toDesignDTO(*design))
}

// AdminUpdateDesign handles PUT /api/v1/admin/designs/{id}.
func (h *Handlers) AdminUpdateDesign(w http.ResponseWriter, r *http.Request) {
	var req designRequest
	if !decodeBody(w, r, &req) {
		return
	}

	design, err := h.catalog.UpdateDesign(r.Context(), chi.URLParam(r, "id"), req.toInput())
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toDesignDTO(*design))
}

type designIDsRequest struct {
	IDs []string `json:"ids"`
}

// AdminRetireDesigns handles POST /api/v1/admin/designs/retire {ids}.
func (h *Handlers) AdminRetireDesigns(w http.ResponseWriter, r *http.Request) {
	h.designStatusChange(w, r, h.catalog.RetireDesigns)
}

// AdminRestoreDesigns handles POST /api/v1/admin/designs/restore {ids}.
func (h *Handlers) AdminRestoreDesigns(w http.ResponseWriter, r *http.Request) {
	h.designStatusChange(w, r, h.catalog.RestoreDesigns)
}

func (h *Handlers) designStatusChange(
	w http.ResponseWriter,
	r *http.Request,
	change func(ctx context.Context, ids []string) error,
) {
	var req designIDsRequest
	if !decodeBody(w, r, &req) {
		return
	}

	err := change(r.Context(), req.IDs)
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AdminDeleteDesign handles DELETE /api/v1/admin/designs/{id} — permanent.
func (h *Handlers) AdminDeleteDesign(w http.ResponseWriter, r *http.Request) {
	err := h.catalog.DeleteDesign(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondCatalogError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
