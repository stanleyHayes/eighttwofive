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

type paymentEventDoc struct {
	ProviderRef string    `bson:"providerRef"`
	Provider    string    `bson:"provider"`
	Type        string    `bson:"type"`
	Payload     []byte    `bson:"payload"`
	CreatedAt   time.Time `bson:"createdAt"`
}

// PaymentEventRepository implements domain.PaymentEventRepository on MongoDB.
type PaymentEventRepository struct {
	col *mongo.Collection
}

// NewPaymentEventRepository returns a repository over the payments collection.
func NewPaymentEventRepository(db *mongo.Database) *PaymentEventRepository {
	return &PaymentEventRepository{col: db.Collection("payments")}
}

// EnsureIndexes creates the audit indexes. The (providerRef, type) index is
// unique so a webhook redelivered by the provider records its event once
// instead of piling up duplicate audit rows. It also serves providerRef-prefix
// lookups, so no separate providerRef index is needed.
func (r *PaymentEventRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "providerRef", Value: 1}, {Key: "type", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: nil,
		},
	})
	if err != nil {
		return fmt.Errorf("create payment event indexes: %w", err)
	}

	return nil
}

// RecordEvent stores a raw provider event.
func (r *PaymentEventRepository) RecordEvent(ctx context.Context, event domain.PaymentEvent) error {
	doc := paymentEventDoc{
		ProviderRef: event.ProviderRef,
		Provider:    event.Provider,
		Type:        event.Type,
		Payload:     event.Payload,
		CreatedAt:   event.CreatedAt.UTC(),
	}

	_, err := r.col.InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		// A redelivered webhook is already on record; recording it again is a
		// no-op, not a failure.
		return nil
	}

	if err != nil {
		return fmt.Errorf("record payment event: %w", err)
	}

	return nil
}
