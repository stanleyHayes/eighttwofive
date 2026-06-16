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

type visitDoc struct {
	ID               bson.ObjectID `bson:"_id,omitempty"`
	OrderID          string        `bson:"orderId"`
	SlotID           bson.ObjectID `bson:"slotId"`
	DepositPaymentID string        `bson:"depositPaymentId"`
	Status           string        `bson:"status"`
	HoldExpiresAt    *time.Time    `bson:"holdExpiresAt,omitempty"`
	CreatedAt        time.Time     `bson:"createdAt"`
	UpdatedAt        time.Time     `bson:"updatedAt"`
}

func (d visitDoc) toDomain() *domain.Visit {
	return &domain.Visit{
		ID:               d.ID.Hex(),
		OrderID:          d.OrderID,
		SlotID:           d.SlotID.Hex(),
		DepositPaymentID: d.DepositPaymentID,
		Status:           domain.VisitStatus(d.Status),
		HoldExpiresAt:    d.HoldExpiresAt,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func visitToDoc(visit *domain.Visit) (*visitDoc, error) {
	slotID, err := bson.ObjectIDFromHex(visit.SlotID)
	if err != nil {
		return nil, fmt.Errorf("%w: bad slot id", domain.ErrInvalidInput)
	}

	var holdExpiresAt *time.Time

	if visit.HoldExpiresAt != nil {
		expiry := visit.HoldExpiresAt.UTC()
		holdExpiresAt = &expiry
	}

	return &visitDoc{
		ID:               bson.ObjectID{},
		OrderID:          visit.OrderID,
		SlotID:           slotID,
		DepositPaymentID: visit.DepositPaymentID,
		Status:           string(visit.Status),
		HoldExpiresAt:    holdExpiresAt,
		CreatedAt:        visit.CreatedAt.UTC(),
		UpdatedAt:        visit.UpdatedAt.UTC(),
	}, nil
}

// VisitRepository implements domain.VisitRepository on MongoDB.
type VisitRepository struct {
	col   *mongo.Collection
	slots *mongo.Collection
}

// NewVisitRepository returns a repository over the visits collection.
func NewVisitRepository(db *mongo.Database) *VisitRepository {
	return &VisitRepository{
		col:   db.Collection("visits"),
		slots: db.Collection("slots"),
	}
}

// Mongo server error codes returned when an index exists with a different
// definition (options or key spec) than the one being created.
const (
	indexOptionsConflictCode  = 85
	indexKeySpecsConflictCode = 86
)

// EnsureIndexes creates the required visit indexes. The slotId unique index is
// partial so only active (booked/done) visits claim a slot: a cancelled visit
// must never block the slot from being rebooked. A pre-existing legacy
// non-partial index is dropped and recreated.
func (r *VisitRepository) EnsureIndexes(ctx context.Context) error {
	err := r.createIndexes(ctx)
	if err == nil {
		return nil
	}

	if !isIndexConflict(err) {
		return fmt.Errorf("create visit indexes: %w", err)
	}

	err = r.col.Indexes().DropOne(ctx, "slotId_1")
	if err != nil {
		return fmt.Errorf("drop legacy slot index: %w", err)
	}

	err = r.createIndexes(ctx)
	if err != nil {
		return fmt.Errorf("create visit indexes: %w", err)
	}

	return nil
}

func isIndexConflict(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == indexOptionsConflictCode || cmdErr.Code == indexKeySpecsConflictCode
	}

	return false
}

// Create inserts a visit. It fills the visit ID on success.
func (r *VisitRepository) Create(ctx context.Context, visit *domain.Visit) error {
	doc, err := visitToDoc(visit)
	if err != nil {
		return err
	}

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrSlotUnavailable
		}

		return fmt.Errorf("insert visit: %w", err)
	}

	if id, ok := res.InsertedID.(bson.ObjectID); ok {
		visit.ID = id.Hex()
	}

	return nil
}

// BookSlot claims an open slot atomically and inserts the visit. It fills the
// visit ID and returns ErrSlotUnavailable if the slot is not open or has
// already been booked. The unique index on slotId in the visits collection
// makes the double-booking race impossible even without a cross-collection
// transaction.
func (r *VisitRepository) BookSlot(ctx context.Context, slotID string, visit *domain.Visit) error {
	slotObjectID, err := bson.ObjectIDFromHex(slotID)
	if err != nil {
		return domain.ErrSlotNotFound
	}

	doc, err := visitToDoc(visit)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// Atomically claim the slot only if it is currently open.
	res, err := r.slots.UpdateOne(ctx,
		bson.M{"_id": slotObjectID, "status": string(domain.SlotStatusOpen)},
		bson.M{"$set": bson.M{
			"status":    string(domain.SlotStatusBooked),
			"updatedAt": now,
		}},
	)
	if err != nil {
		return fmt.Errorf("claim slot: %w", err)
	}

	if res.MatchedCount == 0 {
		return domain.ErrSlotUnavailable
	}

	insertRes, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		// Best-effort rollback: the slot was claimed but the visit could not be
		// created (e.g. a duplicate slotId from a race). Reopen the slot so it
		// does not stay orphaned.
		if mongo.IsDuplicateKeyError(err) {
			rollback := bson.M{"$set": bson.M{
				"status":    string(domain.SlotStatusOpen),
				"updatedAt": time.Now().UTC(),
			}}

			_, rbErr := r.slots.UpdateOne(ctx, bson.M{"_id": slotObjectID}, rollback)
			if rbErr != nil {
				return fmt.Errorf("book slot: %w (rollback failed: %w)", domain.ErrSlotUnavailable, rbErr)
			}

			return domain.ErrSlotUnavailable
		}

		return fmt.Errorf("insert visit: %w", err)
	}

	if id, ok := insertRes.InsertedID.(bson.ObjectID); ok {
		doc.ID = id
	}

	visit.ID = doc.ID.Hex()

	return nil
}

// GetByID loads a visit by its hex id.
func (r *VisitRepository) GetByID(ctx context.Context, id string) (*domain.Visit, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return r.getOne(ctx, bson.M{"_id": objectID})
}

// GetByOrderID loads a visit by its order reference.
func (r *VisitRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Visit, error) {
	return r.getOne(ctx, bson.M{"orderId": orderID})
}

// List returns visits matching the filter, newest first.
func (r *VisitRepository) List(ctx context.Context, filter domain.VisitFilter) ([]domain.Visit, error) {
	query := bson.M{}

	if filter.Status != "" {
		query["status"] = string(filter.Status)
	}

	if filter.SlotID != "" {
		slotID, err := bson.ObjectIDFromHex(filter.SlotID)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}

		query["slotId"] = slotID
	}

	return r.find(ctx, query)
}

// ListExpiredHolds returns booked visits whose deposit hold lapsed before now.
// Visits without a hold (confirmed bookings) never match.
func (r *VisitRepository) ListExpiredHolds(ctx context.Context, now time.Time) ([]domain.Visit, error) {
	query := bson.M{
		"status":        string(domain.VisitStatusBooked),
		"holdExpiresAt": bson.M{"$lte": now.UTC()},
	}

	return r.find(ctx, query)
}

// Update replaces an existing visit.
func (r *VisitRepository) Update(ctx context.Context, visit *domain.Visit) error {
	doc, err := visitToDoc(visit)
	if err != nil {
		return err
	}

	objectID, err := bson.ObjectIDFromHex(visit.ID)
	if err != nil {
		return domain.ErrNotFound
	}

	doc.ID = objectID

	res, err := r.col.ReplaceOne(ctx, bson.M{"_id": objectID}, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrSlotUnavailable
		}

		return fmt.Errorf("update visit: %w", err)
	}

	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *VisitRepository) createIndexes(ctx context.Context) error {
	activeStatuses := bson.D{{Key: "status", Value: bson.D{{Key: "$in", Value: []string{
		string(domain.VisitStatusBooked),
		string(domain.VisitStatusDone),
	}}}}}

	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "slotId", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(activeStatuses),
		},
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: nil,
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: nil,
		},
	})
	if err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}

func (r *VisitRepository) find(ctx context.Context, query bson.M) ([]domain.Visit, error) {
	cur, err := r.col.Find(ctx, query, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(maxListResults))
	if err != nil {
		return nil, fmt.Errorf("find visits: %w", err)
	}

	var docs []visitDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode visits: %w", err)
	}

	visits := make([]domain.Visit, 0, len(docs))
	for _, doc := range docs {
		visits = append(visits, *doc.toDomain())
	}

	return visits, nil
}

func (r *VisitRepository) getOne(ctx context.Context, filter bson.M) (*domain.Visit, error) {
	var doc visitDoc

	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find visit: %w", err)
	}

	return doc.toDomain(), nil
}
