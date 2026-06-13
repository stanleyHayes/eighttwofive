package mongostore_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func TestSlotRepository_CreateAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewSlotRepository(db)

	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)

	slot := &domain.Slot{
		Start:     start,
		End:       end,
		Status:    domain.SlotStatusOpen,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, slot))
	assert.NotEmpty(t, slot.ID)

	loaded, err := repo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, start, loaded.Start)
	assert.Equal(t, end, loaded.End)
	assert.Equal(t, domain.SlotStatusOpen, loaded.Status)

	list, err := repo.List(ctx, domain.SlotFilter{Status: domain.SlotStatusOpen})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, slot.ID, list[0].ID)
}

func TestSlotRepository_DuplicateTimeWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewSlotRepository(db)

	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	start := time.Now().UTC().Add(24 * time.Hour)
	end := start.Add(30 * time.Minute)

	first := &domain.Slot{
		Start:     start,
		End:       end,
		Status:    domain.SlotStatusOpen,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, first))

	second := &domain.Slot{
		Start:     start,
		End:       end,
		Status:    domain.SlotStatusOpen,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, second)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestSlotRepository_UpdateStatusFrom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewSlotRepository(db)

	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	now := time.Now().UTC()
	slot := &domain.Slot{
		Start:     now.Add(24 * time.Hour),
		End:       now.Add(25 * time.Hour),
		Status:    domain.SlotStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, slot))

	require.NoError(t, repo.UpdateStatusFrom(ctx, slot.ID, domain.SlotStatusOpen, domain.SlotStatusClosed))

	loaded, err := repo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusClosed, loaded.Status)

	// A stale precondition must not overwrite the current status.
	err = repo.UpdateStatusFrom(ctx, slot.ID, domain.SlotStatusOpen, domain.SlotStatusBooked)
	require.ErrorIs(t, err, domain.ErrSlotUnavailable)

	loaded, err = repo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusClosed, loaded.Status)

	// Missing slots are reported as not found, not as unavailable.
	err = repo.UpdateStatusFrom(ctx, "ffffffffffffffffffffffff", domain.SlotStatusOpen, domain.SlotStatusBooked)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSlotRepository_UpdateStatusFrom_ConcurrentClaim(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewSlotRepository(db)

	ctx := context.Background()
	require.NoError(t, repo.EnsureIndexes(ctx))

	now := time.Now().UTC()
	slot := &domain.Slot{
		Start:     now.Add(24 * time.Hour),
		End:       now.Add(25 * time.Hour),
		Status:    domain.SlotStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, slot))

	// Two concurrent claims race for the same open slot: exactly one wins.
	results := make(chan error, 2)

	var waitGroup sync.WaitGroup

	for range 2 {
		waitGroup.Go(func() {
			results <- repo.UpdateStatusFrom(ctx, slot.ID, domain.SlotStatusOpen, domain.SlotStatusBooked)
		})
	}

	waitGroup.Wait()
	close(results)

	wins := 0
	losses := 0

	for err := range results {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, domain.ErrSlotUnavailable):
			losses++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	assert.Equal(t, 1, wins, "exactly one concurrent claim must win")
	assert.Equal(t, 1, losses)

	loaded, err := repo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusBooked, loaded.Status)
}
