package mongostore_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func TestRoleRepository_SeedAndCRUD(t *testing.T) {
	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewRoleRepository(db)

	// Seeding creates the four built-in roles with their default permissions.
	require.NoError(t, repo.EnsureIndexes(t.Context()))

	roles, err := repo.List(t.Context())
	require.NoError(t, err)
	assert.Len(t, roles, 4)

	admin, err := repo.Get(t.Context(), "admin")
	require.NoError(t, err)
	assert.True(t, admin.System)
	assert.True(t, admin.Has(domain.PermTeamWrite))

	// A custom role round-trips and can be deleted.
	require.NoError(t, repo.Upsert(t.Context(), &domain.RoleDef{
		Key:         "photographer",
		Name:        "Photographer",
		Permissions: []domain.Permission{domain.PermCatalogueWrite},
		AdminArea:   true,
	}))

	got, err := repo.Get(t.Context(), "photographer")
	require.NoError(t, err)
	assert.True(t, got.Has(domain.PermCatalogueWrite))
	assert.False(t, got.System)

	require.NoError(t, repo.Delete(t.Context(), "photographer"))

	_, err = repo.Get(t.Context(), "photographer")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Re-seeding is insert-if-missing: a built-in edited by an admin is not reset.
	edited := admin
	edited.Permissions = []domain.Permission{domain.PermOrdersRead}
	require.NoError(t, repo.Upsert(t.Context(), edited))
	require.NoError(t, repo.EnsureIndexes(t.Context()))

	after, err := repo.Get(t.Context(), "admin")
	require.NoError(t, err)
	assert.False(t, after.Has(domain.PermTeamWrite), "re-seed must not overwrite an edited built-in")
}
