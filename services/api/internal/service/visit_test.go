package service_test

import (
	"context"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type fakeVisitRepo struct {
	byID     map[string]*domain.Visit
	bySlotID map[string]*domain.Visit
	nextID   int
	// slots mirrors the real repository's atomic slot claim inside BookSlot;
	// nil keeps the claim out of tests that never reach it.
	slots *fakeSlotRepo
}

func newFakeVisitRepo() *fakeVisitRepo {
	return &fakeVisitRepo{
		byID:     map[string]*domain.Visit{},
		bySlotID: map[string]*domain.Visit{},
		nextID:   1,
		slots:    nil,
	}
}

func (f *fakeVisitRepo) Create(_ context.Context, visit *domain.Visit) error {
	visit.ID = "visit-" + strconv.Itoa(f.nextID)
	f.nextID++
	clone := *visit
	f.byID[visit.ID] = &clone
	f.bySlotID[visit.SlotID] = &clone

	return nil
}

func (f *fakeVisitRepo) BookSlot(ctx context.Context, slotID string, visit *domain.Visit) error {
	if existing, ok := f.bySlotID[slotID]; ok && existing.Status != domain.VisitStatusCancelled {
		return domain.ErrSlotUnavailable
	}

	if f.slots != nil {
		err := f.slots.UpdateStatusFrom(ctx, slotID, domain.SlotStatusOpen, domain.SlotStatusBooked)
		if err != nil {
			return domain.ErrSlotUnavailable
		}
	}

	return f.Create(ctx, visit)
}

func (f *fakeVisitRepo) GetByID(_ context.Context, id string) (*domain.Visit, error) {
	if visit, ok := f.byID[id]; ok {
		clone := *visit

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (f *fakeVisitRepo) GetByOrderID(_ context.Context, orderID string) (*domain.Visit, error) {
	for _, visit := range f.byID {
		if visit.OrderID == orderID {
			clone := *visit

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (f *fakeVisitRepo) List(_ context.Context, _ domain.VisitFilter) ([]domain.Visit, error) {
	out := make([]domain.Visit, 0, len(f.byID))
	for _, visit := range f.byID {
		clone := *visit
		out = append(out, clone)
	}

	return out, nil
}

func (f *fakeVisitRepo) ListExpiredHolds(_ context.Context, now time.Time) ([]domain.Visit, error) {
	out := make([]domain.Visit, 0, len(f.byID))

	for _, visit := range f.byID {
		if visit.Status != domain.VisitStatusBooked || visit.HoldExpiresAt == nil {
			continue
		}

		if visit.HoldExpiresAt.After(now) {
			continue
		}

		clone := *visit
		out = append(out, clone)
	}

	return out, nil
}

func (f *fakeVisitRepo) Update(_ context.Context, visit *domain.Visit) error {
	if _, ok := f.byID[visit.ID]; !ok {
		return domain.ErrNotFound
	}

	clone := *visit
	f.byID[visit.ID] = &clone
	f.bySlotID[visit.SlotID] = &clone

	return nil
}

func newVisitService(
	slots *fakeSlotRepo,
	visits *fakeVisitRepo,
	orders *fakeOrderRepo,
	designs *fakeDesignRepo,
	users *fakeUsers,
	payments *fakePaymentProvider,
	sender *recordingOrderSender,
	settings *fakeSettingsRepo,
) *service.CalendarVisit {
	logger := slog.New(slog.DiscardHandler)

	return service.NewCalendarVisit(
		slots, visits, orders, designs, users, payments, settings, sender, "https://shop.test", logger,
	)
}

type visitTestFixtures struct {
	slots    *fakeSlotRepo
	visits   *fakeVisitRepo
	orders   *fakeOrderRepo
	designs  *fakeDesignRepo
	users    *fakeUsers
	payments *fakePaymentProvider
	sender   *recordingOrderSender
	settings *fakeSettingsRepo
}

func newVisitTestFixtures() *visitTestFixtures {
	fixtures := &visitTestFixtures{
		slots:    newFakeSlotRepo(),
		visits:   newFakeVisitRepo(),
		orders:   newFakeOrderRepo(),
		designs:  newFakeDesignRepo(),
		users:    newFakeUsers(),
		payments: newFakePaymentProvider(),
		sender:   &recordingOrderSender{},
		settings: &fakeSettingsRepo{settings: &domain.Settings{DepositPesewas: 50000}},
	}

	fixtures.visits.slots = fixtures.slots

	return fixtures
}

func (f *visitTestFixtures) service() *service.CalendarVisit {
	return newVisitService(f.slots, f.visits, f.orders, f.designs, f.users, f.payments, f.sender, f.settings)
}

func makeSlot(status domain.SlotStatus, offset time.Duration) *domain.Slot {
	start := time.Now().UTC().Add(offset)

	return &domain.Slot{
		Status:    status,
		Start:     start,
		End:       start.Add(time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func TestCalendarVisit_BookSlot(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()
	fixtures.designs.byID["des-1"] = liveDesign()

	svc := fixtures.service()

	slot := makeSlot(domain.SlotStatusOpen, 24*time.Hour)
	require.NoError(t, fixtures.slots.Create(ctx, slot))

	result, err := svc.BookSlot(ctx, slot.ID, "des-1", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	assert.Equal(t, domain.VisitStatusBooked, result.Visit.Status)
	require.NotNil(t, result.Visit.HoldExpiresAt, "unpaid bookings must carry a hold expiry")
	assert.Equal(t, slot.ID, result.Visit.SlotID)
	assert.Equal(t, result.Order.Ref, result.Visit.OrderID)
	assert.Equal(t, domain.OrderTypeVisit, result.Order.Type)
	assert.Equal(t, domain.OrderStatusPendingPayment, result.Order.Status)
	assert.Equal(t, int64(50000), result.Order.TotalPesewas())
	assert.Equal(t, "https://checkout.test/pay", result.PaymentURL)
	assert.Equal(t, "ama@example.com", fixtures.sender.lastTo)
}

func TestCalendarVisit_BookSlot_SlotUnavailable(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	closedSlot := makeSlot(domain.SlotStatusClosed, 24*time.Hour)
	require.NoError(t, fixtures.slots.Create(ctx, closedSlot))

	_, err := svc.BookSlot(ctx, closedSlot.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.ErrorIs(t, err, domain.ErrSlotUnavailable)
}

func TestCalendarVisit_BookSlot_DoubleBooking(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	slot := makeSlot(domain.SlotStatusOpen, 24*time.Hour)
	require.NoError(t, fixtures.slots.Create(ctx, slot))

	_, err := svc.BookSlot(ctx, slot.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	_, err = svc.BookSlot(ctx, slot.ID, "", "kofi@example.com", "Kofi", "+233200000001")
	require.ErrorIs(t, err, domain.ErrSlotUnavailable)
}

func TestCalendarVisit_BookSlot_UnknownDesign(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	slot := makeSlot(domain.SlotStatusOpen, time.Hour)
	require.NoError(t, fixtures.slots.Create(ctx, slot))

	_, err := svc.BookSlot(ctx, slot.ID, "missing", "ama@example.com", "Ama", "+233200000000")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCalendarVisit_BookSlot_Validation(t *testing.T) {
	t.Parallel()

	svc := newVisitService(
		newFakeSlotRepo(), newFakeVisitRepo(), newFakeOrderRepo(), newFakeDesignRepo(),
		newFakeUsers(), newFakePaymentProvider(), &recordingOrderSender{}, &fakeSettingsRepo{},
	)

	cases := []struct {
		name string
		call func() (*service.BookSlotResult, error)
	}{
		{"missing email", func() (*service.BookSlotResult, error) {
			return svc.BookSlot(t.Context(), "slot-1", "", "", "Ama", "+233200000000")
		}},
		{"missing phone", func() (*service.BookSlotResult, error) {
			return svc.BookSlot(t.Context(), "slot-1", "", "ama@example.com", "Ama", "")
		}},
		{"missing name", func() (*service.BookSlotResult, error) {
			return svc.BookSlot(t.Context(), "slot-1", "", "ama@example.com", "", "+233200000000")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.call()
			require.ErrorIs(t, err, domain.ErrInvalidInput)
		})
	}
}

func TestCalendarVisit_RescheduleVisit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	oldSlot := makeSlot(domain.SlotStatusOpen, 24*time.Hour)
	newSlot := makeSlot(domain.SlotStatusOpen, 48*time.Hour)

	require.NoError(t, fixtures.slots.Create(ctx, oldSlot))
	require.NoError(t, fixtures.slots.Create(ctx, newSlot))

	booked, err := svc.BookSlot(ctx, oldSlot.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	rescheduled, err := svc.RescheduleVisit(ctx, booked.Visit.ID, newSlot.ID)
	require.NoError(t, err)

	assert.Equal(t, newSlot.ID, rescheduled.SlotID)
	assert.Equal(t, domain.SlotStatusOpen, fixtures.slots.byID[oldSlot.ID].Status)
	assert.Equal(t, domain.SlotStatusBooked, fixtures.slots.byID[newSlot.ID].Status)
}

func TestCalendarVisit_CancelVisit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	slot := makeSlot(domain.SlotStatusOpen, 24*time.Hour)

	require.NoError(t, fixtures.slots.Create(ctx, slot))

	booked, err := svc.BookSlot(ctx, slot.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	cancelled, err := svc.CancelVisit(ctx, booked.Visit.ID)
	require.NoError(t, err)

	assert.Equal(t, domain.VisitStatusCancelled, cancelled.Status)
	assert.Equal(t, domain.SlotStatusOpen, fixtures.slots.byID[slot.ID].Status)
}

func TestCalendarVisit_CancelVisit_AlreadyCancelled(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	slot := makeSlot(domain.SlotStatusOpen, time.Hour)

	require.NoError(t, fixtures.slots.Create(ctx, slot))

	booked, err := svc.BookSlot(ctx, slot.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	_, err = svc.CancelVisit(ctx, booked.Visit.ID)
	require.NoError(t, err)

	_, err = svc.CancelVisit(ctx, booked.Visit.ID)
	require.ErrorIs(t, err, domain.ErrVisitAlreadyCancelled)
}

func TestCalendarVisit_Reschedule_NewSlotAlreadyBooked(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	oldSlot := makeSlot(domain.SlotStatusOpen, 24*time.Hour)
	takenSlot := makeSlot(domain.SlotStatusBooked, 48*time.Hour)

	require.NoError(t, fixtures.slots.Create(ctx, oldSlot))
	require.NoError(t, fixtures.slots.Create(ctx, takenSlot))

	booked, err := svc.BookSlot(ctx, oldSlot.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	// The conditional claim must refuse the taken slot without touching it or
	// the visit, so nobody's booking gets stomped.
	_, err = svc.RescheduleVisit(ctx, booked.Visit.ID, takenSlot.ID)
	require.ErrorIs(t, err, domain.ErrSlotUnavailable)

	assert.Equal(t, domain.SlotStatusBooked, fixtures.slots.byID[takenSlot.ID].Status)
	assert.Equal(t, domain.SlotStatusBooked, fixtures.slots.byID[oldSlot.ID].Status)
	assert.Equal(t, oldSlot.ID, fixtures.visits.byID[booked.Visit.ID].SlotID)
}

func TestCalendarVisit_ExpiredUnpaidHold_ReleasesSlot(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	slotA := makeSlot(domain.SlotStatusOpen, 24*time.Hour)
	slotB := makeSlot(domain.SlotStatusOpen, 48*time.Hour)

	require.NoError(t, fixtures.slots.Create(ctx, slotA))
	require.NoError(t, fixtures.slots.Create(ctx, slotB))

	first, err := svc.BookSlot(ctx, slotA.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	// The deposit never arrives and the hold lapses.
	past := time.Now().UTC().Add(-time.Minute)
	fixtures.visits.byID[first.Visit.ID].HoldExpiresAt = &past

	// The next booking attempt lazily releases the lapsed hold.
	_, err = svc.BookSlot(ctx, slotB.ID, "", "kofi@example.com", "Kofi", "+233200000001")
	require.NoError(t, err)

	assert.Equal(t, domain.VisitStatusCancelled, fixtures.visits.byID[first.Visit.ID].Status)
	assert.Equal(t, domain.SlotStatusOpen, fixtures.slots.byID[slotA.ID].Status)

	// The released slot is bookable again.
	_, err = svc.BookSlot(ctx, slotA.ID, "", "esi@example.com", "Esi", "+233200000002")
	require.NoError(t, err)
}

func TestCalendarVisit_ExpiredPaidHold_BecomesFirmBooking(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixtures := newVisitTestFixtures()

	svc := fixtures.service()

	slotA := makeSlot(domain.SlotStatusOpen, 24*time.Hour)
	slotB := makeSlot(domain.SlotStatusOpen, 48*time.Hour)

	require.NoError(t, fixtures.slots.Create(ctx, slotA))
	require.NoError(t, fixtures.slots.Create(ctx, slotB))

	first, err := svc.BookSlot(ctx, slotA.ID, "", "ama@example.com", "Ama", "+233200000000")
	require.NoError(t, err)

	// The deposit webhook confirmed payment before the hold lapsed.
	paid := fixtures.orders.byRef[first.Order.Ref]
	_, err = paid.MarkPaid(domain.Payment{
		ProviderRef:   first.Order.Ref,
		AmountPesewas: 50000,
	}, "payment_webhook", time.Now().UTC())
	require.NoError(t, err)

	past := time.Now().UTC().Add(-time.Minute)
	fixtures.visits.byID[first.Visit.ID].HoldExpiresAt = &past

	_, err = svc.BookSlot(ctx, slotB.ID, "", "kofi@example.com", "Kofi", "+233200000001")
	require.NoError(t, err)

	promoted := fixtures.visits.byID[first.Visit.ID]
	assert.Equal(t, domain.VisitStatusBooked, promoted.Status)
	assert.Nil(t, promoted.HoldExpiresAt, "paid holds must promote to firm bookings")
	assert.Equal(t, domain.SlotStatusBooked, fixtures.slots.byID[slotA.ID].Status)
}
