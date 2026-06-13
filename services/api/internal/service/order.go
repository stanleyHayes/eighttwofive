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
	refPrefix            = "E25-"
	refBytes             = 6
	maxRefRetries        = 5
	maxPhoneLength       = 30
	maxOrderNameLength   = 120
	deliveryParts        = 2
	deliverySeparator    = ":"
	deliveryModePickup   = "pickup"
	deliveryModeDispatch = "dispatch"
	sizeModeBand         = "band"
	sizeModeSelf         = "self"
	sizeModeHomeVisit    = "home_visit"
	sizeModeWorkplace    = "workplace"
	defaultTimeframe     = "roughly two weeks, depending on current bookings"
	webhookActor         = "payment_webhook"
	maxUpdateRetries     = 3
)

// Order implements the order use-cases (standard checkout path and admin
// listing). It depends only on domain ports.
type Order struct {
	orders   domain.OrderRepository
	designs  domain.DesignRepository
	users    domain.UserRepository
	payments domain.PaymentProvider
	events   domain.PaymentEventRepository
	email    domain.EmailSender
	settings domain.SettingsRepository
	webURL   string
	logger   *slog.Logger
	now      func() time.Time
}

// NewOrder wires the order service.
func NewOrder(
	orders domain.OrderRepository,
	designs domain.DesignRepository,
	users domain.UserRepository,
	payments domain.PaymentProvider,
	events domain.PaymentEventRepository,
	email domain.EmailSender,
	settings domain.SettingsRepository,
	webURL string,
	logger *slog.Logger,
) *Order {
	return &Order{
		orders:   orders,
		designs:  designs,
		users:    users,
		payments: payments,
		events:   events,
		email:    email,
		settings: settings,
		webURL:   strings.TrimRight(webURL, "/"),
		logger:   logger,
		now:      time.Now,
	}
}

// StandardOrderResult is the output of a successful standard checkout.
type StandardOrderResult struct {
	Order      *domain.Order
	PaymentURL string
	User       *domain.User
}

// CreateStandardOrder creates a paid-online standard order, snapshots the
// design price, adds a delivery rate if dispatching to a known area, and
// initializes a Paystack checkout.
func (s *Order) CreateStandardOrder(
	ctx context.Context,
	designID, bandLabel, delivery, customerPhone, email, name string,
) (*StandardOrderResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	customerPhone = strings.TrimSpace(customerPhone)

	err := validateCustomer(email, name, customerPhone)
	if err != nil {
		return nil, err
	}

	design, err := s.loadLiveDesign(ctx, designID)
	if err != nil {
		return nil, err
	}

	band, ok := findBand(design.SizeBands, bandLabel)
	if !ok {
		return nil, fmt.Errorf("%w: unknown size band %q", domain.ErrInvalidInput, bandLabel)
	}

	choice, err := s.resolveDelivery(ctx, delivery)
	if err != nil {
		return nil, err
	}

	user, err := s.upsertCustomer(ctx, email, name)
	if err != nil {
		return nil, err
	}

	order, paymentURL, providerRef, err := s.createPendingOrder(
		ctx, design, band, choice, customerPhone, user.ID, email)
	if err != nil {
		return nil, err
	}

	s.recordEvent(ctx, providerRef, "transaction_initialized", nil)

	return &StandardOrderResult{Order: order, PaymentURL: paymentURL, User: user}, nil
}

// CreateCustomRequest creates a requested order for a custom size or design
// change. It snapshots the design, upserts the customer, and returns the order
// with status requested and no payment URL.
func (s *Order) CreateCustomRequest(
	ctx context.Context,
	designID string,
	customisation domain.Customisation,
	delivery, customerPhone, email, name string,
) (*domain.Order, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	customerPhone = strings.TrimSpace(customerPhone)

	err := validateCustomer(email, name, customerPhone)
	if err != nil {
		return nil, err
	}

	design, err := s.loadLiveDesign(ctx, designID)
	if err != nil {
		return nil, err
	}

	err = validateCustomRequest(customisation, design)
	if err != nil {
		return nil, err
	}

	choice, err := s.resolveDelivery(ctx, delivery)
	if err != nil {
		return nil, err
	}

	user, err := s.upsertCustomer(ctx, email, name)
	if err != nil {
		return nil, err
	}

	order := s.buildCustomOrder(design, customisation, choice, customerPhone)

	ref, err := s.generateUniqueRef(ctx)
	if err != nil {
		return nil, err
	}

	order.Ref = ref
	order.CustomerID = user.ID

	err = s.orders.Create(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("create custom request: %w", err)
	}

	return order, nil
}

// HandlePaymentWebhook verifies a Paystack webhook, records the event, and —
// for a charge.success whose amount matches the recorded pending payment —
// marks the matching order paid and books it. Non-charge events, unknown
// references, and duplicate deliveries are acknowledged without side effects
// so the provider stops retrying; amount mismatches are flagged for admin
// review instead of booking. Email notification is best-effort and never
// fails the webhook.
func (s *Order) HandlePaymentWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := s.payments.VerifyWebhook(payload, signature)
	if err != nil {
		return fmt.Errorf("verify webhook: %w", err)
	}

	s.recordEvent(ctx, event.ProviderRef, "webhook", payload)

	if event.Type != domain.WebhookEventChargeSuccess || event.Status != domain.PaymentStatusSuccess {
		return nil
	}

	for range maxUpdateRetries {
		retry, err := s.applyChargeEvent(ctx, event)
		if !retry {
			return err
		}
	}

	return fmt.Errorf("apply charge event: %w", domain.ErrConflict)
}

// GetOrder returns an order. It does not enforce authorization; callers should
// ensure the requesting user owns the order or is an admin.
func (s *Order) GetOrder(ctx context.Context, ref string) (*domain.Order, error) {
	order, err := s.orders.GetByRef(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	return order, nil
}

// ListCustomerOrders returns all orders belonging to a customer, newest first.
func (s *Order) ListCustomerOrders(ctx context.Context, customerID string) ([]domain.Order, error) {
	orders, err := s.orders.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list customer orders: %w", err)
	}

	return orders, nil
}

// ListAdminOrders returns orders sorted for the admin inbox: standard bookings,
// custom requests, then visit bookings; newest first within each bucket.
func (s *Order) ListAdminOrders(ctx context.Context) ([]domain.Order, error) {
	orders, err := s.orders.List(ctx, domain.OrderFilter{})
	if err != nil {
		return nil, fmt.Errorf("list admin orders: %w", err)
	}

	return orders, nil
}

// ListAdminOrdersPaged returns one page of admin orders with the total count,
// for the paginated admin orders table.
func (s *Order) ListAdminOrdersPaged(ctx context.Context, page, pageSize int) (domain.Page[domain.Order], error) {
	params := domain.NormalizePageParams(page, pageSize)
	filter := domain.OrderFilter{}

	total, err := s.orders.Count(ctx, filter)
	if err != nil {
		return domain.Page[domain.Order]{}, fmt.Errorf("count admin orders: %w", err)
	}

	orders, err := s.orders.ListPaged(ctx, filter, params)
	if err != nil {
		return domain.Page[domain.Order]{}, fmt.Errorf("list admin orders: %w", err)
	}

	return domain.NewPage(orders, total, params), nil
}

// GetAdminOrder loads any order for the admin dashboard. It does not enforce
// ownership; callers must ensure the request is admin-authenticated.
func (s *Order) GetAdminOrder(ctx context.Context, ref string) (*domain.Order, error) {
	order, err := s.orders.GetByRef(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get admin order: %w", err)
	}

	return order, nil
}

// UpdateQuote sets design/price/timeline notes on a custom request. Only
// allowed while the order is still in the quote phase.
func (s *Order) UpdateQuote(ctx context.Context, ref string, quote domain.Quote) error {
	if quote.PricePesewas < 0 {
		return fmt.Errorf("%w: price cannot be negative", domain.ErrInvalidInput)
	}

	_, err := s.mutateOrder(ctx, ref, func(order *domain.Order) error {
		if order.Status != domain.OrderStatusRequested && order.Status != domain.OrderStatusQuoted {
			return fmt.Errorf("%w: quote can only be updated for requested or quoted orders", domain.ErrInvalidInput)
		}

		order.Quote = quote
		if order.Status == domain.OrderStatusRequested {
			_, err := order.Transition(domain.OrderStatusQuoted, "merchant", s.now().UTC())
			if err != nil {
				return fmt.Errorf("transition to quoted: %w", err)
			}
		}

		return nil
	})

	return err
}

// SendPaymentLink initializes a provider checkout for the quoted total and
// transitions the order to payment_link_sent. The transaction uses the order
// ref as its reference and the customer's email, exactly like the standard
// checkout path, so the success webhook resolves straight back to this order.
func (s *Order) SendPaymentLink(ctx context.Context, ref string) (string, error) {
	order, err := s.orders.GetByRef(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("get order for payment link: %w", err)
	}

	err = validatePaymentLinkOrder(order)
	if err != nil {
		return "", err
	}

	user, err := s.users.GetByID(ctx, order.CustomerID)
	if err != nil {
		return "", fmt.Errorf("load customer for payment link: %w", err)
	}

	total := order.TotalPesewas()
	callbackURL := s.webURL + "/payments/callback"

	linkURL, providerRef, err := s.payments.InitTransaction(ctx, total, user.Email, order.Ref, callbackURL)
	if err != nil {
		return "", fmt.Errorf("create payment link: %w", err)
	}

	order.Payments = append(order.Payments, domain.Payment{
		ProviderRef:   providerRef,
		AmountPesewas: total,
		Status:        domain.PaymentStatusPending,
		Method:        "",
		PaidAt:        nil,
	})

	_, err = order.Transition(domain.OrderStatusPaymentLinkSent, "merchant", s.now().UTC())
	if err != nil {
		return "", fmt.Errorf("transition to payment link sent: %w", err)
	}

	err = s.orders.Update(ctx, order)
	if err != nil {
		return "", fmt.Errorf("save payment link order: %w", err)
	}

	s.recordEvent(ctx, providerRef, "payment_link_created", nil)

	return linkURL, nil
}

// validatePaymentLinkOrder enforces invariant 3: custom orders are never
// chargeable before the merchant quotes, so only a quoted order with a
// positive quote may receive a payment link.
func validatePaymentLinkOrder(order *domain.Order) error {
	if order.Status != domain.OrderStatusQuoted {
		return fmt.Errorf("%w: payment link can only be sent after quoting", domain.ErrInvalidInput)
	}

	if order.Quote.PricePesewas <= 0 {
		return fmt.Errorf("%w: a positive quote is required before sending a payment link", domain.ErrInvalidInput)
	}

	return nil
}

// MarkPaidManually records an off-platform payment (cash, direct transfer,
// etc.) attributed to the acting admin and books the order. The note is
// stored on the payment record.
func (s *Order) MarkPaidManually(ctx context.Context, ref, note, by string) error {
	order, err := s.mutateOrder(ctx, ref, func(order *domain.Order) error {
		err := validateManualPayment(order)
		if err != nil {
			return err
		}

		payment := domain.Payment{
			ProviderRef:   "manual-" + ref,
			AmountPesewas: order.TotalPesewas(),
			Status:        "",
			Method:        "manual",
			PaidAt:        nil,
		}

		now := s.now().UTC()

		_, err = order.MarkPaid(payment, by, now)
		if err != nil {
			return fmt.Errorf("mark paid manually: %w", err)
		}

		applyManualNote(order, payment.ProviderRef, note)
		bookAfterManualPayment(order, by, now)

		return nil
	})
	if err != nil {
		return err
	}

	s.notifyCustomerOfManualPayment(ctx, order)

	return nil
}

// manualPayable reports whether an admin may mark an order in this state paid
// off-platform; paid, terminal, and in-progress orders are out.
func manualPayable(status domain.OrderStatus) bool {
	switch status {
	case domain.OrderStatusPendingPayment,
		domain.OrderStatusRequested,
		domain.OrderStatusQuoted,
		domain.OrderStatusPaymentLinkSent:
		return true
	case domain.OrderStatusBooked,
		domain.OrderStatusInProduction,
		domain.OrderStatusReady,
		domain.OrderStatusFulfilled,
		domain.OrderStatusCancelled:
		return false
	default:
		return false
	}
}

func validateManualPayment(order *domain.Order) error {
	if order.IsPaid() {
		return fmt.Errorf("%w: order is already paid", domain.ErrInvalidInput)
	}

	if !manualPayable(order.Status) {
		return fmt.Errorf("%w: a %s order cannot be marked paid", domain.ErrInvalidInput, order.Status)
	}

	// Orders still in the quote phase are custom requests: they must carry a
	// merchant quote before any payment — a delivery rate alone is not a price.
	inQuotePhase := order.Status == domain.OrderStatusRequested || order.Status == domain.OrderStatusQuoted
	if inQuotePhase && order.Quote.PricePesewas <= 0 {
		return fmt.Errorf("%w: custom orders need a quote before payment", domain.ErrInvalidInput)
	}

	if order.TotalPesewas() <= 0 {
		return fmt.Errorf("%w: order total must be greater than zero", domain.ErrInvalidInput)
	}

	return nil
}

// UpdateOrderStatus transitions an order to an admin-selected status. Unknown
// statuses are rejected outright, booked is reserved for the payment webhook
// and the manual mark-paid flow, and the domain transition table guards every
// remaining edge (including paid-before-production).
func (s *Order) UpdateOrderStatus(ctx context.Context, ref string, status domain.OrderStatus, by string) error {
	if !domain.KnownOrderStatus(status) {
		return fmt.Errorf("%w: unknown status %q", domain.ErrInvalidInput, status)
	}

	if status == domain.OrderStatusBooked {
		return fmt.Errorf("%w: orders are booked through payment or mark-paid", domain.ErrInvalidInput)
	}

	order, err := s.mutateOrder(ctx, ref, func(order *domain.Order) error {
		_, err := order.Transition(status, by, s.now().UTC())
		if err != nil {
			return fmt.Errorf("transition order: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusInProduction || order.Status == domain.OrderStatusReady {
		s.notifyStatusChange(ctx, order)
	}

	return nil
}

// applyChargeEvent loads the order behind a successful charge and applies it
// once. It reports retry=true when a concurrent writer invalidated the write.
func (s *Order) applyChargeEvent(ctx context.Context, event *domain.WebhookEvent) (bool, error) {
	order, err := s.orders.GetByRef(ctx, event.ProviderRef)
	if errors.Is(err, domain.ErrNotFound) {
		s.logger.WarnContext(ctx, "payment webhook: no order for reference", "reference", event.ProviderRef)

		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("find order by provider ref: %w", err)
	}

	if paymentByRef(order, event.ProviderRef, domain.PaymentStatusSuccess) != nil {
		s.logger.InfoContext(ctx, "payment webhook: duplicate delivery ignored", "ref", order.Ref)

		return false, nil
	}

	pending := paymentByRef(order, event.ProviderRef, "")
	if pending == nil {
		s.logger.WarnContext(ctx, "payment webhook: no recorded payment for reference",
			"ref", order.Ref, "reference", event.ProviderRef)

		return false, nil
	}

	if event.AmountPesewas != pending.AmountPesewas {
		return s.flagAmountMismatch(ctx, order, pending, event)
	}

	return s.bookPaidOrder(ctx, order, event)
}

// flagAmountMismatch keeps the expected amount on the payment record, marks it
// for admin attention, and acknowledges the event so the provider stops
// retrying. The order is never booked from a mismatched charge.
func (s *Order) flagAmountMismatch(
	ctx context.Context,
	order *domain.Order,
	pending *domain.Payment,
	event *domain.WebhookEvent,
) (bool, error) {
	s.logger.WarnContext(ctx, "payment webhook: amount mismatch",
		"ref", order.Ref,
		"expectedPesewas", pending.AmountPesewas,
		"receivedPesewas", event.AmountPesewas,
	)

	mismatch := fmt.Sprintf(`{"reference":%q,"expectedPesewas":%d,"receivedPesewas":%d}`,
		event.ProviderRef, pending.AmountPesewas, event.AmountPesewas)
	s.recordEvent(ctx, event.ProviderRef, "amount_mismatch", []byte(mismatch))

	pending.Status = domain.PaymentStatusMismatch
	order.UpdatedAt = s.now().UTC()

	err := s.orders.Update(ctx, order)
	if errors.Is(err, domain.ErrConflict) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("save amount mismatch: %w", err)
	}

	return false, nil
}

// bookPaidOrder records the verified payment and books the order, notifying
// the customer only when the status actually changed.
func (s *Order) bookPaidOrder(ctx context.Context, order *domain.Order, event *domain.WebhookEvent) (bool, error) {
	payment := domain.Payment{
		ProviderRef:   event.ProviderRef,
		AmountPesewas: event.AmountPesewas,
		Status:        "",
		Method:        "",
		PaidAt:        nil,
	}

	prev, err := order.MarkPaid(payment, webhookActor, s.now().UTC())
	if err != nil {
		return false, fmt.Errorf("mark paid: %w", err)
	}

	err = s.orders.Update(ctx, order)
	if errors.Is(err, domain.ErrConflict) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("update order: %w", err)
	}

	if prev != order.Status {
		s.notifyStatusChange(ctx, order)
	}

	return false, nil
}

// mutateOrder loads an order by ref, applies mutate, and saves it, retrying
// the whole cycle when a concurrent writer bumped the order version first.
func (s *Order) mutateOrder(
	ctx context.Context,
	ref string,
	mutate func(order *domain.Order) error,
) (*domain.Order, error) {
	for range maxUpdateRetries {
		order, err := s.orders.GetByRef(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("get order: %w", err)
		}

		err = mutate(order)
		if err != nil {
			return nil, err
		}

		err = s.orders.Update(ctx, order)
		if errors.Is(err, domain.ErrConflict) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("update order: %w", err)
		}

		return order, nil
	}

	return nil, fmt.Errorf("update order: %w", domain.ErrConflict)
}

// paymentByRef returns the order payment matching the provider reference, or
// nil. An empty status matches any payment status.
func paymentByRef(order *domain.Order, providerRef, status string) *domain.Payment {
	for i := range order.Payments {
		p := &order.Payments[i]
		if p.ProviderRef == providerRef && (status == "" || p.Status == status) {
			return p
		}
	}

	return nil
}

func applyManualNote(order *domain.Order, providerRef, note string) {
	for i := range order.Payments {
		if order.Payments[i].ProviderRef == providerRef {
			order.Payments[i].Method = "manual: " + note
		}
	}
}

// bookAfterManualPayment books a quoted order after a manual payment;
// pending_payment and payment_link_sent orders were already booked inside
// MarkPaid, and every other state is rejected by validateManualPayment.
func bookAfterManualPayment(order *domain.Order, by string, at time.Time) {
	if order.Status != domain.OrderStatusQuoted {
		return
	}

	_, _ = order.Transition(domain.OrderStatusBooked, by, at)
}

func (s *Order) notifyCustomerOfManualPayment(ctx context.Context, order *domain.Order) {
	s.notifyStatusChange(ctx, order)
}

func (s *Order) notifyStatusChange(ctx context.Context, order *domain.Order) {
	user, err := s.users.GetByID(ctx, order.CustomerID)
	if err != nil {
		s.logger.WarnContext(ctx, "status notification: load user", "error", err)

		return
	}

	label := customerFacingStatus(order.Status)
	if label == "" {
		return
	}

	timeframe := order.Quote.Timeline
	if timeframe == "" {
		timeframe = defaultTimeframe
	}

	err = s.email.SendOrderStatusUpdate(ctx, user.Email, user.Name, order.Ref, label, timeframe)
	if err != nil {
		s.logger.WarnContext(ctx, "status notification: send email", "error", err)
	}
}

func customerFacingStatus(status domain.OrderStatus) string {
	switch status {
	case domain.OrderStatusBooked:
		return "order confirmed"
	case domain.OrderStatusInProduction:
		return "in production"
	case domain.OrderStatusReady:
		return "ready"
	case domain.OrderStatusPendingPayment,
		domain.OrderStatusRequested,
		domain.OrderStatusQuoted,
		domain.OrderStatusPaymentLinkSent,
		domain.OrderStatusFulfilled,
		domain.OrderStatusCancelled:
		return ""
	default:
		return ""
	}
}

func (s *Order) loadLiveDesign(ctx context.Context, designID string) (*domain.Design, error) {
	design, err := s.designs.GetByID(ctx, designID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: unknown design", domain.ErrInvalidInput)
		}

		return nil, fmt.Errorf("load design: %w", err)
	}

	if design.Status != domain.StatusLive {
		return nil, fmt.Errorf("%w: design is not available", domain.ErrInvalidInput)
	}

	return design, nil
}

func (s *Order) upsertCustomer(ctx context.Context, email, name string) (*domain.User, error) {
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

type deliveryChoice struct {
	mode string
	area string
	rate *int64
}

func (s *Order) createPendingOrder(
	ctx context.Context,
	design *domain.Design,
	band domain.SizeBand,
	choice deliveryChoice,
	customerPhone, customerID, email string,
) (*domain.Order, string, string, error) {
	order := s.buildStandardOrder(design, band, choice, customerPhone)

	ref, err := s.generateUniqueRef(ctx)
	if err != nil {
		return nil, "", "", err
	}

	order.Ref = ref
	order.CustomerID = customerID

	total := order.TotalPesewas()
	callbackURL := s.webURL + "/payments/callback"

	authURL, providerRef, err := s.payments.InitTransaction(ctx, total, email, order.Ref, callbackURL)
	if err != nil {
		return nil, "", "", fmt.Errorf("init payment: %w", err)
	}

	order.Payments = []domain.Payment{{
		ProviderRef:   providerRef,
		AmountPesewas: total,
		Status:        domain.PaymentStatusPending,
		Method:        "",
		PaidAt:        nil,
	}}

	err = s.orders.Create(ctx, order)
	if err != nil {
		return nil, "", "", fmt.Errorf("create order: %w", err)
	}

	return order, authURL, providerRef, nil
}

func (s *Order) buildStandardOrder(
	design *domain.Design,
	band domain.SizeBand,
	choice deliveryChoice,
	customerPhone string,
) *domain.Order {
	photoPublicID := ""
	if len(design.Photos) > 0 {
		photoPublicID = design.Photos[0].PublicID
	}

	createdAt := s.now().UTC()

	order := &domain.Order{
		Ref:        "",
		CustomerID: "",
		DesignID:   design.ID,
		DesignSnapshot: domain.DesignSnapshot{
			Name:          design.Name,
			PhotoPublicID: photoPublicID,
			PricePesewas:  band.PricePesewas,
		},
		Type: domain.OrderTypeStandard,
		Customisation: domain.Customisation{
			SizeMode:  sizeModeBand,
			BandLabel: band.Label,
		},
		Delivery: domain.Delivery{
			Mode:        choice.mode,
			Area:        choice.area,
			RatePesewas: choice.rate,
		},
		Status:        domain.OrderStatusPendingPayment,
		CustomerPhone: customerPhone,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}

	order.RecordInitialStatus("customer", createdAt)

	return order
}

func (s *Order) buildCustomOrder(
	design *domain.Design,
	customisation domain.Customisation,
	choice deliveryChoice,
	customerPhone string,
) *domain.Order {
	photoPublicID := ""
	if len(design.Photos) > 0 {
		photoPublicID = design.Photos[0].PublicID
	}

	createdAt := s.now().UTC()

	orderType := domain.OrderTypeCustomSize
	if customisation.SizeMode == sizeModeHomeVisit {
		orderType = domain.OrderTypeVisit
	} else if customisation.DesignChange != "" {
		orderType = domain.OrderTypeDesignChange
	}

	order := &domain.Order{
		Ref:        "",
		CustomerID: "",
		DesignID:   design.ID,
		DesignSnapshot: domain.DesignSnapshot{
			Name:          design.Name,
			PhotoPublicID: photoPublicID,
			PricePesewas:  0,
		},
		Type: orderType,
		Customisation: domain.Customisation{
			SizeMode:     customisation.SizeMode,
			BandLabel:    customisation.BandLabel,
			Measurements: customisation.Measurements,
			DesignChange: customisation.DesignChange,
		},
		Delivery: domain.Delivery{
			Mode:        choice.mode,
			Area:        choice.area,
			RatePesewas: choice.rate,
		},
		Status:        domain.OrderStatusRequested,
		CustomerPhone: customerPhone,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}

	order.RecordInitialStatus("customer", createdAt)

	return order
}

func validateCustomRequest(customisation domain.Customisation, design *domain.Design) error {
	switch customisation.SizeMode {
	case sizeModeBand, sizeModeSelf, sizeModeHomeVisit, sizeModeWorkplace:
		// ok
	default:
		return fmt.Errorf("%w: invalid size mode", domain.ErrInvalidInput)
	}

	if customisation.SizeMode == sizeModeBand {
		if customisation.BandLabel == "" {
			return fmt.Errorf("%w: band label is required", domain.ErrInvalidInput)
		}

		if _, ok := findBand(design.SizeBands, customisation.BandLabel); !ok {
			return fmt.Errorf("%w: unknown size band", domain.ErrInvalidInput)
		}
	}

	return nil
}

func (s *Order) resolveDelivery(ctx context.Context, delivery string) (deliveryChoice, error) {
	parts := strings.SplitN(delivery, deliverySeparator, deliveryParts)
	mode := strings.ToLower(strings.TrimSpace(parts[0]))

	switch mode {
	case deliveryModePickup:
		return deliveryChoice{mode: deliveryModePickup}, nil
	case deliveryModeDispatch:
		area, err := parseDispatchArea(parts)
		if err != nil {
			return deliveryChoice{}, err
		}

		rate, found, err := s.lookupDeliveryRate(ctx, area)
		if err != nil {
			return deliveryChoice{}, err
		}

		var ratePtr *int64
		if found {
			ratePtr = &rate
		}

		return deliveryChoice{mode: deliveryModeDispatch, area: area, rate: ratePtr}, nil
	default:
		return deliveryChoice{}, fmt.Errorf("%w: delivery mode must be pickup or dispatch:area", domain.ErrInvalidInput)
	}
}

func parseDispatchArea(parts []string) (string, error) {
	if len(parts) != deliveryParts {
		return "", fmt.Errorf("%w: dispatch requires an area", domain.ErrInvalidInput)
	}

	area := strings.TrimSpace(parts[1])
	if area == "" {
		return "", fmt.Errorf("%w: dispatch area cannot be empty", domain.ErrInvalidInput)
	}

	return area, nil
}

func (s *Order) lookupDeliveryRate(ctx context.Context, area string) (int64, bool, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("load settings: %w", err)
	}

	for _, rate := range settings.DeliveryRates {
		if strings.EqualFold(rate.Area, area) {
			return rate.RatePesewas, true, nil
		}
	}

	return 0, false, nil
}

func validateCustomer(email, name, phone string) error {
	if name == "" || len(name) > maxOrderNameLength {
		return fmt.Errorf("%w: name must be 1-%d characters", domain.ErrInvalidInput, maxOrderNameLength)
	}

	if len(email) > maxEmailLength {
		return fmt.Errorf("%w: email too long", domain.ErrInvalidInput)
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("%w: invalid email address", domain.ErrInvalidInput)
	}

	if phone == "" || len(phone) > maxPhoneLength {
		return fmt.Errorf("%w: customer phone is required", domain.ErrInvalidInput)
	}

	return nil
}

func findBand(bands []domain.SizeBand, label string) (domain.SizeBand, bool) {
	for _, b := range bands {
		if strings.EqualFold(b.Label, label) {
			return b, true
		}
	}

	return domain.SizeBand{}, false
}

func (s *Order) generateUniqueRef(ctx context.Context) (string, error) {
	for range maxRefRetries {
		ref := refPrefix + randomRef()

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

func randomRef() string {
	buf := make([]byte, refBytes)
	_, _ = rand.Read(buf)

	return base64.RawURLEncoding.EncodeToString(buf)
}

func (s *Order) recordEvent(ctx context.Context, providerRef, eventType string, payload []byte) {
	if s.events == nil {
		return
	}

	event := domain.PaymentEvent{
		ProviderRef: providerRef,
		Provider:    "paystack",
		Type:        eventType,
		Payload:     payload,
		CreatedAt:   s.now().UTC(),
	}

	err := s.events.RecordEvent(ctx, event)
	if err != nil {
		s.logger.WarnContext(ctx, "record payment event", "error", err)
	}
}
