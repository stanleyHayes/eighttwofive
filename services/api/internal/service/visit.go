package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	maxVisitNameLength  = 120
	maxVisitPhoneLength = 30
	refVisitPrefix      = "E25-VISIT-"
	refVisitBytes       = 6
	maxRefVisitRetries  = 5
	// slotHoldDuration bounds how long an unpaid booking may hold a slot:
	// when the deposit is not confirmed in time the hold is released and the
	// slot reopens, so unpaid bookings cannot exhaust the calendar.
	slotHoldDuration = 45 * time.Minute
)

// BookSlotResult is the output of a successful home-visit booking.
type BookSlotResult struct {
	Visit      *domain.Visit
	Order      *domain.Order
	PaymentURL string
	User       *domain.User
}

// CalendarVisit implements visit booking and management use-cases.
type CalendarVisit struct {
	slots    domain.SlotRepository
	visits   domain.VisitRepository
	orders   domain.OrderRepository
	designs  domain.DesignRepository
	users    domain.UserRepository
	payments domain.PaymentProvider
	settings domain.SettingsRepository
	email    domain.EmailSender
	webURL   string
	logger   *slog.Logger
	now      func() time.Time
}

// NewCalendarVisit wires the visit service.
func NewCalendarVisit(
	slots domain.SlotRepository,
	visits domain.VisitRepository,
	orders domain.OrderRepository,
	designs domain.DesignRepository,
	users domain.UserRepository,
	payments domain.PaymentProvider,
	settings domain.SettingsRepository,
	email domain.EmailSender,
	webURL string,
	logger *slog.Logger,
) *CalendarVisit {
	return &CalendarVisit{
		slots:    slots,
		visits:   visits,
		orders:   orders,
		designs:  designs,
		users:    users,
		payments: payments,
		settings: settings,
		email:    email,
		webURL:   strings.TrimRight(webURL, "/"),
		logger:   logger,
		now:      time.Now,
	}
}

// BookSlot claims a slot, creates a visit order, and initializes deposit payment.
// The designID may be empty when the booking does not originate from a design page.
func (s *CalendarVisit) BookSlot(
	ctx context.Context,
	slotID, designID, email, name, phone string,
) (*BookSlotResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)

	err := validateVisitCustomer(email, name, phone)
	if err != nil {
		return nil, err
	}

	// Lazily release lapsed unpaid holds so their slots are bookable again.
	s.releaseExpiredHolds(ctx)

	slot, err := s.slots.GetByID(ctx, slotID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrSlotNotFound
		}

		return nil, fmt.Errorf("load slot: %w", err)
	}

	if slot.Status != domain.SlotStatusOpen {
		return nil, domain.ErrSlotUnavailable
	}

	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	if settings.DepositPesewas <= 0 {
		return nil, fmt.Errorf("%w: deposit amount is not configured", domain.ErrInvalidInput)
	}

	user, err := s.upsertCustomer(ctx, email, name)
	if err != nil {
		return nil, err
	}

	order, visit, paymentURL, err := s.createVisitOrder(ctx, slot, designID, phone, user, settings.DepositPesewas)
	if err != nil {
		return nil, err
	}

	s.recordOrderConfirmation(ctx, user, order)

	return &BookSlotResult{
		Visit:      visit,
		Order:      order,
		PaymentURL: paymentURL,
		User:       user,
	}, nil
}

// ListVisits returns visits matching the filter, newest first.
func (s *CalendarVisit) ListVisits(ctx context.Context, filter domain.VisitFilter) ([]domain.Visit, error) {
	visits, err := s.visits.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list visits: %w", err)
	}

	return visits, nil
}

// RescheduleVisit moves a booked visit to a different open slot and reopens
// the old slot. The new slot is claimed atomically before the old one is
// released, so a racing customer booking can never be stomped.
func (s *CalendarVisit) RescheduleVisit(ctx context.Context, visitID, newSlotID string) (*domain.Visit, error) {
	visit, err := s.loadVisitForReschedule(ctx, visitID)
	if err != nil {
		return nil, err
	}

	err = s.claimOpenSlot(ctx, newSlotID)
	if err != nil {
		return nil, err
	}

	oldSlotID := visit.SlotID
	visit.SlotID = newSlotID
	visit.UpdatedAt = s.now().UTC()

	err = s.visits.Update(ctx, visit)
	if err != nil {
		// The claim above is ours, so releasing it cannot stomp anyone else's
		// booking: the conditional update only reopens a still-booked slot.
		rbErr := s.slots.UpdateStatusFrom(ctx, newSlotID, domain.SlotStatusBooked, domain.SlotStatusOpen)
		if rbErr != nil {
			s.logger.WarnContext(ctx, "reschedule: rollback new slot failed", "error", rbErr)
		}

		return nil, fmt.Errorf("update visit: %w", err)
	}

	err = s.slots.UpdateStatusFrom(ctx, oldSlotID, domain.SlotStatusBooked, domain.SlotStatusOpen)
	if err != nil {
		s.logger.WarnContext(ctx, "reschedule: reopen old slot failed", "error", err)
	}

	return visit, nil
}

// CancelVisit cancels a visit and reopens its slot for rebooking.
func (s *CalendarVisit) CancelVisit(ctx context.Context, visitID string) (*domain.Visit, error) {
	visit, err := s.visits.GetByID(ctx, visitID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrVisitNotFound
		}

		return nil, fmt.Errorf("load visit: %w", err)
	}

	if visit.Status == domain.VisitStatusCancelled {
		return nil, domain.ErrVisitAlreadyCancelled
	}

	visit.Status = domain.VisitStatusCancelled
	visit.UpdatedAt = s.now().UTC()

	err = s.visits.Update(ctx, visit)
	if err != nil {
		return nil, fmt.Errorf("update visit: %w", err)
	}

	err = s.slots.UpdateStatusFrom(ctx, visit.SlotID, domain.SlotStatusBooked, domain.SlotStatusOpen)
	if err != nil {
		s.logger.WarnContext(ctx, "cancel: reopen slot failed", "error", err)
	}

	return visit, nil
}

// ReleaseExpiredHolds settles every lapsed unpaid hold. Bookings sweep
// opportunistically, but a background ticker calls this so holds are still
// released (and slots freed) on a quiet calendar with no new bookings.
func (s *CalendarVisit) ReleaseExpiredHolds(ctx context.Context) {
	s.releaseExpiredHolds(ctx)
}

// releaseExpiredHolds settles every lapsed unpaid hold: holds whose deposit
// order was paid in the meantime become firm bookings, the rest are cancelled
// and their slots reopened. Failures are logged and retried on the next call.
func (s *CalendarVisit) releaseExpiredHolds(ctx context.Context) {
	expired, err := s.visits.ListExpiredHolds(ctx, s.now().UTC())
	if err != nil {
		s.logger.WarnContext(ctx, "release holds: list", "error", err)

		return
	}

	for i := range expired {
		s.settleExpiredHold(ctx, &expired[i])
	}
}

// holdOutcome is the fate of a lapsed hold once its deposit has been checked.
type holdOutcome int

const (
	holdSkip    holdOutcome = iota // leave untouched and retry on the next sweep
	holdCancel                     // free the slot — the deposit will never land
	holdPromote                    // the deposit arrived — make it a firm booking
	holdExtend                     // the deposit is still settling — keep holding
)

func (s *CalendarVisit) settleExpiredHold(ctx context.Context, visit *domain.Visit) {
	switch s.classifyExpiredHold(ctx, visit) {
	case holdPromote:
		s.promoteHold(ctx, visit)
	case holdExtend:
		s.extendHold(ctx, visit)
	case holdCancel:
		s.cancelHeldVisit(ctx, visit)
	case holdSkip:
	}
}

// classifyExpiredHold decides a lapsed hold's fate. A paid order is promoted; a
// deposit still in flight (or unverifiable) keeps its slot — a webhook can lag
// past the hold window, and cancelling here would orphan a paid booking and
// resell the slot. Only a deposit the provider says will never arrive frees it.
func (s *CalendarVisit) classifyExpiredHold(ctx context.Context, visit *domain.Visit) holdOutcome {
	order, err := s.orders.GetByRef(ctx, visit.OrderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		s.logger.WarnContext(ctx, "release holds: load order", "visit", visit.ID, "error", err)

		return holdSkip
	}

	if order != nil && order.IsPaid() {
		return holdPromote
	}

	if visit.DepositPaymentID == "" {
		return holdCancel
	}

	status, verr := s.payments.VerifyTransaction(ctx, visit.DepositPaymentID)
	switch {
	case verr != nil:
		s.logger.WarnContext(ctx, "release holds: verify deposit", "visit", visit.ID, "error", verr)

		return holdExtend
	case status == depositStatusSuccess:
		return holdPromote
	case !depositDefinitelyFailed(status):
		return holdExtend
	default:
		return holdCancel
	}
}

// cancelHeldVisit cancels a lapsed visit and reopens its slot for rebooking.
func (s *CalendarVisit) cancelHeldVisit(ctx context.Context, visit *domain.Visit) {
	visit.Status = domain.VisitStatusCancelled
	visit.UpdatedAt = s.now().UTC()

	err := s.visits.Update(ctx, visit)
	if err != nil {
		s.logger.WarnContext(ctx, "release holds: cancel visit", "visit", visit.ID, "error", err)

		return
	}

	err = s.slots.UpdateStatusFrom(ctx, visit.SlotID, domain.SlotStatusBooked, domain.SlotStatusOpen)
	if err != nil {
		s.logger.WarnContext(ctx, "release holds: reopen slot", "slot", visit.SlotID, "error", err)
	}
}

// promoteHold turns a held visit into a firm booking by clearing its expiry.
func (s *CalendarVisit) promoteHold(ctx context.Context, visit *domain.Visit) {
	visit.HoldExpiresAt = nil
	visit.UpdatedAt = s.now().UTC()

	err := s.visits.Update(ctx, visit)
	if err != nil {
		s.logger.WarnContext(ctx, "release holds: promote visit", "visit", visit.ID, "error", err)
	}
}

// extendHold pushes the hold expiry out so an in-flight deposit gets another
// settlement window instead of losing its slot.
func (s *CalendarVisit) extendHold(ctx context.Context, visit *domain.Visit) {
	next := s.now().UTC().Add(slotHoldDuration)
	visit.HoldExpiresAt = &next
	visit.UpdatedAt = s.now().UTC()

	err := s.visits.Update(ctx, visit)
	if err != nil {
		s.logger.WarnContext(ctx, "release holds: extend hold", "visit", visit.ID, "error", err)
	}
}

const depositStatusSuccess = "success"

// depositDefinitelyFailed reports whether a Paystack transaction status means
// the deposit will never complete, so a lapsed hold can be safely released.
func depositDefinitelyFailed(status string) bool {
	switch status {
	case "failed", "abandoned", "reversed", "cancelled":
		return true
	default:
		return false
	}
}

func (s *CalendarVisit) loadVisitForReschedule(ctx context.Context, visitID string) (*domain.Visit, error) {
	visit, err := s.visits.GetByID(ctx, visitID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrVisitNotFound
		}

		return nil, fmt.Errorf("load visit: %w", err)
	}

	if visit.Status == domain.VisitStatusCancelled {
		return nil, domain.ErrVisitAlreadyCancelled
	}

	return visit, nil
}

// claimOpenSlot atomically moves an open slot to booked in a single
// conditional write; a slot that is missing or no longer open is reported
// without ever overwriting another booking's claim.
func (s *CalendarVisit) claimOpenSlot(ctx context.Context, slotID string) error {
	err := s.slots.UpdateStatusFrom(ctx, slotID, domain.SlotStatusOpen, domain.SlotStatusBooked)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrSlotNotFound
		}

		if errors.Is(err, domain.ErrSlotUnavailable) {
			return domain.ErrSlotUnavailable
		}

		return fmt.Errorf("claim new slot: %w", err)
	}

	return nil
}

func (s *CalendarVisit) upsertCustomer(ctx context.Context, email, name string) (*domain.User, error) {
	user := &domain.User{
		ID:        "",
		Email:     email,
		Name:      name,
		Role:      domain.RoleCustomer,
		CreatedAt: s.now().UTC(),
	}

	err := s.users.Upsert(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	return user, nil
}

func (s *CalendarVisit) createVisitOrder(
	ctx context.Context,
	slot *domain.Slot,
	designID, phone string,
	user *domain.User,
	depositPesewas int64,
) (*domain.Order, *domain.Visit, string, error) {
	designSnapshot, err := s.buildDesignSnapshot(ctx, designID, depositPesewas)
	if err != nil {
		return nil, nil, "", err
	}

	ref, err := s.generateUniqueRef(ctx)
	if err != nil {
		return nil, nil, "", err
	}

	createdAt := s.now().UTC()

	order := &domain.Order{
		Ref:            ref,
		CustomerID:     user.ID,
		DesignID:       designID,
		DesignSnapshot: designSnapshot,
		Type:           domain.OrderTypeVisit,
		Customisation: domain.Customisation{
			SizeMode: "home_visit",
		},
		Delivery:      domain.Delivery{},
		Payments:      []domain.Payment{},
		Status:        domain.OrderStatusPendingPayment,
		CustomerPhone: phone,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}

	order.RecordInitialStatus("customer", createdAt)

	paymentURL, providerRef, err := s.initDepositPayment(ctx, order, user.Email, depositPesewas)
	if err != nil {
		return nil, nil, "", err
	}

	visit, err := s.bookVisit(ctx, slot, order, providerRef)
	if err != nil {
		return nil, nil, "", err
	}

	return order, visit, paymentURL, nil
}

func (s *CalendarVisit) buildDesignSnapshot(
	ctx context.Context,
	designID string,
	depositPesewas int64,
) (domain.DesignSnapshot, error) {
	if designID == "" {
		return domain.DesignSnapshot{Name: "Home visit deposit", PricePesewas: depositPesewas}, nil
	}

	design, err := s.designs.GetByID(ctx, designID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.DesignSnapshot{}, fmt.Errorf("%w: unknown design", domain.ErrInvalidInput)
		}

		return domain.DesignSnapshot{}, fmt.Errorf("load design: %w", err)
	}

	photoPublicID := ""
	if len(design.Photos) > 0 {
		photoPublicID = design.Photos[0].PublicID
	}

	return domain.DesignSnapshot{
		Name:          design.Name,
		PhotoPublicID: photoPublicID,
		PricePesewas:  depositPesewas,
	}, nil
}

func (s *CalendarVisit) initDepositPayment(
	ctx context.Context,
	order *domain.Order,
	email string,
	depositPesewas int64,
) (string, string, error) {
	callbackURL := s.webURL + "/payments/callback"

	authURL, providerRef, err := s.payments.InitTransaction(ctx, depositPesewas, email, order.Ref, callbackURL)
	if err != nil {
		return "", "", fmt.Errorf("init payment: %w", err)
	}

	order.Payments = []domain.Payment{{
		ProviderRef:   providerRef,
		AmountPesewas: depositPesewas,
		Status:        domain.PaymentStatusPending,
		Method:        "",
		PaidAt:        nil,
	}}

	return authURL, providerRef, nil
}

func (s *CalendarVisit) bookVisit(
	ctx context.Context,
	slot *domain.Slot,
	order *domain.Order,
	providerRef string,
) (*domain.Visit, error) {
	createdAt := order.CreatedAt
	holdExpiresAt := createdAt.Add(slotHoldDuration)

	visit := &domain.Visit{
		OrderID:          "",
		SlotID:           slot.ID,
		DepositPaymentID: providerRef,
		Status:           domain.VisitStatusBooked,
		HoldExpiresAt:    &holdExpiresAt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}

	err := s.visits.BookSlot(ctx, slot.ID, visit)
	if err != nil {
		return nil, fmt.Errorf("book slot: %w", err)
	}

	rollback := func() {
		visit.Status = domain.VisitStatusCancelled
		visit.UpdatedAt = s.now().UTC()

		rbErr := s.visits.Update(ctx, visit)
		if rbErr != nil {
			s.logger.WarnContext(ctx, "book slot: rollback visit failed", "error", rbErr)
		}

		rbErr = s.slots.UpdateStatusFrom(ctx, slot.ID, domain.SlotStatusBooked, domain.SlotStatusOpen)
		if rbErr != nil {
			s.logger.WarnContext(ctx, "book slot: rollback slot failed", "error", rbErr)
		}
	}

	err = s.orders.Create(ctx, order)
	if err != nil {
		rollback()

		return nil, fmt.Errorf("create order: %w", err)
	}

	visit.OrderID = order.Ref
	visit.UpdatedAt = s.now().UTC()

	err = s.visits.Update(ctx, visit)
	if err != nil {
		rollback()

		return nil, fmt.Errorf("link visit to order: %w", err)
	}

	return visit, nil
}

func (s *CalendarVisit) generateUniqueRef(ctx context.Context) (string, error) {
	for range maxRefVisitRetries {
		ref := refVisitPrefix + randomVisitRef()

		_, err := s.orders.GetByRef(ctx, ref)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ref, nil
			}

			return "", fmt.Errorf("check ref uniqueness: %w", err)
		}
	}

	return "", fmt.Errorf("%w: could not generate unique reference", domain.ErrInvalidInput)
}

func randomVisitRef() string {
	buf := make([]byte, refVisitBytes)
	_, _ = rand.Read(buf)

	return base64.RawURLEncoding.EncodeToString(buf)
}

func validateVisitCustomer(email, name, phone string) error {
	if name == "" || len(name) > maxVisitNameLength {
		return fmt.Errorf("%w: name must be 1-%d characters", domain.ErrInvalidInput, maxVisitNameLength)
	}

	if len(email) > maxEmailLength {
		return fmt.Errorf("%w: email too long", domain.ErrInvalidInput)
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("%w: invalid email address", domain.ErrInvalidInput)
	}

	if phone == "" || len(phone) > maxVisitPhoneLength {
		return fmt.Errorf("%w: customer phone is required", domain.ErrInvalidInput)
	}

	return nil
}

func (s *CalendarVisit) recordOrderConfirmation(ctx context.Context, user *domain.User, order *domain.Order) {
	if s.email == nil {
		return
	}

	err := s.email.SendOrderConfirmation(ctx, user.Email, user.Name, order.Ref, string(order.Status))
	if err != nil {
		s.logger.WarnContext(ctx, "visit booking: send confirmation", "error", err)
	}
}
