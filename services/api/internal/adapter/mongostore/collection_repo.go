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

type collectionDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Name      string        `bson:"name"`
	Slug      string        `bson:"slug"`
	Note      string        `bson:"note"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"createdAt"`
	RetiredAt *time.Time    `bson:"retiredAt,omitempty"`
}

func (d collectionDoc) toDomain() *domain.Collection {
	return &domain.Collection{
		ID:        d.ID.Hex(),
		Name:      d.Name,
		Slug:      d.Slug,
		Note:      d.Note,
		Status:    domain.Status(d.Status),
		CreatedAt: d.CreatedAt,
		RetiredAt: d.RetiredAt,
	}
}

// CollectionRepository implements domain.CollectionRepository on MongoDB.
type CollectionRepository struct {
	col *mongo.Collection
}

// NewCollectionRepository returns a repository over the collections collection.
func NewCollectionRepository(db *mongo.Database) *CollectionRepository {
	return &CollectionRepository{col: db.Collection("collections")}
}

// EnsureIndexes creates the unique slug index.
func (r *CollectionRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create collection slug index: %w", err)
	}

	return nil
}

// Create inserts a collection; a slug clash maps to ErrDuplicateSlug.
func (r *CollectionRepository) Create(ctx context.Context, c *domain.Collection) error {
	doc := collectionDoc{
		ID:        bson.ObjectID{},
		Name:      c.Name,
		Slug:      c.Slug,
		Note:      c.Note,
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt,
		RetiredAt: c.RetiredAt,
	}

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicateSlug
		}

		return fmt.Errorf("insert collection: %w", err)
	}

	if id, ok := res.InsertedID.(bson.ObjectID); ok {
		c.ID = id.Hex()
	}

	return nil
}

// Update renames/re-notes a collection.
func (r *CollectionRepository) Update(ctx context.Context, id, name, note string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	res, err := r.col.UpdateOne(ctx, bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"name": name, "note": note}})
	if err != nil {
		return fmt.Errorf("update collection: %w", err)
	}

	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetByID loads a collection by hex id.
func (r *CollectionRepository) GetByID(ctx context.Context, id string) (*domain.Collection, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return r.getOne(ctx, bson.M{"_id": objectID})
}

// GetBySlug loads a collection by slug.
func (r *CollectionRepository) GetBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	return r.getOne(ctx, bson.M{"slug": slug})
}

// List returns collections, newest first; retired only when asked.
func (r *CollectionRepository) List(ctx context.Context, includeRetired bool) ([]domain.Collection, error) {
	return r.find(ctx, collectionQuery(includeRetired),
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetLimit(maxListResults))
}

// Count returns the number of collections matching includeRetired.
func (r *CollectionRepository) Count(ctx context.Context, includeRetired bool) (int64, error) {
	total, err := r.col.CountDocuments(ctx, collectionQuery(includeRetired))
	if err != nil {
		return 0, fmt.Errorf("count collections: %w", err)
	}

	return total, nil
}

// ListPaged returns one page of collections, newest first.
func (r *CollectionRepository) ListPaged(
	ctx context.Context, includeRetired bool, params domain.PageParams,
) ([]domain.Collection, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(params.Skip()).
		SetLimit(params.Limit())

	return r.find(ctx, collectionQuery(includeRetired), opts)
}

// collectionQuery builds the Mongo filter for a collection listing.
func collectionQuery(includeRetired bool) bson.M {
	filter := bson.M{}
	if !includeRetired {
		filter["status"] = string(domain.StatusLive)
	}

	return filter
}

// SetStatus flips a collection's lifecycle state.
func (r *CollectionRepository) SetStatus(ctx context.Context, id string, status domain.Status, at time.Time) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	update := bson.M{"$set": bson.M{"status": string(status)}, "$unset": bson.M{"retiredAt": ""}}
	if status == domain.StatusRetired {
		update = bson.M{"$set": bson.M{"status": string(status), "retiredAt": at}}
	}

	res, err := r.col.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("set collection status: %w", err)
	}

	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete permanently removes a collection.
func (r *CollectionRepository) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	_, err = r.col.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}

	return nil
}

func (r *CollectionRepository) getOne(ctx context.Context, filter bson.M) (*domain.Collection, error) {
	var doc collectionDoc

	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find collection: %w", err)
	}

	return doc.toDomain(), nil
}

func (r *CollectionRepository) find(
	ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder,
) ([]domain.Collection, error) {
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find collections: %w", err)
	}

	var docs []collectionDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode collections: %w", err)
	}

	collections := make([]domain.Collection, 0, len(docs))
	for _, doc := range docs {
		collections = append(collections, *doc.toDomain())
	}

	return collections, nil
}
