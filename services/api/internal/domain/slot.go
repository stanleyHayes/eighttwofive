package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSlotUnavailable is returned when a slot cannot be booked because it is
	// already booked, closed, or otherwise not open.
	ErrSlotUnavailable = errors.New("slot unavailable")

	// ErrSlotNotFound is returned when a requested slot does not exist.
	ErrSlotNotFound = errors.New("slot not found")
)

// SlotStatus is the availability state of a calendar slot.
type SlotStatus string

// Slot lifecycle states.
const (
	SlotStatusOpen   SlotStatus = "open"
	SlotStatusBooked SlotStatus = "booked"
	SlotStatusClosed SlotStatus = "closed"
)

// Slot is a merchant-opened time window that customers can book for a home visit.
type Slot struct {
	ID        string
	Start     time.Time
	End       time.Time
	Status    SlotStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SlotFilter narrows slot listings.
type SlotFilter struct {
	Status SlotStatus
	After  time.Time
	Before time.Time
}

// SlotRepository is the persistence port for calendar slots.
type SlotRepository interface {
	Create(ctx context.Context, s *Slot) error
	GetByID(ctx context.Context, id string) (*Slot, error)
	List(ctx context.Context, filter SlotFilter) ([]Slot, error)
	// Overlaps reports whether any open or booked slot intersects the half-open
	// interval [start, end). Closed slots are ignored since they are not
	// bookable and cannot double-book the atelier.
	Overlaps(ctx context.Context, start, end time.Time) (bool, error)
	// UpdateStatusFrom atomically moves a slot from one status to another in a
	// single conditional write. It returns ErrSlotUnavailable when the slot is
	// not currently in the from status, and ErrNotFound when it does not exist,
	// so check-then-act races cannot stomp a concurrent booking.
	UpdateStatusFrom(ctx context.Context, id string, from, to SlotStatus) error
}
