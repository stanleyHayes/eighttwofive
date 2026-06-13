package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// CalendarSlot implements admin slot management use-cases.
type CalendarSlot struct {
	slots domain.SlotRepository
	now   func() time.Time
}

// NewCalendarSlot wires the slot service.
func NewCalendarSlot(slots domain.SlotRepository) *CalendarSlot {
	return &CalendarSlot{slots: slots, now: time.Now}
}

// CreateSlot opens a new time window for bookings.
func (s *CalendarSlot) CreateSlot(ctx context.Context, start, end time.Time) (*domain.Slot, error) {
	if end.Before(start) || end.Equal(start) {
		return nil, fmt.Errorf("%w: end must be after start", domain.ErrInvalidInput)
	}

	now := s.now().UTC()
	slot := &domain.Slot{
		Start:     start.UTC(),
		End:       end.UTC(),
		Status:    domain.SlotStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.slots.Create(ctx, slot)
	if err != nil {
		return nil, fmt.Errorf("create slot: %w", err)
	}

	return slot, nil
}

// ListSlots returns slots matching the filter, newest first.
func (s *CalendarSlot) ListSlots(ctx context.Context, filter domain.SlotFilter) ([]domain.Slot, error) {
	slots, err := s.slots.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list slots: %w", err)
	}

	return slots, nil
}

// CloseSlot removes an open slot from public availability without a booking.
func (s *CalendarSlot) CloseSlot(ctx context.Context, id string) error {
	err := s.setStatus(ctx, id, domain.SlotStatusClosed)
	if err != nil {
		return err
	}

	return nil
}

// ReopenSlot makes a closed slot open again. It is safe to call on
// already-open slots.
func (s *CalendarSlot) ReopenSlot(ctx context.Context, id string) error {
	err := s.setStatus(ctx, id, domain.SlotStatusOpen)
	if err != nil {
		return err
	}

	return nil
}

// setStatus changes a slot's status with a conditional write keyed on the
// status that was just read, so a concurrent booking between the read and the
// write surfaces as ErrSlotUnavailable instead of being overwritten.
func (s *CalendarSlot) setStatus(ctx context.Context, id string, status domain.SlotStatus) error {
	slot, err := s.slots.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrSlotNotFound
		}

		return fmt.Errorf("load slot: %w", err)
	}

	if slot.Status == domain.SlotStatusBooked {
		// A booked slot belongs to a visit: it may only change through the
		// visit reschedule/cancel operations, never directly.
		return fmt.Errorf("%w: cannot change a booked slot directly", domain.ErrInvalidInput)
	}

	err = s.slots.UpdateStatusFrom(ctx, id, slot.Status, status)
	if err != nil {
		return fmt.Errorf("update slot: %w", err)
	}

	return nil
}
