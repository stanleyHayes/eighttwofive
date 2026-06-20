package mongostore

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

type roleDoc struct {
	Key         string   `bson:"_id"`
	Name        string   `bson:"name"`
	Description string   `bson:"description"`
	Permissions []string `bson:"permissions"`
	System      bool     `bson:"system"`
	AdminArea   bool     `bson:"adminArea"`
}

func (d roleDoc) toDomain() domain.RoleDef {
	perms := make([]domain.Permission, 0, len(d.Permissions))
	for _, p := range d.Permissions {
		perms = append(perms, domain.Permission(p))
	}

	return domain.RoleDef{
		Key:         d.Key,
		Name:        d.Name,
		Description: d.Description,
		Permissions: perms,
		System:      d.System,
		AdminArea:   d.AdminArea,
	}
}

func roleToDoc(r *domain.RoleDef) roleDoc {
	perms := make([]string, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, string(p))
	}

	return roleDoc{
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
		System:      r.System,
		AdminArea:   r.AdminArea,
	}
}

// RoleRepository implements domain.RoleRepository on MongoDB. Each role is one
// document keyed by its stable role key.
type RoleRepository struct {
	col *mongo.Collection
}

// NewRoleRepository returns a repository over the roles collection.
func NewRoleRepository(db *mongo.Database) *RoleRepository {
	return &RoleRepository{col: db.Collection("roles")}
}

// EnsureIndexes seeds the built-in roles (insert-if-missing, so admin edits to
// a built-in's permissions survive restarts) and is a no-op for indexes since
// the role key is the document _id.
func (r *RoleRepository) EnsureIndexes(ctx context.Context) error {
	for _, def := range domain.BuiltInRoles() {
		doc := roleToDoc(&def)

		_, err := r.col.UpdateOne(ctx,
			bson.M{"_id": def.Key},
			bson.M{"$setOnInsert": bson.M{
				"name":        doc.Name,
				"description": doc.Description,
				"permissions": doc.Permissions,
				"system":      doc.System,
				"adminArea":   doc.AdminArea,
			}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return fmt.Errorf("seed role %q: %w", def.Key, err)
		}
	}

	return nil
}

// List returns every role definition, ordered by key for a stable UI.
func (r *RoleRepository) List(ctx context.Context) ([]domain.RoleDef, error) {
	cur, err := r.col.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find roles: %w", err)
	}

	var docs []roleDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode roles: %w", err)
	}

	roles := make([]domain.RoleDef, 0, len(docs))
	for _, doc := range docs {
		roles = append(roles, doc.toDomain())
	}

	return roles, nil
}

// Get returns one role definition by key, or ErrNotFound.
func (r *RoleRepository) Get(ctx context.Context, key string) (*domain.RoleDef, error) {
	var doc roleDoc

	err := r.col.FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find role: %w", err)
	}

	role := doc.toDomain()

	return &role, nil
}

// Upsert creates or replaces a role definition.
func (r *RoleRepository) Upsert(ctx context.Context, role *domain.RoleDef) error {
	doc := roleToDoc(role)

	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": role.Key}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("replace role: %w", err)
	}

	return nil
}

// Delete removes a role definition by key.
func (r *RoleRepository) Delete(ctx context.Context, key string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": key})
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}

	return nil
}
