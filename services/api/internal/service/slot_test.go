package service_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type fakeSlotRepo struct {
	byID   map[string]*domain.Slot
	nextID int
}

func newFakeSlotRepo() *fakeSlotRepo {
	return &fakeSlotRepo{byID: map[string]*domain.Slot{}, nextID: 1}
}

func (f *fakeSlotRepo) Create(_ context.Context, slot *domain.Slot) error {
	slot.ID = "slot-" + strconv.Itoa(f.nextID)
	f.nextID++
	clone := *slot
	f.byID[slot.ID] = &clone

	return nil
}

func (f *fakeSlotRepo) GetByID(_ context.Context, id string) (*domain.Slot, error) {
	if slot, ok := f.byID[id]; ok {
		clone := *slot

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (f *fakeSlotRepo) List(_ context.Context, filter domain.SlotFilter) ([]domain.Slot, error) {
	out := make([]domain.Slot, 0, len(f.byID))
	for _, slot := range f.byID {
		if filter.Status != "" && slot.Status != filter.Status {
			continue
		}

		clone := *slot
		out = append(out, clone)
	}

	return out, nil
}

func (f *fakeSlotRepo) Overlaps(_ context.Context, start, end time.Time) (bool, error) {
	for _, slot := range f.byID {
		if slot.Status == domain.SlotStatusClosed {
			continue
		}

		if slot.Start.Before(end) && slot.End.After(start) {
			return true, nil
		}
	}

	return false, nil
}

func (f *fakeSlotRepo) UpdateStatusFrom(_ context.Context, id string, from, to domain.SlotStatus) error {
	slot, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}

	if slot.Status != from {
		return domain.ErrSlotUnavailable
	}

	slot.Status = to

	return nil
}

func TestCalendarSlot_CreateSlot(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	svc := service.NewCalendarSlot(newFakeSlotRepo())
	start := time.Now().UTC().Add(24 * time.Hour)
	end := start.Add(30 * time.Minute)

	slot, err := svc.CreateSlot(ctx, start, end)
	require.NoError(t, err)

	assert.Equal(t, domain.SlotStatusOpen, slot.Status)
	assert.Equal(t, start.Truncate(time.Second), slot.Start.Truncate(time.Second))
	assert.Equal(t, end.Truncate(time.Second), slot.End.Truncate(time.Second))
}

func TestCalendarSlot_CreateSlot_EndBeforeStart(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	svc := service.NewCalendarSlot(newFakeSlotRepo())
	now := time.Now().UTC()

	_, err := svc.CreateSlot(ctx, now, now.Add(-time.Hour))
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCalendarSlot_CreateSlot_RejectsOverlap(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	svc := service.NewCalendarSlot(newFakeSlotRepo())
	start := time.Now().UTC().Add(24 * time.Hour)

	_, err := svc.CreateSlot(ctx, start, start.Add(time.Hour))
	require.NoError(t, err)

	// A second window straddling the first must be rejected.
	_, err = svc.CreateSlot(ctx, start.Add(30*time.Minute), start.Add(90*time.Minute))
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	// A back-to-back window that merely touches the boundary is fine.
	_, err = svc.CreateSlot(ctx, start.Add(time.Hour), start.Add(2*time.Hour))
	require.NoError(t, err)
}

func TestCalendarSlot_ListSlots(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	repo := newFakeSlotRepo()
	svc := service.NewCalendarSlot(repo)

	openSlot := &domain.Slot{
		Status:    domain.SlotStatusOpen,
		Start:     time.Now().UTC(),
		End:       time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	closedSlot := &domain.Slot{
		Status:    domain.SlotStatusClosed,
		Start:     time.Now().UTC().Add(2 * time.Hour),
		End:       time.Now().UTC().Add(3 * time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, repo.Create(ctx, openSlot))
	require.NoError(t, repo.Create(ctx, closedSlot))

	all, err := svc.ListSlots(ctx, domain.SlotFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 2)

	openOnly, err := svc.ListSlots(ctx, domain.SlotFilter{Status: domain.SlotStatusOpen})
	require.NoError(t, err)
	assert.Len(t, openOnly, 1)
	assert.Equal(t, domain.SlotStatusOpen, openOnly[0].Status)
}

func TestCalendarSlot_CloseAndReopenSlot(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	repo := newFakeSlotRepo()
	svc := service.NewCalendarSlot(repo)

	slot := &domain.Slot{
		Status:    domain.SlotStatusOpen,
		Start:     time.Now().UTC(),
		End:       time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, slot))

	err := svc.CloseSlot(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusClosed, repo.byID[slot.ID].Status)

	err = svc.ReopenSlot(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SlotStatusOpen, repo.byID[slot.ID].Status)
}

func TestCalendarSlot_ReopenSlot_NotFound(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	svc := service.NewCalendarSlot(newFakeSlotRepo())

	err := svc.ReopenSlot(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrSlotNotFound)
}

func TestCalendarSlot_ReopenBookedSlot_Rejected(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	repo := newFakeSlotRepo()
	svc := service.NewCalendarSlot(repo)

	slot := &domain.Slot{
		Status:    domain.SlotStatusBooked,
		Start:     time.Now().UTC(),
		End:       time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, slot))

	err := svc.ReopenSlot(ctx, slot.ID)
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}
