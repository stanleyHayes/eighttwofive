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

type userDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Email     string        `bson:"email"`
	Name      string        `bson:"name"`
	Role      string        `bson:"role"`
	CreatedAt time.Time     `bson:"createdAt"`
}

func (d userDoc) toDomain() *domain.User {
	return &domain.User{
		ID:        d.ID.Hex(),
		Email:     d.Email,
		Name:      d.Name,
		Role:      domain.Role(d.Role),
		CreatedAt: d.CreatedAt,
	}
}

// UserRepository implements domain.UserRepository on MongoDB.
type UserRepository struct {
	col *mongo.Collection
}

// NewUserRepository returns a repository bound to the users collection.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

// EnsureIndexes creates the unique email index. Call once at startup.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create user email index: %w", err)
	}

	return nil
}

// Upsert creates the user when the email is new, otherwise backfills the
// stored identity. An admin role promotes an existing user; never demotes.
func (r *UserRepository) Upsert(ctx context.Context, u *domain.User) error {
	onInsert := bson.M{"email": u.Email, "name": u.Name, "createdAt": u.CreatedAt}
	update := bson.M{"$setOnInsert": onInsert}

	// $set and $setOnInsert cannot share a path, so role lives in exactly one.
	if u.Role == domain.RoleAdmin {
		update["$set"] = bson.M{"role": string(domain.RoleAdmin)}
	} else {
		onInsert["role"] = string(u.Role)
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var doc userDoc

	err := r.col.FindOneAndUpdate(ctx, bson.M{"email": u.Email}, update, opts).Decode(&doc)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	*u = *doc.toDomain()

	return nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	total, err := r.col.CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return total, nil
}

// ListPaged returns one page of users, newest first.
func (r *UserRepository) ListPaged(ctx context.Context, params domain.PageParams) ([]domain.User, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(params.Skip()).
		SetLimit(params.Limit())

	cursor, err := r.col.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	defer func() { _ = cursor.Close(ctx) }()

	var docs []userDoc

	err = cursor.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode users: %w", err)
	}

	users := make([]domain.User, 0, len(docs))
	for i := range docs {
		users = append(users, *docs[i].toDomain())
	}

	return users, nil
}

// UpdateRole sets a user's role unconditionally.
func (r *UserRepository) UpdateRole(ctx context.Context, id string, role domain.Role) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	result, err := r.col.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": bson.M{"role": string(role)}})
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetByID loads a user by its hex ObjectID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc userDoc

	err = r.col.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	return doc.toDomain(), nil
}
