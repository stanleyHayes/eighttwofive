package mongostore

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

type subscriberDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Email     string        `bson:"email"`
	Name      string        `bson:"name"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// SubscriberRepository implements domain.SubscriberRepository on MongoDB.
type SubscriberRepository struct {
	col *mongo.Collection
}

// NewSubscriberRepository returns a repository bound to the subscribers collection.
func NewSubscriberRepository(db *mongo.Database) *SubscriberRepository {
	return &SubscriberRepository{col: db.Collection("subscribers")}
}

// EnsureIndexes creates the unique email index. Call once at startup.
func (r *SubscriberRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create email index: %w", err)
	}

	return nil
}

// Create inserts a subscriber and backfills the generated ID.
func (r *SubscriberRepository) Create(ctx context.Context, s *domain.Subscriber) error {
	doc := subscriberDoc{Email: s.Email, Name: s.Name, CreatedAt: s.CreatedAt}

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicateEmail
		}

		return fmt.Errorf("insert subscriber: %w", err)
	}

	if id, ok := res.InsertedID.(bson.ObjectID); ok {
		s.ID = id.Hex()
	}

	return nil
}

// List returns subscribers, newest first.
func (r *SubscriberRepository) List(ctx context.Context, limit int64) ([]domain.Subscriber, error) {
	return r.find(ctx, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(limit))
}

// Count returns the total number of subscribers.
func (r *SubscriberRepository) Count(ctx context.Context) (int64, error) {
	total, err := r.col.CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, fmt.Errorf("count subscribers: %w", err)
	}

	return total, nil
}

// ListPaged returns one page of subscribers, newest first.
func (r *SubscriberRepository) ListPaged(ctx context.Context, params domain.PageParams) ([]domain.Subscriber, error) {
	return r.find(ctx, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(params.Skip()).
		SetLimit(params.Limit()))
}

func (r *SubscriberRepository) find(
	ctx context.Context, opts *options.FindOptionsBuilder,
) ([]domain.Subscriber, error) {
	cur, err := r.col.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, fmt.Errorf("find subscribers: %w", err)
	}

	var docs []subscriberDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode subscribers: %w", err)
	}

	subs := make([]domain.Subscriber, 0, len(docs))
	for _, d := range docs {
		subs = append(subs, domain.Subscriber{
			ID:        d.ID.Hex(),
			Email:     d.Email,
			Name:      d.Name,
			CreatedAt: d.CreatedAt,
		})
	}

	return subs, nil
}
