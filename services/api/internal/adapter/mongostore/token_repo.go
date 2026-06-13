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

// TokenRepository implements domain.TokenRepository on MongoDB. Login tokens
// and sessions are stored hashed, in separate collections, each with a TTL
// index so expired documents are removed by the server.
type TokenRepository struct {
	loginTokens *mongo.Collection
	sessions    *mongo.Collection
	now         func() time.Time
}

// NewTokenRepository returns a repository over the token collections.
func NewTokenRepository(db *mongo.Database) *TokenRepository {
	return &TokenRepository{
		loginTokens: db.Collection("login_tokens"),
		sessions:    db.Collection("sessions"),
		now:         time.Now,
	}
}

// EnsureIndexes creates unique hash indexes and TTL expiry indexes.
func (r *TokenRepository) EnsureIndexes(ctx context.Context) error {
	for _, col := range []*mongo.Collection{r.loginTokens, r.sessions} {
		_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "tokenHash", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "expiresAt", Value: 1}},
				Options: options.Index().SetExpireAfterSeconds(0),
			},
		})
		if err != nil {
			return fmt.Errorf("create token indexes for %s: %w", col.Name(), err)
		}
	}

	return nil
}

// StoreLoginToken stores a hashed single-use login token.
func (r *TokenRepository) StoreLoginToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := r.loginTokens.InsertOne(ctx, bson.M{
		"tokenHash": tokenHash,
		"userId":    userID,
		"expiresAt": expiresAt,
		"usedAt":    nil,
	})
	if err != nil {
		return fmt.Errorf("insert login token: %w", err)
	}

	return nil
}

// ConsumeLoginToken atomically marks the token used and returns its user ID.
func (r *TokenRepository) ConsumeLoginToken(ctx context.Context, tokenHash string) (string, error) {
	filter := bson.M{
		"tokenHash": tokenHash,
		"usedAt":    nil,
		"expiresAt": bson.M{"$gt": r.now()},
	}
	update := bson.M{"$set": bson.M{"usedAt": r.now()}}

	var doc struct {
		UserID string `bson:"userId"`
	}

	err := r.loginTokens.FindOneAndUpdate(ctx, filter, update).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", domain.ErrTokenInvalid
	}

	if err != nil {
		return "", fmt.Errorf("consume login token: %w", err)
	}

	return doc.UserID, nil
}

// CreateSession stores a hashed session token.
func (r *TokenRepository) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := r.sessions.InsertOne(ctx, bson.M{
		"tokenHash": tokenHash,
		"userId":    userID,
		"expiresAt": expiresAt,
	})
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// GetSession returns the user ID for a live session.
func (r *TokenRepository) GetSession(ctx context.Context, tokenHash string) (string, error) {
	filter := bson.M{
		"tokenHash": tokenHash,
		"expiresAt": bson.M{"$gt": r.now()},
	}

	var doc struct {
		UserID string `bson:"userId"`
	}

	err := r.sessions.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", domain.ErrTokenInvalid
	}

	if err != nil {
		return "", fmt.Errorf("find session: %w", err)
	}

	return doc.UserID, nil
}

// DeleteSession revokes a session; deleting a missing session is not an error.
func (r *TokenRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.sessions.DeleteOne(ctx, bson.M{"tokenHash": tokenHash})
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
