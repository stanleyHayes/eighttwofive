package mongostore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func setupRepo(t *testing.T) *mongostore.SubscriberRepository {
	t.Helper()

	repo := mongostore.NewSubscriberRepository(setupDatabase(t))
	require.NoError(t, repo.EnsureIndexes(context.Background()))

	return repo
}

// The assertions run as one sequential flow (create -> duplicate -> list)
// because they share container state; parallelism is at the test level.
func TestSubscriberRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	repo := setupRepo(t)
	ctx := context.Background()

	// Create assigns an ID.
	first := &domain.Subscriber{
		Email:     "ada@example.com",
		Name:      "Ada",
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, repo.Create(ctx, first))
	assert.NotEmpty(t, first.ID)

	// Duplicate email is rejected with the domain error.
	dup := &domain.Subscriber{
		Email:     "ada@example.com",
		Name:      "Imposter",
		CreatedAt: time.Now().UTC(),
	}
	require.ErrorIs(t, repo.Create(ctx, dup), domain.ErrDuplicateEmail)

	// List returns newest first.
	older := &domain.Subscriber{
		Email:     "grace@example.com",
		Name:      "Grace",
		CreatedAt: time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond),
	}
	require.NoError(t, repo.Create(ctx, older))

	subs, err := repo.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, subs, 2)
	assert.Equal(t, "ada@example.com", subs[0].Email)
	assert.Equal(t, "grace@example.com", subs[1].Email)
	assert.True(t, subs[0].CreatedAt.After(subs[1].CreatedAt))

	// List respects the limit.
	limited, err := repo.List(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, limited, 1)
}

func TestSubscriberRepository_CountAndListPaged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	repo := setupRepo(t)
	ctx := context.Background()

	const seeded = 25

	base := time.Now().UTC().Truncate(time.Millisecond)
	for i := range seeded {
		require.NoError(t, repo.Create(ctx, &domain.Subscriber{
			Email: fmt.Sprintf("sub%02d@example.com", i),
			Name:  fmt.Sprintf("Sub %02d", i),
			// Newer index => more recent createdAt, so newest-first is deterministic.
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}))
	}

	total, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(seeded), total)

	// First page: full page, newest first.
	first, err := repo.ListPaged(ctx, domain.NormalizePageParams(1, 10))
	require.NoError(t, err)
	require.Len(t, first, 10)
	assert.Equal(t, "sub24@example.com", first[0].Email)

	// Last page: the remainder.
	last, err := repo.ListPaged(ctx, domain.NormalizePageParams(3, 10))
	require.NoError(t, err)
	assert.Len(t, last, seeded-20)

	// Past the end: empty, never an error.
	beyond, err := repo.ListPaged(ctx, domain.NormalizePageParams(99, 10))
	require.NoError(t, err)
	assert.Empty(t, beyond)
}
