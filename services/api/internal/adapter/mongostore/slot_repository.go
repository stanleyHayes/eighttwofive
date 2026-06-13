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

type slotDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Start     time.Time     `bson:"start"`
	End       time.Time     `bson:"end"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

func (d slotDoc) toDomain() *domain.Slot {
	return &domain.Slot{
		ID:        d.ID.Hex(),
		Start:     d.Start,
		End:       d.End,
		Status:    domain.SlotStatus(d.Status),
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func slotToDoc(s *domain.Slot) *slotDoc {
	return &slotDoc{
		ID:        bson.ObjectID{},
		Start:     s.Start.UTC(),
		End:       s.End.UTC(),
		Status:    string(s.Status),
		CreatedAt: s.CreatedAt.UTC(),
		UpdatedAt: s.UpdatedAt.UTC(),
	}
}

// SlotRepository implements domain.SlotRepository on MongoDB.
type SlotRepository struct {
	col *mongo.Collection
}

// NewSlotRepository returns a repository over the slots collection.
func NewSlotRepository(db *mongo.Database) *SlotRepository {
	return &SlotRepository{col: db.Collection("slots")}
}

// EnsureIndexes creates the required slot indexes.
func (r *SlotRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "start", Value: 1}, {Key: "end", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: nil,
		},
	})
	if err != nil {
		return fmt.Errorf("create slot indexes: %w", err)
	}

	return nil
}

// Create inserts a new slot.
func (r *SlotRepository) Create(ctx context.Context, s *domain.Slot) error {
	doc := slotToDoc(s)

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: a slot already exists for this time window", domain.ErrInvalidInput)
		}

		return fmt.Errorf("insert slot: %w", err)
	}

	if id, ok := res.InsertedID.(bson.ObjectID); ok {
		s.ID = id.Hex()
	}

	return nil
}

// GetByID loads a slot by its hex id.
func (r *SlotRepository) GetByID(ctx context.Context, id string) (*domain.Slot, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return r.getOne(ctx, bson.M{"_id": objectID})
}

// List returns slots matching the filter, newest first.
func (r *SlotRepository) List(ctx context.Context, filter domain.SlotFilter) ([]domain.Slot, error) {
	query := bson.M{}

	if filter.Status != "" {
		query["status"] = string(filter.Status)
	}

	if !filter.After.IsZero() {
		query["start"] = bson.M{"$gte": filter.After}
	}

	if !filter.Before.IsZero() {
		query["end"] = bson.M{"$lte": filter.Before}
	}

	cur, err := r.col.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "start", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find slots: %w", err)
	}

	var docs []slotDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode slots: %w", err)
	}

	slots := make([]domain.Slot, 0, len(docs))
	for _, doc := range docs {
		slots = append(slots, *doc.toDomain())
	}

	return slots, nil
}

// UpdateStatusFrom atomically moves a slot from one status to another. The
// filter includes the expected current status so a concurrent writer that got
// there first makes this call return ErrSlotUnavailable instead of stomping
// the other side's claim.
func (r *SlotRepository) UpdateStatusFrom(ctx context.Context, id string, from, to domain.SlotStatus) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	res, err := r.col.UpdateOne(ctx,
		bson.M{"_id": objectID, "status": string(from)},
		bson.M{"$set": bson.M{
			"status":    string(to),
			"updatedAt": time.Now().UTC(),
		}},
	)
	if err != nil {
		return fmt.Errorf("update slot: %w", err)
	}

	if res.MatchedCount == 0 {
		// Distinguish a missing slot from a status that moved underneath us.
		_, getErr := r.getOne(ctx, bson.M{"_id": objectID})
		if getErr != nil {
			return getErr
		}

		return domain.ErrSlotUnavailable
	}

	return nil
}

func (r *SlotRepository) getOne(ctx context.Context, filter bson.M) (*domain.Slot, error) {
	var doc slotDoc

	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find slot: %w", err)
	}

	return doc.toDomain(), nil
}
