package mongostore_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func TestUserRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	repo := mongostore.NewUserRepository(setupDatabase(t))
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	// First upsert creates the user and backfills the ID.
	user := &domain.User{
		Email:     "ama@example.com",
		Name:      "Ama",
		Role:      domain.RoleCustomer,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, repo.Upsert(ctx, user))
	require.NotEmpty(t, user.ID)

	// A second upsert keeps the stored identity (name is not overwritten).
	again := &domain.User{
		Email: "ama@example.com", Name: "Different",
		Role: domain.RoleCustomer, CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Upsert(ctx, again))
	assert.Equal(t, user.ID, again.ID)
	assert.Equal(t, "Ama", again.Name)

	// An admin upsert promotes; it never demotes.
	promote := &domain.User{Email: "ama@example.com", Name: "Ama", Role: domain.RoleAdmin, CreatedAt: time.Now().UTC()}
	require.NoError(t, repo.Upsert(ctx, promote))
	assert.Equal(t, domain.RoleAdmin, promote.Role)

	demote := &domain.User{Email: "ama@example.com", Name: "Ama", Role: domain.RoleCustomer, CreatedAt: time.Now().UTC()}
	require.NoError(t, repo.Upsert(ctx, demote))
	assert.Equal(t, domain.RoleAdmin, demote.Role, "customer upsert must not demote an admin")

	// Lookups.
	loaded, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "ama@example.com", loaded.Email)

	_, err = repo.GetByID(ctx, "not-a-hex-id")
	require.ErrorIs(t, err, domain.ErrNotFound)

	_, err = repo.GetByID(ctx, "6a2c000000000000deadbeef")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestTokenRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	repo := mongostore.NewTokenRepository(setupDatabase(t))
	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	// Login tokens are single-use.
	require.NoError(t, repo.StoreLoginToken(ctx, "hash-1", "user-1", future))

	userID, err := repo.ConsumeLoginToken(ctx, "hash-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)

	_, err = repo.ConsumeLoginToken(ctx, "hash-1")
	require.ErrorIs(t, err, domain.ErrTokenInvalid, "second consume must fail")

	// Expired login tokens are rejected.
	require.NoError(t, repo.StoreLoginToken(ctx, "hash-expired", "user-1", past))

	_, err = repo.ConsumeLoginToken(ctx, "hash-expired")
	require.ErrorIs(t, err, domain.ErrTokenInvalid)

	// Unknown tokens are rejected.
	_, err = repo.ConsumeLoginToken(ctx, "hash-unknown")
	require.ErrorIs(t, err, domain.ErrTokenInvalid)

	// Sessions: create, read, expire, delete.
	require.NoError(t, repo.CreateSession(ctx, "sess-1", "user-2", future))

	userID, err = repo.GetSession(ctx, "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "user-2", userID)

	require.NoError(t, repo.CreateSession(ctx, "sess-expired", "user-2", past))

	_, err = repo.GetSession(ctx, "sess-expired")
	require.ErrorIs(t, err, domain.ErrTokenInvalid)

	require.NoError(t, repo.DeleteSession(ctx, "sess-1"))

	_, err = repo.GetSession(ctx, "sess-1")
	require.ErrorIs(t, err, domain.ErrTokenInvalid)

	require.NoError(t, repo.DeleteSession(ctx, "sess-missing"), "deleting a missing session is not an error")
}

func TestSettingsRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	repo := mongostore.NewSettingsRepository(setupDatabase(t))
	ctx := context.Background()

	// Defaults before anything is saved — GHS 500 deposit per scope §4.3.
	settings, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(50000), settings.DepositPesewas)

	// Round-trip.
	saved := &domain.Settings{
		DepositPesewas: 75000,
		WhatsAppNumber: "+233200000000",
		VisitLocation:  "Osu, Accra",
		DeliveryRates:  []domain.DeliveryRate{},
	}
	require.NoError(t, repo.Update(ctx, saved))

	loaded, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, saved, loaded)

	// Update is idempotent over the single document.
	saved.DepositPesewas = 80000
	require.NoError(t, repo.Update(ctx, saved))

	reloaded, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(80000), reloaded.DepositPesewas)
}
