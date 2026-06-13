package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrVisitNotFound is returned when a requested visit does not exist.
	ErrVisitNotFound = errors.New("visit not found")

	// ErrVisitAlreadyCancelled is returned when an operation is attempted on a
	// visit that has already been cancelled.
	ErrVisitAlreadyCancelled = errors.New("visit already cancelled")
)

// VisitStatus is the lifecycle state of a home-visit booking.
type VisitStatus string

// Visit lifecycle states.
const (
	VisitStatusBooked    VisitStatus = "booked"
	VisitStatusDone      VisitStatus = "done"
	VisitStatusCancelled VisitStatus = "cancelled"
)

// Visit links an order to a booked slot. The deposit payment on the order
// confirms the visit.
type Visit struct {
	ID               string
	OrderID          string
	SlotID           string
	DepositPaymentID string
	Status           VisitStatus
	// HoldExpiresAt bounds an unpaid booking: until the deposit is confirmed
	// the slot is only soft-held, and once the hold lapses the visit is
	// cancelled and the slot reopened. Nil means the booking is firm.
	HoldExpiresAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// VisitFilter narrows visit listings.
type VisitFilter struct {
	Status VisitStatus
	SlotID string
}

// VisitRepository is the persistence port for home-visit bookings.
type VisitRepository interface {
	// Create inserts a visit. It fills the visit ID on success.
	Create(ctx context.Context, v *Visit) error
	// BookSlot claims an open slot atomically and inserts the visit. It fills
	// the visit ID and returns ErrSlotUnavailable if the slot is not open.
	BookSlot(ctx context.Context, slotID string, v *Visit) error
	GetByID(ctx context.Context, id string) (*Visit, error)
	GetByOrderID(ctx context.Context, orderID string) (*Visit, error)
	List(ctx context.Context, filter VisitFilter) ([]Visit, error)
	// ListExpiredHolds returns booked visits whose deposit hold lapsed before now.
	ListExpiredHolds(ctx context.Context, now time.Time) ([]Visit, error)
	Update(ctx context.Context, v *Visit) error
}
