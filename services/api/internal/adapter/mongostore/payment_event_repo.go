package mongostore

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

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

// EnsureIndexes creates the audit indexes.
func (r *PaymentEventRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "providerRef", Value: 1}},
			Options: nil,
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
	if err != nil {
		return fmt.Errorf("record payment event: %w", err)
	}

	return nil
}
