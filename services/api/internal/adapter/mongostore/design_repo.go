package mongostore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

type photoDoc struct {
	PublicID string `bson:"publicId"`
	Order    int    `bson:"order"`
}

type sizeBandDoc struct {
	Label        string            `bson:"label"`
	PricePesewas int64             `bson:"pricePesewas"`
	Chart        map[string]string `bson:"chart"`
}

type designDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	CollectionID bson.ObjectID `bson:"collectionId"`
	Name         string        `bson:"name"`
	Slug         string        `bson:"slug"`
	Note         string        `bson:"note"`
	Photos       []photoDoc    `bson:"photos"`
	SizeBands    []sizeBandDoc `bson:"sizeBands"`
	Status       string        `bson:"status"`
	CreatedAt    time.Time     `bson:"createdAt"`
	RetiredAt    *time.Time    `bson:"retiredAt,omitempty"`
}

func (d designDoc) toDomain() *domain.Design {
	photos := make([]domain.Photo, 0, len(d.Photos))
	for _, p := range d.Photos {
		photos = append(photos, domain.Photo{PublicID: p.PublicID, Order: p.Order})
	}

	bands := make([]domain.SizeBand, 0, len(d.SizeBands))
	for _, b := range d.SizeBands {
		bands = append(bands, domain.SizeBand{Label: b.Label, PricePesewas: b.PricePesewas, Chart: b.Chart})
	}

	return &domain.Design{
		ID:           d.ID.Hex(),
		CollectionID: d.CollectionID.Hex(),
		Name:         d.Name,
		Slug:         d.Slug,
		Note:         d.Note,
		Photos:       photos,
		SizeBands:    bands,
		Status:       domain.Status(d.Status),
		CreatedAt:    d.CreatedAt,
		RetiredAt:    d.RetiredAt,
	}
}

func designToDoc(d *domain.Design) (*designDoc, error) {
	collectionID, err := bson.ObjectIDFromHex(d.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("%w: bad collection id", domain.ErrInvalidInput)
	}

	photos := make([]photoDoc, 0, len(d.Photos))
	for _, p := range d.Photos {
		photos = append(photos, photoDoc{PublicID: p.PublicID, Order: p.Order})
	}

	bands := make([]sizeBandDoc, 0, len(d.SizeBands))
	for _, b := range d.SizeBands {
		bands = append(bands, sizeBandDoc{Label: b.Label, PricePesewas: b.PricePesewas, Chart: b.Chart})
	}

	return &designDoc{
		ID:           bson.ObjectID{},
		CollectionID: collectionID,
		Name:         d.Name,
		Slug:         d.Slug,
		Note:         d.Note,
		Photos:       photos,
		SizeBands:    bands,
		Status:       string(d.Status),
		CreatedAt:    d.CreatedAt,
		RetiredAt:    d.RetiredAt,
	}, nil
}

// DesignRepository implements domain.DesignRepository on MongoDB.
type DesignRepository struct {
	col *mongo.Collection
}

// NewDesignRepository returns a repository over the designs collection.
func NewDesignRepository(db *mongo.Database) *DesignRepository {
	return &DesignRepository{col: db.Collection("designs")}
}

// EnsureIndexes creates the unique slug index, the collection index, and the
// text index used for storefront search.
func (r *DesignRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "collectionId", Value: 1}},
			Options: nil,
		},
		{
			Keys:    bson.D{{Key: "name", Value: "text"}, {Key: "note", Value: "text"}},
			Options: nil,
		},
	})
	if err != nil {
		return fmt.Errorf("create design indexes: %w", err)
	}

	return nil
}

// Create inserts a design; a slug clash maps to ErrDuplicateSlug.
func (r *DesignRepository) Create(ctx context.Context, d *domain.Design) error {
	doc, err := designToDoc(d)
	if err != nil {
		return err
	}

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicateSlug
		}

		return fmt.Errorf("insert design: %w", err)
	}

	if id, ok := res.InsertedID.(bson.ObjectID); ok {
		d.ID = id.Hex()
	}

	return nil
}

// Update replaces the editable fields of a design.
func (r *DesignRepository) Update(ctx context.Context, d *domain.Design) error {
	objectID, err := bson.ObjectIDFromHex(d.ID)
	if err != nil {
		return domain.ErrNotFound
	}

	doc, err := designToDoc(d)
	if err != nil {
		return err
	}

	res, err := r.col.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": bson.M{
		"collectionId": doc.CollectionID,
		"name":         doc.Name,
		"note":         doc.Note,
		"photos":       doc.Photos,
		"sizeBands":    doc.SizeBands,
	}})
	if err != nil {
		return fmt.Errorf("update design: %w", err)
	}

	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetByID loads a design by hex id.
func (r *DesignRepository) GetByID(ctx context.Context, id string) (*domain.Design, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return r.getOne(ctx, bson.M{"_id": objectID})
}

// GetBySlug loads a design by slug.
func (r *DesignRepository) GetBySlug(ctx context.Context, slug string) (*domain.Design, error) {
	return r.getOne(ctx, bson.M{"slug": slug})
}

// List returns designs newest first, narrowed by the filter.
func (r *DesignRepository) List(ctx context.Context, filter domain.DesignFilter) ([]domain.Design, error) {
	query, err := designQuery(filter)
	if err != nil {
		return nil, err
	}

	return r.find(ctx, query, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
}

// Count returns the number of designs matching the filter.
func (r *DesignRepository) Count(ctx context.Context, filter domain.DesignFilter) (int64, error) {
	query, err := designQuery(filter)
	if err != nil {
		return 0, err
	}

	total, err := r.col.CountDocuments(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("count designs: %w", err)
	}

	return total, nil
}

// ListPaged returns one page of designs matching the filter, newest first.
func (r *DesignRepository) ListPaged(
	ctx context.Context, filter domain.DesignFilter, params domain.PageParams,
) ([]domain.Design, error) {
	query, err := designQuery(filter)
	if err != nil {
		return nil, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(params.Skip()).
		SetLimit(params.Limit())

	return r.find(ctx, query, opts)
}

// designQuery builds the Mongo filter for a design listing.
func designQuery(filter domain.DesignFilter) (bson.M, error) {
	query := bson.M{}

	if !filter.IncludeRetired {
		query["status"] = string(domain.StatusLive)
	}

	if filter.CollectionID != "" {
		collectionID, err := bson.ObjectIDFromHex(filter.CollectionID)
		if err != nil {
			return nil, domain.ErrNotFound
		}

		query["collectionId"] = collectionID
	}

	if filter.Query != "" {
		query["$text"] = bson.M{"$search": filter.Query}
	}

	return query, nil
}

// SetStatusBulk flips the lifecycle state of several designs at once.
func (r *DesignRepository) SetStatusBulk(ctx context.Context, ids []string, status domain.Status, at time.Time) error {
	objectIDs := make([]bson.ObjectID, 0, len(ids))

	for _, id := range ids {
		objectID, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return domain.ErrNotFound
		}

		objectIDs = append(objectIDs, objectID)
	}

	_, err := r.col.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": objectIDs}}, statusUpdate(status, at))
	if err != nil {
		return fmt.Errorf("set designs status: %w", err)
	}

	return nil
}

// SetStatusByCollection flips every design in a collection.
func (r *DesignRepository) SetStatusByCollection(
	ctx context.Context, collectionID string, status domain.Status, at time.Time,
) error {
	objectID, err := bson.ObjectIDFromHex(collectionID)
	if err != nil {
		return domain.ErrNotFound
	}

	_, err = r.col.UpdateMany(ctx, bson.M{"collectionId": objectID}, statusUpdate(status, at))
	if err != nil {
		return fmt.Errorf("set collection designs status: %w", err)
	}

	return nil
}

func statusUpdate(status domain.Status, at time.Time) bson.M {
	if status == domain.StatusRetired {
		return bson.M{"$set": bson.M{"status": string(status), "retiredAt": at}}
	}

	return bson.M{"$set": bson.M{"status": string(status)}, "$unset": bson.M{"retiredAt": ""}}
}

// Delete permanently removes a design.
func (r *DesignRepository) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	_, err = r.col.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("delete design: %w", err)
	}

	return nil
}

// DeleteByCollection permanently removes every design in a collection.
func (r *DesignRepository) DeleteByCollection(ctx context.Context, collectionID string) error {
	objectID, err := bson.ObjectIDFromHex(collectionID)
	if err != nil {
		return domain.ErrNotFound
	}

	_, err = r.col.DeleteMany(ctx, bson.M{"collectionId": objectID})
	if err != nil {
		return fmt.Errorf("delete collection designs: %w", err)
	}

	return nil
}

func (r *DesignRepository) getOne(ctx context.Context, filter bson.M) (*domain.Design, error) {
	var doc designDoc

	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find design: %w", err)
	}

	return doc.toDomain(), nil
}

func (r *DesignRepository) find(
	ctx context.Context, query bson.M, opts *options.FindOptionsBuilder,
) ([]domain.Design, error) {
	cur, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("find designs: %w", err)
	}

	var docs []designDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode designs: %w", err)
	}

	designs := make([]domain.Design, 0, len(docs))
	for _, doc := range docs {
		designs = append(designs, *doc.toDomain())
	}

	return designs, nil
}
