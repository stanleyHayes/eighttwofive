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

	// Re-seeding preserves an edit to a NON-admin built-in role.
	staff, err := repo.Get(t.Context(), "staff")
	require.NoError(t, err)

	staff.Permissions = []domain.Permission{domain.PermOrdersRead}
	require.NoError(t, repo.Upsert(t.Context(), staff))

	// The admin role, by contrast, is always re-synced to the full permission set
	// on re-seed (it is the recovery path and must never lose access).
	demoted := *admin
	demoted.Permissions = []domain.Permission{domain.PermOrdersRead}
	demoted.AdminArea = false
	require.NoError(t, repo.Upsert(t.Context(), &demoted))

	require.NoError(t, repo.EnsureIndexes(t.Context()))

	afterStaff, err := repo.Get(t.Context(), "staff")
	require.NoError(t, err)
	assert.False(t, afterStaff.Has(domain.PermCatalogueWrite), "an edited non-admin built-in survives re-seed")

	afterAdmin, err := repo.Get(t.Context(), "admin")
	require.NoError(t, err)
	assert.True(t, afterAdmin.Has(domain.PermTeamWrite), "admin is re-synced to full on re-seed")
	assert.True(t, afterAdmin.Has(domain.PermSubscribersWrite), "admin gains newly added capabilities on re-seed")
	assert.True(t, afterAdmin.AdminArea, "admin keeps dashboard access on re-seed")
}
