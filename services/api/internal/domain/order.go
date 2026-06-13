package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// ErrDuplicateRef is returned when an order reference already exists.
var ErrDuplicateRef = errors.New("order reference already exists")

// ErrConflict is returned when a concurrent writer updated a document first;
// callers should reload and retry.
var ErrConflict = errors.New("concurrent update conflict")

// OrderType classifies how the customer engaged with the design page.
type OrderType string

// Order types match the three incoming order buckets in the admin inbox.
const (
	OrderTypeStandard     OrderType = "standard"
	OrderTypeCustomSize   OrderType = "custom_size"
	OrderTypeDesignChange OrderType = "design_change"
	OrderTypeVisit        OrderType = "visit"
)

// OrderStatus is the internal production state of an order.
type OrderStatus string

// Order lifecycle states. Booked is reached automatically when a payment
// succeeds; in_production is only valid once the order is paid.
const (
	OrderStatusPendingPayment  OrderStatus = "pending_payment"
	OrderStatusRequested       OrderStatus = "requested"
	OrderStatusQuoted          OrderStatus = "quoted"
	OrderStatusPaymentLinkSent OrderStatus = "payment_link_sent"
	OrderStatusBooked          OrderStatus = "booked"
	OrderStatusInProduction    OrderStatus = "in_production"
	OrderStatusReady           OrderStatus = "ready"
	OrderStatusFulfilled       OrderStatus = "fulfilled"
	OrderStatusCancelled       OrderStatus = "cancelled"
)

// DesignSnapshot freezes the design name, lead photo and price at order time
// so later catalog edits never alter an existing order.
type DesignSnapshot struct {
	Name          string
	PhotoPublicID string
	PricePesewas  int64
}

// Customisation captures the customer's sizing and design choices.
type Customisation struct {
	SizeMode     string            // band | self | home_visit | workplace
	BandLabel    string            // set when SizeMode == "band"
	Measurements map[string]string // set when SizeMode == "self"
	DesignChange string            // free-text change request
}

// Quote is the merchant-provided price/timeline for a custom request.
type Quote struct {
	PricePesewas int64
	Timeline     string
	Notes        string
}

// Delivery captures the fulfilment choice and any auto-added dispatch rate.
type Delivery struct {
	Mode        string // pickup | dispatch
	Area        string
	RatePesewas *int64 // nil means "arrange directly" (off-rate dispatch)
}

// Payment lifecycle states. A payment flagged amount_mismatch was confirmed by
// the provider for a different amount than the order expected and needs admin
// attention before the order can be considered paid.
const (
	PaymentStatusPending  = "pending"
	PaymentStatusSuccess  = "success"
	PaymentStatusFailed   = "failed"
	PaymentStatusMismatch = "amount_mismatch"
)

// Payment records one provider transaction against an order.
type Payment struct {
	ProviderRef   string
	AmountPesewas int64
	Status        string // pending | success | failed | amount_mismatch
	Method        string
	PaidAt        *time.Time
}

// StatusChange is an append-only audit entry for every status transition.
type StatusChange struct {
	Status OrderStatus
	At     time.Time
	By     string
}

// Order is a customer's request to make a garment.
type Order struct {
	ID             string
	Ref            string
	CustomerID     string
	DesignID       string
	DesignSnapshot DesignSnapshot
	Type           OrderType
	Customisation  Customisation
	Quote          Quote
	Delivery       Delivery
	Payments       []Payment
	Status         OrderStatus
	StatusHistory  []StatusChange
	CustomerPhone  string
	// Version is an optimistic-concurrency token bumped on every update so
	// concurrent read-modify-write cycles cannot silently overwrite each other.
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsPaid reports whether the order has a successful payment covering at least
// the charged amount. Standard orders are paid at the full garment+delivery
// total; custom requests are paid when a payment link or manual mark succeeds.
func (o *Order) IsPaid() bool {
	for _, p := range o.Payments {
		if p.Status == PaymentStatusSuccess {
			return true
		}
	}

	return false
}

// TotalPesewas returns the amount the customer was asked to pay: the garment
// price plus any delivery rate. Custom quotes replace the garment price once set.
func (o *Order) TotalPesewas() int64 {
	garment := o.DesignSnapshot.PricePesewas
	if o.Quote.PricePesewas > 0 {
		garment = o.Quote.PricePesewas
	}

	total := garment
	if o.Delivery.RatePesewas != nil {
		total += *o.Delivery.RatePesewas
	}

	return total
}

// allowedTransitions is the explicit state machine: each status maps to the
// only statuses it may move to. Production stages cannot be skipped, terminal
// states have no outgoing edges, and cancellation is allowed from every
// non-terminal state. Booked additionally requires the payment paths (webhook
// or manual mark-paid); the admin status endpoint rejects it outright.
func allowedTransitions(from OrderStatus) []OrderStatus {
	switch from {
	case OrderStatusPendingPayment:
		return []OrderStatus{OrderStatusBooked, OrderStatusCancelled}
	case OrderStatusRequested:
		return []OrderStatus{OrderStatusQuoted, OrderStatusCancelled}
	case OrderStatusQuoted:
		return []OrderStatus{OrderStatusPaymentLinkSent, OrderStatusBooked, OrderStatusCancelled}
	case OrderStatusPaymentLinkSent:
		return []OrderStatus{OrderStatusBooked, OrderStatusCancelled}
	case OrderStatusBooked:
		return []OrderStatus{OrderStatusInProduction, OrderStatusCancelled}
	case OrderStatusInProduction:
		return []OrderStatus{OrderStatusReady, OrderStatusCancelled}
	case OrderStatusReady:
		return []OrderStatus{OrderStatusFulfilled, OrderStatusCancelled}
	case OrderStatusFulfilled, OrderStatusCancelled:
		return nil
	default:
		return nil
	}
}

// KnownOrderStatus reports whether the value is a member of the order status
// enum. Handlers use it to reject arbitrary strings before they are persisted.
func KnownOrderStatus(status OrderStatus) bool {
	switch status {
	case OrderStatusPendingPayment,
		OrderStatusRequested,
		OrderStatusQuoted,
		OrderStatusPaymentLinkSent,
		OrderStatusBooked,
		OrderStatusInProduction,
		OrderStatusReady,
		OrderStatusFulfilled,
		OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransition reports whether the order is allowed to enter the target
// status: the edge must exist in the transition table, and in_production
// additionally requires a successful payment.
func (o *Order) CanTransition(to OrderStatus) bool {
	if to == OrderStatusInProduction && !o.IsPaid() {
		return false
	}

	return slices.Contains(allowedTransitions(o.Status), to)
}

// Transition moves the order to a new status if allowed and appends a history
// entry. It updates UpdatedAt and returns the previous status.
func (o *Order) Transition(to OrderStatus, by string, at time.Time) (OrderStatus, error) {
	if !o.CanTransition(to) {
		return o.Status, fmt.Errorf("%w: cannot move order from %s to %s", ErrInvalidInput, o.Status, to)
	}

	prev := o.Status
	o.Status = to
	o.UpdatedAt = at
	o.StatusHistory = append(o.StatusHistory, StatusChange{
		Status: to,
		At:     at,
		By:     by,
	})

	return prev, nil
}

// RecordInitialStatus seeds the audit trail with the order's current status.
// It is used at creation time, where Transition does not apply because the
// transition table has no self-edges.
func (o *Order) RecordInitialStatus(by string, at time.Time) {
	o.StatusHistory = append(o.StatusHistory, StatusChange{
		Status: o.Status,
		At:     at,
		By:     by,
	})
}

// MarkPaid records a successful payment attributed to by, marking the order
// booked if it was waiting for payment. An existing payment record with the
// same provider reference keeps its originally expected amount; only its
// status, method, and paid time change. It returns the previous status.
func (o *Order) MarkPaid(payment Payment, by string, at time.Time) (OrderStatus, error) {
	existing := o.paymentByProviderRef(payment.ProviderRef)
	if existing != nil {
		existing.Status = PaymentStatusSuccess
		existing.PaidAt = &at

		if payment.Method != "" {
			existing.Method = payment.Method
		}
	} else {
		payment.Status = PaymentStatusSuccess
		payment.PaidAt = &at
		o.Payments = append(o.Payments, payment)
	}

	prev := o.Status
	if o.Status == OrderStatusPendingPayment || o.Status == OrderStatusPaymentLinkSent {
		_, err := o.Transition(OrderStatusBooked, by, at)
		if err != nil {
			return prev, err
		}
	}

	o.UpdatedAt = at

	return prev, nil
}

func (o *Order) paymentByProviderRef(providerRef string) *Payment {
	for i := range o.Payments {
		if o.Payments[i].ProviderRef == providerRef {
			return &o.Payments[i]
		}
	}

	return nil
}

// OrderFilter narrows order listings.
type OrderFilter struct {
	Status OrderStatus
	Type   OrderType
	Limit  int64
}

// OrderRepository is the persistence port for orders.
type OrderRepository interface {
	Create(ctx context.Context, o *Order) error
	// Update persists the order only when its Version still matches the stored
	// document, returning ErrConflict when a concurrent writer won the race.
	Update(ctx context.Context, o *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	GetByRef(ctx context.Context, ref string) (*Order, error)
	ListByCustomer(ctx context.Context, customerID string) ([]Order, error)
	List(ctx context.Context, filter OrderFilter) ([]Order, error)
	// Count returns the number of orders matching the filter.
	Count(ctx context.Context, filter OrderFilter) (int64, error)
	// ListPaged returns one page of orders matching the filter, newest first.
	ListPaged(ctx context.Context, filter OrderFilter, params PageParams) ([]Order, error)
}
