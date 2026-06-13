package mongostore_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func newTestVisitSlot(ctx context.Context, t *testing.T, slotRepo *mongostore.SlotRepository) *domain.Slot {
	t.Helper()

	slot := &domain.Slot{
		Start:     time.Now().UTC().Add(24 * time.Hour),
		End:       time.Now().UTC().Add(25 * time.Hour),
		Status:    domain.SlotStatusOpen,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, slotRepo.Create(ctx, slot))

	return slot
}

func newTestVisit(orderID, slotID, paymentID string) *domain.Visit {
	now := time.Now().UTC()

	return &domain.Visit{
		OrderID:          orderID,
		SlotID:           slotID,
		DepositPaymentID: paymentID,
		Status:           domain.VisitStatusBooked,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func TestVisitRepository_BookSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	ctx := context.Background()
	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	slot := newTestVisitSlot(ctx, t, slotRepo)
	visit := newTestVisit("E25-VISIT-123", slot.ID, "ps-1")

	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, visit))
	assert.NotEmpty(t, visit.ID)

	loadedSlot, err := slotRepo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusBooked, loadedSlot.Status)

	loadedVisit, err := visitRepo.GetByOrderID(ctx, visit.OrderID)
	require.NoError(t, err)
	assert.Equal(t, visit.ID, loadedVisit.ID)
}

func TestVisitRepository_BookSlot_DoubleBooking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	ctx := context.Background()
	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	slot := newTestVisitSlot(ctx, t, slotRepo)
	first := newTestVisit("E25-VISIT-1", slot.ID, "ps-1")

	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, first))

	second := newTestVisit("E25-VISIT-2", slot.ID, "ps-2")
	err := visitRepo.BookSlot(ctx, slot.ID, second)
	require.ErrorIs(t, err, domain.ErrSlotUnavailable)
}

func TestVisitRepository_BookSlot_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	ctx := context.Background()
	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	slot := newTestVisitSlot(ctx, t, slotRepo)

	const goroutines = 10

	var waitGroup sync.WaitGroup

	successes := make(chan bool, goroutines)

	for index := range goroutines {
		waitGroup.Add(1)

		go func(index int) {
			defer waitGroup.Done()

			visit := newTestVisit(
				"E25-VISIT-"+string(rune('A'+index)),
				slot.ID,
				"ps-"+string(rune('A'+index)),
			)

			err := visitRepo.BookSlot(ctx, slot.ID, visit)
			successes <- err == nil
		}(index)
	}

	waitGroup.Wait()
	close(successes)

	successCount := 0

	for ok := range successes {
		if ok {
			successCount++
		}
	}

	assert.Equal(t, 1, successCount, "only one concurrent booking should succeed")

	loadedSlot, err := slotRepo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusBooked, loadedSlot.Status)
}

func TestVisitRepository_EnsureIndexes_ReplacesLegacySlotIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	ctx := context.Background()

	// Simulate a database created before the partial index existed.
	_, err := db.Collection("visits").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slotId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	require.NoError(t, err)

	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	// After migration a cancelled visit must no longer block the slot.
	slot := newTestVisitSlot(ctx, t, slotRepo)
	first := newTestVisit("E25-VISIT-1", slot.ID, "ps-1")
	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, first))

	first.Status = domain.VisitStatusCancelled
	first.UpdatedAt = time.Now().UTC()
	require.NoError(t, visitRepo.Update(ctx, first))
	require.NoError(t, slotRepo.UpdateStatusFrom(ctx, slot.ID, domain.SlotStatusBooked, domain.SlotStatusOpen))

	second := newTestVisit("E25-VISIT-2", slot.ID, "ps-2")
	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, second))
}

func TestVisitRepository_CancelledVisit_SlotRebookable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	ctx := context.Background()
	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	slot := newTestVisitSlot(ctx, t, slotRepo)
	first := newTestVisit("E25-VISIT-1", slot.ID, "ps-1")

	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, first))

	// Cancel the visit and reopen the slot, mirroring the cancel flow.
	first.Status = domain.VisitStatusCancelled
	first.UpdatedAt = time.Now().UTC()
	require.NoError(t, visitRepo.Update(ctx, first))
	require.NoError(t, slotRepo.UpdateStatusFrom(ctx, slot.ID, domain.SlotStatusBooked, domain.SlotStatusOpen))

	// The cancelled visit must not occupy the unique slot index: a fresh
	// booking on the same slot has to succeed.
	second := newTestVisit("E25-VISIT-2", slot.ID, "ps-2")
	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, second))
	assert.NotEmpty(t, second.ID)

	loadedSlot, err := slotRepo.GetByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusBooked, loadedSlot.Status)
}

func TestVisitRepository_ListExpiredHolds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	ctx := context.Background()
	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	now := time.Now().UTC()

	expiredSlot := newTestVisitSlot(ctx, t, slotRepo)
	expired := newTestVisit("E25-VISIT-EXPIRED", expiredSlot.ID, "ps-1")
	past := now.Add(-time.Minute)
	expired.HoldExpiresAt = &past
	require.NoError(t, visitRepo.BookSlot(ctx, expiredSlot.ID, expired))

	activeSlot := &domain.Slot{
		Start:     now.Add(48 * time.Hour),
		End:       now.Add(49 * time.Hour),
		Status:    domain.SlotStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, slotRepo.Create(ctx, activeSlot))

	active := newTestVisit("E25-VISIT-ACTIVE", activeSlot.ID, "ps-2")
	future := now.Add(time.Hour)
	active.HoldExpiresAt = &future
	require.NoError(t, visitRepo.BookSlot(ctx, activeSlot.ID, active))

	holds, err := visitRepo.ListExpiredHolds(ctx, now)
	require.NoError(t, err)
	require.Len(t, holds, 1)
	assert.Equal(t, "E25-VISIT-EXPIRED", holds[0].OrderID)
	require.NotNil(t, holds[0].HoldExpiresAt)
}

func TestVisitRepository_ListAndUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	slotRepo := mongostore.NewSlotRepository(db)
	visitRepo := mongostore.NewVisitRepository(db)

	ctx := context.Background()
	require.NoError(t, slotRepo.EnsureIndexes(ctx))
	require.NoError(t, visitRepo.EnsureIndexes(ctx))

	slot := newTestVisitSlot(ctx, t, slotRepo)
	visit := newTestVisit("E25-VISIT-1", slot.ID, "ps-1")

	require.NoError(t, visitRepo.BookSlot(ctx, slot.ID, visit))

	visit.Status = domain.VisitStatusCancelled
	visit.UpdatedAt = time.Now().UTC()
	require.NoError(t, visitRepo.Update(ctx, visit))

	list, err := visitRepo.List(ctx, domain.VisitFilter{Status: domain.VisitStatusCancelled})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, domain.VisitStatusCancelled, list[0].Status)
}
