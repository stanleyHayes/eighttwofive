package service_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/paystack"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

// --- fakes --------------------------------------------------------------------

type fakeOrderRepo struct {
	byID   map[string]*domain.Order
	byRef  map[string]*domain.Order
	nextID int
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{
		byID:   map[string]*domain.Order{},
		byRef:  map[string]*domain.Order{},
		nextID: 1,
	}
}

func (f *fakeOrderRepo) Create(_ context.Context, o *domain.Order) error {
	if _, exists := f.byRef[o.Ref]; exists {
		return domain.ErrDuplicateRef
	}

	o.ID = "ord-" + strconv.Itoa(f.nextID)
	f.nextID++
	clone := *o
	f.byID[o.ID] = &clone
	f.byRef[o.Ref] = &clone

	return nil
}

func (f *fakeOrderRepo) Update(_ context.Context, o *domain.Order) error {
	f.byID[o.ID] = o
	f.byRef[o.Ref] = o

	return nil
}

func (f *fakeOrderRepo) GetByID(_ context.Context, id string) (*domain.Order, error) {
	if o, ok := f.byID[id]; ok {
		clone := *o

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (f *fakeOrderRepo) GetByRef(_ context.Context, ref string) (*domain.Order, error) {
	if o, ok := f.byRef[ref]; ok {
		clone := *o

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (f *fakeOrderRepo) ListByCustomer(_ context.Context, customerID string) ([]domain.Order, error) {
	var out []domain.Order

	for _, o := range f.byID {
		if o.CustomerID == customerID {
			clone := *o
			out = append(out, clone)
		}
	}

	return out, nil
}

func (f *fakeOrderRepo) List(_ context.Context, _ domain.OrderFilter) ([]domain.Order, error) {
	out := make([]domain.Order, 0, len(f.byID))
	for _, o := range f.byID {
		clone := *o
		out = append(out, clone)
	}

	slices.SortFunc(out, func(a, b domain.Order) int {
		if a.Type != b.Type {
			return strings.Compare(string(a.Type), string(b.Type))
		}

		return b.CreatedAt.Compare(a.CreatedAt)
	})

	return out, nil
}

func (f *fakeOrderRepo) Count(ctx context.Context, filter domain.OrderFilter) (int64, error) {
	out, err := f.List(ctx, filter)

	return int64(len(out)), err
}

func (f *fakeOrderRepo) ListPaged(
	ctx context.Context, filter domain.OrderFilter, params domain.PageParams,
) ([]domain.Order, error) {
	out, err := f.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return catalogPageSlice(out, params), nil
}

type fakeDesignRepo struct {
	byID map[string]*domain.Design
}

func newFakeDesignRepo() *fakeDesignRepo {
	return &fakeDesignRepo{byID: map[string]*domain.Design{}}
}

func (f *fakeDesignRepo) Create(_ context.Context, d *domain.Design) error {
	f.byID[d.ID] = d

	return nil
}

func (f *fakeDesignRepo) Update(_ context.Context, d *domain.Design) error {
	f.byID[d.ID] = d

	return nil
}

func (f *fakeDesignRepo) GetByID(_ context.Context, id string) (*domain.Design, error) {
	if d, ok := f.byID[id]; ok {
		clone := *d

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (f *fakeDesignRepo) GetBySlug(_ context.Context, slug string) (*domain.Design, error) {
	for _, d := range f.byID {
		if d.Slug == slug {
			clone := *d

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (f *fakeDesignRepo) List(_ context.Context, _ domain.DesignFilter) ([]domain.Design, error) {
	out := make([]domain.Design, 0, len(f.byID))
	for _, d := range f.byID {
		clone := *d
		out = append(out, clone)
	}

	return out, nil
}

func (f *fakeDesignRepo) Count(ctx context.Context, filter domain.DesignFilter) (int64, error) {
	out, err := f.List(ctx, filter)

	return int64(len(out)), err
}

func (f *fakeDesignRepo) ListPaged(
	ctx context.Context, filter domain.DesignFilter, params domain.PageParams,
) ([]domain.Design, error) {
	out, err := f.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return catalogPageSlice(out, params), nil
}

func (f *fakeDesignRepo) SetStatusBulk(_ context.Context, _ []string, _ domain.Status, _ time.Time) error {
	return nil
}

func (f *fakeDesignRepo) SetStatusByCollection(_ context.Context, _ string, _ domain.Status, _ time.Time) error {
	return nil
}

func (f *fakeDesignRepo) Delete(_ context.Context, _ string) error { return nil }

func (f *fakeDesignRepo) DeleteByCollection(_ context.Context, _ string) error { return nil }

type fakePaymentProvider struct {
	secret       string
	authURL      string
	verifyStatus string
	linkURL      string
}

func newFakePaymentProvider() *fakePaymentProvider {
	return &fakePaymentProvider{
		secret:       "secret",
		authURL:      "https://checkout.test/pay",
		verifyStatus: "success",
		linkURL:      "https://pay.test/link",
	}
}

func (f *fakePaymentProvider) InitTransaction(
	_ context.Context, _ int64, _, reference, _ string,
) (string, string, error) {
	return f.authURL, reference, nil
}

func (f *fakePaymentProvider) VerifyWebhook(payload []byte, signature string) (*domain.WebhookEvent, error) {
	mac := hmac.New(sha512.New, []byte(f.secret))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, domain.ErrWebhookInvalid
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`
			Status    string `json:"status"`
		} `json:"data"`
	}

	err := json.Unmarshal(payload, &event)
	if err != nil {
		return nil, domain.ErrWebhookInvalid
	}

	status := event.Data.Status
	if status == "" {
		status = "success"
	}

	return &domain.WebhookEvent{
		Type:          event.Event,
		ProviderRef:   event.Data.Reference,
		Status:        status,
		AmountPesewas: event.Data.Amount,
	}, nil
}

func (f *fakePaymentProvider) VerifyTransaction(_ context.Context, _ string) (string, error) {
	return f.verifyStatus, nil
}

type fakePaymentEvents struct {
	events []domain.PaymentEvent
}

func (f *fakePaymentEvents) RecordEvent(_ context.Context, event domain.PaymentEvent) error {
	f.events = append(f.events, event)

	return nil
}

type fakeSettingsRepo struct {
	settings *domain.Settings
}

func (f *fakeSettingsRepo) Get(_ context.Context) (*domain.Settings, error) {
	if f.settings == nil {
		return domain.DefaultSettings(), nil
	}

	clone := *f.settings

	return &clone, nil
}

func (f *fakeSettingsRepo) Update(_ context.Context, s *domain.Settings) error {
	clone := *s
	f.settings = &clone

	return nil
}

type recordingOrderSender struct {
	lastTo        string
	lastName      string
	lastRef       string
	lastStatus    string
	lastTimeframe string
	statusUpdates int
}

func (r *recordingOrderSender) SendWelcome(context.Context, string, string) error   { return nil }
func (r *recordingOrderSender) SendLoginLink(context.Context, string, string) error { return nil }
func (r *recordingOrderSender) SendOrderConfirmation(_ context.Context, to, name, ref, status string) error {
	r.lastTo = to
	r.lastName = name
	r.lastRef = ref
	r.lastStatus = status

	return nil
}

func (r *recordingOrderSender) SendOrderStatusUpdate(
	_ context.Context, to, name, ref, status, timeframe string,
) error {
	r.lastTo = to
	r.lastName = name
	r.lastRef = ref
	r.lastStatus = status
	r.lastTimeframe = timeframe
	r.statusUpdates++

	return nil
}

// --- helpers ------------------------------------------------------------------

func newOrderService(
	orders *fakeOrderRepo,
	designs *fakeDesignRepo,
	users *fakeUsers,
	payments *fakePaymentProvider,
	events *fakePaymentEvents,
	sender *recordingOrderSender,
	settings *fakeSettingsRepo,
) *service.Order {
	logger := slog.New(slog.DiscardHandler)

	return service.NewOrder(orders, designs, users, payments, events, sender, settings, "https://shop.test", logger)
}

func liveDesign() *domain.Design {
	return &domain.Design{
		ID:           "des-1",
		CollectionID: "col-1",
		Name:         "Boardroom Blazer",
		Slug:         "boardroom-blazer",
		Note:         "",
		Photos:       []domain.Photo{{PublicID: "e25/blazer", Order: 0}},
		SizeBands: []domain.SizeBand{
			{Label: "8", PricePesewas: 50000, Chart: map[string]string{"bust": "86 cm"}},
		},
		Status:    domain.StatusLive,
		CreatedAt: time.Now().UTC(),
	}
}

func signWebhook(t *testing.T, secret string, payload []byte) string {
	t.Helper()

	mac := hmac.New(sha512.New, []byte(secret))
	_, _ = mac.Write(payload)

	return hex.EncodeToString(mac.Sum(nil))
}

// --- tests --------------------------------------------------------------------

func TestCreateStandardOrder_Pickup(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()
	payments := newFakePaymentProvider()
	events := &fakePaymentEvents{}
	sender := &recordingOrderSender{}
	settings := &fakeSettingsRepo{}

	svc := newOrderService(orders, designs, users, payments, events, sender, settings)

	result, err := svc.CreateStandardOrder(t.Context(), "des-1", "8", "pickup", "+233200000000", "ama@example.com", "Ama")
	require.NoError(t, err)

	assert.Equal(t, domain.OrderTypeStandard, result.Order.Type)
	assert.Equal(t, "band", result.Order.Customisation.SizeMode)
	assert.Equal(t, "8", result.Order.Customisation.BandLabel)
	assert.Equal(t, int64(50000), result.Order.TotalPesewas())
	assert.Equal(t, domain.OrderStatusPendingPayment, result.Order.Status)
	assert.Equal(t, "https://checkout.test/pay", result.PaymentURL)
	assert.NotEmpty(t, result.Order.Ref)
	assert.Equal(t, result.User.ID, result.Order.CustomerID)

	user := users.byEmail["ama@example.com"]
	require.NotNil(t, user)
	assert.Equal(t, "Ama", user.Name)
}

func TestCreateStandardOrder_DispatchWithRate(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()
	payments := newFakePaymentProvider()
	events := &fakePaymentEvents{}
	sender := &recordingOrderSender{}
	settings := &fakeSettingsRepo{settings: &domain.Settings{
		DepositPesewas: 50000,
		DeliveryRates:  []domain.DeliveryRate{{Area: "East Legon", RatePesewas: 2000}},
	}}

	svc := newOrderService(orders, designs, users, payments, events, sender, settings)

	result, err := svc.CreateStandardOrder(t.Context(), "des-1", "8", "dispatch:East Legon",
		"+233200000000", "ama@example.com", "Ama")
	require.NoError(t, err)

	assert.Equal(t, "dispatch", result.Order.Delivery.Mode)
	assert.Equal(t, "East Legon", result.Order.Delivery.Area)
	require.NotNil(t, result.Order.Delivery.RatePesewas)
	assert.Equal(t, int64(2000), *result.Order.Delivery.RatePesewas)
	assert.Equal(t, int64(52000), result.Order.TotalPesewas())
}

func TestCreateStandardOrder_DispatchUnservedArea(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()
	payments := newFakePaymentProvider()
	events := &fakePaymentEvents{}
	sender := &recordingOrderSender{}
	settings := &fakeSettingsRepo{settings: &domain.Settings{DeliveryRates: []domain.DeliveryRate{}}}

	svc := newOrderService(orders, designs, users, payments, events, sender, settings)

	// Dispatching to an area with no configured rate must be rejected, not
	// silently shipped for free.
	_, err := svc.CreateStandardOrder(t.Context(), "des-1", "8", "dispatch:Unknown Area",
		"+233200000000", "ama@example.com", "Ama")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCreateStandardOrder_Validation(t *testing.T) {
	t.Parallel()

	svc := newOrderService(
		newFakeOrderRepo(), newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{},
	)

	cases := []struct {
		name string
		call func() (*service.StandardOrderResult, error)
	}{
		{"missing email", func() (*service.StandardOrderResult, error) {
			return svc.CreateStandardOrder(t.Context(), "des-1", "8", "pickup", "+233200000000", "", "Ama")
		}},
		{"missing phone", func() (*service.StandardOrderResult, error) {
			return svc.CreateStandardOrder(t.Context(), "des-1", "8", "pickup", "", "ama@example.com", "Ama")
		}},
		{"unknown design", func() (*service.StandardOrderResult, error) {
			return svc.CreateStandardOrder(t.Context(), "missing", "8", "pickup", "+233200000000", "ama@example.com", "Ama")
		}},
		{"unknown band", func() (*service.StandardOrderResult, error) {
			designs := newFakeDesignRepo()
			designs.byID["des-1"] = liveDesign()
			s := newOrderService(
				newFakeOrderRepo(), designs, newFakeUsers(), newFakePaymentProvider(),
				&fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{},
			)

			return s.CreateStandardOrder(t.Context(), "des-1", "99", "pickup", "+233200000000", "ama@example.com", "Ama")
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

func TestHandlePaymentWebhook_BooksOrder(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()
	payments := newFakePaymentProvider()
	events := &fakePaymentEvents{}
	sender := &recordingOrderSender{}
	settings := &fakeSettingsRepo{}

	svc := newOrderService(orders, designs, users, payments, events, sender, settings)

	result, err := svc.CreateStandardOrder(t.Context(), "des-1", "8", "pickup",
		"+233200000000", "ama@example.com", "Ama")
	require.NoError(t, err)

	payload := []byte(`{"event":"charge.success","data":{"reference":"` +
		result.Order.Payments[0].ProviderRef + `","amount":50000}}`)
	sig := signWebhook(t, "secret", payload)

	err = svc.HandlePaymentWebhook(t.Context(), payload, sig)
	require.NoError(t, err)

	loaded, err := orders.GetByRef(t.Context(), result.Order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, loaded.Status)
	assert.True(t, loaded.IsPaid())
	assert.Equal(t, "ama@example.com", sender.lastTo)
	assert.Equal(t, "order confirmed", sender.lastStatus)
	assert.Equal(t, "roughly two weeks, depending on current bookings", sender.lastTimeframe)
}

func TestHandlePaymentWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	svc := newOrderService(
		newFakeOrderRepo(), newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{},
	)

	err := svc.HandlePaymentWebhook(t.Context(), []byte(`{}`), "bad")
	assert.ErrorIs(t, err, domain.ErrWebhookInvalid)
}

// webhookFixtures wires an order service around a freshly created standard
// order so webhook scenarios can be exercised against a known pending payment.
type webhookFixtures struct {
	svc    *service.Order
	orders *fakeOrderRepo
	sender *recordingOrderSender
	order  *domain.Order
}

func newWebhookFixtures(t *testing.T) *webhookFixtures {
	t.Helper()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	sender := &recordingOrderSender{}

	svc := newOrderService(orders, designs, newFakeUsers(), newFakePaymentProvider(),
		&fakePaymentEvents{}, sender, &fakeSettingsRepo{})

	result, err := svc.CreateStandardOrder(t.Context(), "des-1", "8", "pickup",
		"+233200000000", "ama@example.com", "Ama")
	require.NoError(t, err)

	return &webhookFixtures{svc: svc, orders: orders, sender: sender, order: result.Order}
}

// deliver posts a signed webhook payload and requires the provider-facing
// result to be a 200-style acknowledgement (nil error).
func (f *webhookFixtures) deliver(t *testing.T, payload string) {
	t.Helper()

	raw := []byte(payload)
	err := f.svc.HandlePaymentWebhook(t.Context(), raw, signWebhook(t, "secret", raw))
	require.NoError(t, err)
}

func TestHandlePaymentWebhook_AmountMismatch_DoesNotBook(t *testing.T) {
	t.Parallel()

	fixtures := newWebhookFixtures(t)

	// The order total is 50000 pesewas; a signed webhook for 1 pesewa must
	// never book it, but must be acknowledged so the provider stops retrying.
	fixtures.deliver(t, `{"event":"charge.success","data":{"reference":"`+
		fixtures.order.Payments[0].ProviderRef+`","amount":1}}`)

	loaded, err := fixtures.orders.GetByRef(t.Context(), fixtures.order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPendingPayment, loaded.Status)
	assert.False(t, loaded.IsPaid())
	require.Len(t, loaded.Payments, 1)
	assert.Equal(t, domain.PaymentStatusMismatch, loaded.Payments[0].Status)
	// The originally expected amount stays on the record for admin review.
	assert.Equal(t, int64(50000), loaded.Payments[0].AmountPesewas)
	assert.Equal(t, 0, fixtures.sender.statusUpdates)
}

func TestHandlePaymentWebhook_DuplicateDelivery_IsNoOp(t *testing.T) {
	t.Parallel()

	fixtures := newWebhookFixtures(t)
	payload := `{"event":"charge.success","data":{"reference":"` +
		fixtures.order.Payments[0].ProviderRef + `","amount":50000}}`

	fixtures.deliver(t, payload)
	fixtures.deliver(t, payload)

	loaded, err := fixtures.orders.GetByRef(t.Context(), fixtures.order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, loaded.Status)
	require.Len(t, loaded.Payments, 1)
	assert.Equal(t, 1, fixtures.sender.statusUpdates, "duplicate delivery must not re-send email")
}

func TestHandlePaymentWebhook_UnknownReference_Acks(t *testing.T) {
	t.Parallel()

	fixtures := newWebhookFixtures(t)

	// Unknown references must be acknowledged, not retried by the provider.
	fixtures.deliver(t, `{"event":"charge.success","data":{"reference":"E25-NOPE","amount":50000}}`)

	loaded, err := fixtures.orders.GetByRef(t.Context(), fixtures.order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPendingPayment, loaded.Status)
}

func TestHandlePaymentWebhook_IgnoresNonChargeEvents(t *testing.T) {
	t.Parallel()

	fixtures := newWebhookFixtures(t)

	// A signed transfer.success carrying the order reference must not book it.
	fixtures.deliver(t, `{"event":"transfer.success","data":{"reference":"`+
		fixtures.order.Payments[0].ProviderRef+`","amount":50000,"status":"success"}}`)

	loaded, err := fixtures.orders.GetByRef(t.Context(), fixtures.order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPendingPayment, loaded.Status)
	assert.False(t, loaded.IsPaid())
}

func TestHandlePaymentWebhook_NoPendingPayment_DoesNotBook(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-NOPAY")))

	sender := &recordingOrderSender{}
	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, sender, &fakeSettingsRepo{})

	// A signed charge for an order ref with no recorded pending payment (the
	// merchant never initialized one) must be acknowledged without booking.
	payload := []byte(`{"event":"charge.success","data":{"reference":"E25-NOPAY","amount":1}}`)
	err := svc.HandlePaymentWebhook(t.Context(), payload, signWebhook(t, "secret", payload))
	require.NoError(t, err)

	loaded, err := orders.GetByRef(t.Context(), "E25-NOPAY")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusRequested, loaded.Status)
	assert.False(t, loaded.IsPaid())
	assert.Empty(t, loaded.Payments)
}

func TestOrderStateMachine_NoUnpaidInProduction(t *testing.T) {
	t.Parallel()

	order := &domain.Order{
		Status:         domain.OrderStatusPendingPayment,
		DesignSnapshot: domain.DesignSnapshot{PricePesewas: 50000},
	}

	_, err := order.Transition(domain.OrderStatusInProduction, "merchant", time.Now().UTC())
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = order.MarkPaid(domain.Payment{ProviderRef: "ps-1", AmountPesewas: 50000}, "payment_webhook", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, order.Status)

	_, err = order.Transition(domain.OrderStatusInProduction, "merchant", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusInProduction, order.Status)
}

func TestListAdminOrders_SortsByType(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	now := time.Now().UTC()

	for _, o := range []struct {
		ref  string
		typ  domain.OrderType
		time time.Time
	}{
		{"E25-CUSTOM", domain.OrderTypeCustomSize, now.Add(-1 * time.Hour)},
		{"E25-STANDARD", domain.OrderTypeStandard, now},
		{"E25-VISIT", domain.OrderTypeVisit, now.Add(-2 * time.Hour)},
	} {
		require.NoError(t, orders.Create(context.Background(), &domain.Order{
			Ref:            o.ref,
			CustomerID:     "user-1",
			DesignSnapshot: domain.DesignSnapshot{PricePesewas: 10000},
			Type:           o.typ,
			Status:         domain.OrderStatusPendingPayment,
			CreatedAt:      o.time,
			UpdatedAt:      o.time,
		}))
	}

	svc := newOrderService(
		orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{},
	)
	list, err := svc.ListAdminOrders(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 3)

	types := make([]string, 0, len(list))
	for _, o := range list {
		types = append(types, string(o.Type))
	}

	assert.Equal(t, []string{"custom_size", "standard", "visit"}, types)
}

func TestListCustomerOrders(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()

	svc := newOrderService(
		orders, designs, users, newFakePaymentProvider(),
		&fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{},
	)

	result, err := svc.CreateStandardOrder(t.Context(), "des-1", "8", "pickup",
		"+233200000000", "ama@example.com", "Ama")
	require.NoError(t, err)

	list, err := svc.ListCustomerOrders(t.Context(), result.User.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, result.Order.Ref, list[0].Ref)
}

// --- custom request tests -----------------------------------------------------

func TestCreateCustomRequest_SelfMeasure(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()
	payments := newFakePaymentProvider()
	events := &fakePaymentEvents{}
	sender := &recordingOrderSender{}
	settings := &fakeSettingsRepo{}

	svc := newOrderService(orders, designs, users, payments, events, sender, settings)

	order, err := svc.CreateCustomRequest(
		t.Context(),
		"des-1",
		domain.Customisation{
			SizeMode:     "self",
			Measurements: map[string]string{"bust": "90 cm", "waist": "74 cm"},
		},
		"pickup",
		"+233200000000",
		"ama@example.com",
		"Ama",
	)
	require.NoError(t, err)

	assert.Equal(t, domain.OrderTypeCustomSize, order.Type)
	assert.Equal(t, domain.OrderStatusRequested, order.Status)
	assert.Equal(t, "self", order.Customisation.SizeMode)
	assert.Equal(t, "90 cm", order.Customisation.Measurements["bust"])
	assert.Equal(t, int64(0), order.TotalPesewas())
	assert.NotEmpty(t, order.Ref)
	assert.Equal(t, "Ama", users.byEmail["ama@example.com"].Name)
}

func TestCreateCustomRequest_DesignChange(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()

	settings := &fakeSettingsRepo{settings: &domain.Settings{
		DeliveryRates: []domain.DeliveryRate{{Area: "East Legon", RatePesewas: 2000}},
	}}

	svc := newOrderService(
		orders, designs, users, newFakePaymentProvider(),
		&fakePaymentEvents{}, &recordingOrderSender{}, settings,
	)

	order, err := svc.CreateCustomRequest(
		t.Context(),
		"des-1",
		domain.Customisation{
			SizeMode:     "band",
			BandLabel:    "8",
			DesignChange: "sleeveless",
		},
		"dispatch:East Legon",
		"+233200000000",
		"ama@example.com",
		"Ama",
	)
	require.NoError(t, err)

	assert.Equal(t, domain.OrderTypeDesignChange, order.Type)
	assert.Equal(t, "sleeveless", order.Customisation.DesignChange)
	assert.Equal(t, "dispatch", order.Delivery.Mode)
	assert.Equal(t, "East Legon", order.Delivery.Area)
}

func TestCreateCustomRequest_Validation(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	designs := newFakeDesignRepo()
	designs.byID["des-1"] = liveDesign()
	users := newFakeUsers()

	svc := newOrderService(
		orders, designs, users, newFakePaymentProvider(),
		&fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{},
	)

	cases := []struct {
		name string
		call func() (*domain.Order, error)
	}{
		{"missing email", func() (*domain.Order, error) {
			return svc.CreateCustomRequest(t.Context(), "des-1",
				domain.Customisation{SizeMode: "self"}, "pickup", "+233200000000", "", "Ama")
		}},
		{"unknown design", func() (*domain.Order, error) {
			return svc.CreateCustomRequest(t.Context(), "missing",
				domain.Customisation{SizeMode: "self"}, "pickup", "+233200000000", "ama@example.com", "Ama")
		}},
		{"invalid size mode", func() (*domain.Order, error) {
			return svc.CreateCustomRequest(t.Context(), "des-1",
				domain.Customisation{SizeMode: "unknown"}, "pickup", "+233200000000", "ama@example.com", "Ama")
		}},
		{"band missing label", func() (*domain.Order, error) {
			return svc.CreateCustomRequest(t.Context(), "des-1",
				domain.Customisation{SizeMode: "band"}, "pickup", "+233200000000", "ama@example.com", "Ama")
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

// --- admin quote / payment / status tests -------------------------------------

func customRequestOrder(ref string) *domain.Order {
	createdAt := time.Now().UTC()

	order := &domain.Order{
		Ref:            ref,
		CustomerID:     "user-1",
		DesignID:       "des-1",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "e25/blazer", PricePesewas: 0},
		Type:           domain.OrderTypeDesignChange,
		Customisation:  domain.Customisation{SizeMode: "self", DesignChange: "longer sleeves"},
		Status:         domain.OrderStatusRequested,
		CustomerPhone:  "+233200000000",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	order.StatusHistory = []domain.StatusChange{{
		Status: domain.OrderStatusRequested,
		At:     createdAt,
		By:     "customer",
	}}

	return order
}

// bookedCustomOrder is a quoted custom order that was paid through a payment
// link, leaving it booked — the launch point for production-stage transitions.
func bookedCustomOrder(ref string) *domain.Order {
	order := customRequestOrder(ref)
	order.Quote = domain.Quote{PricePesewas: 60000, Timeline: "10 working days", Notes: ""}
	order.Status = domain.OrderStatusPaymentLinkSent

	_, _ = order.MarkPaid(
		domain.Payment{ProviderRef: "ps-1", AmountPesewas: 60000}, "payment_webhook", time.Now().UTC())

	return order
}

func TestUpdateQuote_RequestedBecomesQuoted(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-QUOTE")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateQuote(t.Context(), "E25-QUOTE", domain.Quote{
		PricePesewas: 75000,
		Timeline:     "2 weeks",
		Notes:        "Premium fabric",
	})
	require.NoError(t, err)

	loaded, err := orders.GetByRef(t.Context(), "E25-QUOTE")
	require.NoError(t, err)
	assert.Equal(t, int64(75000), loaded.Quote.PricePesewas)
	assert.Equal(t, domain.OrderStatusQuoted, loaded.Status)
}

func TestUpdateQuote_CanUpdateExistingQuote(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	order := customRequestOrder("E25-QUOTE")
	order.Status = domain.OrderStatusQuoted
	require.NoError(t, orders.Create(context.Background(), order))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateQuote(t.Context(), "E25-QUOTE", domain.Quote{
		PricePesewas: 80000,
		Timeline:     "10 days",
		Notes:        "Updated",
	})
	require.NoError(t, err)

	loaded, err := orders.GetByRef(t.Context(), "E25-QUOTE")
	require.NoError(t, err)
	assert.Equal(t, int64(80000), loaded.Quote.PricePesewas)
	assert.Equal(t, domain.OrderStatusQuoted, loaded.Status)
}

func TestUpdateQuote_RejectedAfterPaymentLinkSent(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	order := customRequestOrder("E25-QUOTE")
	order.Status = domain.OrderStatusPaymentLinkSent
	require.NoError(t, orders.Create(context.Background(), order))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateQuote(t.Context(), "E25-QUOTE", domain.Quote{PricePesewas: 10000})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestUpdateQuote_RejectedNegativePrice(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-QUOTE")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateQuote(t.Context(), "E25-QUOTE", domain.Quote{PricePesewas: -1})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestSendPaymentLink_CreatesPendingPayment(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-LINK")))

	users := newFakeUsers()
	_ = users.Upsert(context.Background(), &domain.User{
		ID:    "user-1",
		Email: "ama@example.com",
		Name:  "Ama",
		Role:  domain.RoleCustomer,
	})

	svc := newOrderService(orders, newFakeDesignRepo(), users,
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateQuote(t.Context(), "E25-LINK", domain.Quote{PricePesewas: 60000})
	require.NoError(t, err)

	url, err := svc.SendPaymentLink(t.Context(), "E25-LINK")
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.test/pay", url)

	loaded, err := orders.GetByRef(t.Context(), "E25-LINK")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPaymentLinkSent, loaded.Status)
	require.Len(t, loaded.Payments, 1)
	// The transaction reference is the order ref, so the success webhook
	// resolves straight back to this order.
	assert.Equal(t, "E25-LINK", loaded.Payments[0].ProviderRef)
	assert.Equal(t, int64(60000), loaded.Payments[0].AmountPesewas)
	assert.Equal(t, "pending", loaded.Payments[0].Status)
}

func TestSendPaymentLink_RejectedBeforeQuote(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-LINK")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	_, err := svc.SendPaymentLink(t.Context(), "E25-LINK")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

// TestPaymentLink_EndToEnd_BooksOrderOnWebhook drives the full payment-link
// round trip against the real Paystack adapter (stubbed transport): quote →
// link via /transaction/initialize with reference=order ref → signed
// charge.success webhook → order booked.
func TestPaymentLink_EndToEnd_BooksOrderOnWebhook(t *testing.T) {
	t.Parallel()

	var initRequest map[string]any

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transaction/initialize", r.URL.Path)

		err := json.NewDecoder(r.Body).Decode(&initRequest)
		if err != nil {
			t.Errorf("decode request: %v", err)

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  true,
			"message": "Initialized",
			"data": map[string]any{
				"authorization_url": "https://checkout.paystack.com/link-abc",
				"reference":         initRequest["reference"],
				"access_code":       "access_123",
			},
		})
	}))
	defer stub.Close()

	const secret = "sk_test_link"

	provider := paystack.NewClientWithBaseURL(secret, stub.URL)

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-E2E")))

	users := newFakeUsers()
	_ = users.Upsert(context.Background(), &domain.User{
		ID:    "user-1",
		Email: "ama@example.com",
		Name:  "Ama",
		Role:  domain.RoleCustomer,
	})

	sender := &recordingOrderSender{}
	logger := slog.New(slog.DiscardHandler)
	svc := service.NewOrder(orders, newFakeDesignRepo(), users, provider,
		&fakePaymentEvents{}, sender, &fakeSettingsRepo{}, "https://shop.test", logger)

	require.NoError(t, svc.UpdateQuote(t.Context(), "E25-E2E", domain.Quote{
		PricePesewas: 75000,
		Timeline:     "2 weeks",
		Notes:        "",
	}))

	linkURL, err := svc.SendPaymentLink(t.Context(), "E25-E2E")
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.paystack.com/link-abc", linkURL)
	assert.Equal(t, "E25-E2E", initRequest["reference"], "link must use the order ref as reference")
	assert.Equal(t, "ama@example.com", initRequest["email"], "link must carry the customer email")
	assert.EqualValues(t, 75000, initRequest["amount"])

	// Paystack confirms the charge, echoing the reference we initialized with.
	payload := []byte(`{"event":"charge.success","data":{"reference":"E25-E2E","status":"success","amount":75000}}`)
	require.NoError(t, svc.HandlePaymentWebhook(t.Context(), payload, signWebhook(t, secret, payload)))

	loaded, err := orders.GetByRef(t.Context(), "E25-E2E")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, loaded.Status)
	assert.True(t, loaded.IsPaid())
	require.Len(t, loaded.Payments, 1)
	assert.Equal(t, "E25-E2E", loaded.Payments[0].ProviderRef)
	assert.Equal(t, int64(75000), loaded.Payments[0].AmountPesewas)
	assert.Equal(t, 1, sender.statusUpdates)
}

func TestMarkPaidManually_BooksOrderAndSendsConfirmation(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	order := customRequestOrder("E25-MANUAL")
	order.Quote = domain.Quote{PricePesewas: 60000}
	order.Status = domain.OrderStatusPaymentLinkSent
	require.NoError(t, orders.Create(context.Background(), order))

	users := newFakeUsers()
	_ = users.Upsert(context.Background(), &domain.User{
		ID:    "user-1",
		Email: "ama@example.com",
		Name:  "Ama",
		Role:  domain.RoleCustomer,
	})

	sender := &recordingOrderSender{}
	svc := newOrderService(orders, newFakeDesignRepo(), users,
		newFakePaymentProvider(), &fakePaymentEvents{}, sender, &fakeSettingsRepo{})

	err := svc.MarkPaidManually(t.Context(), "E25-MANUAL", "Cash on pickup", "merchant@e25.com")
	require.NoError(t, err)

	loaded, err := orders.GetByRef(t.Context(), "E25-MANUAL")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, loaded.Status)
	assert.True(t, loaded.IsPaid())
	require.Len(t, loaded.Payments, 1)
	assert.Contains(t, loaded.Payments[0].Method, "manual")
	assert.Contains(t, loaded.Payments[0].Method, "Cash on pickup")

	// The booked transition must be attributed to the acting admin, not the
	// payment webhook, so the audit trail distinguishes manual overrides.
	lastChange := loaded.StatusHistory[len(loaded.StatusHistory)-1]
	assert.Equal(t, domain.OrderStatusBooked, lastChange.Status)
	assert.Equal(t, "merchant@e25.com", lastChange.By)
	assert.Equal(t, "ama@example.com", sender.lastTo)
	assert.Equal(t, "order confirmed", sender.lastStatus)
	assert.Equal(t, "roughly two weeks, depending on current bookings", sender.lastTimeframe)
}

func TestMarkPaidManually_RejectedZeroTotal(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-MANUAL")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.MarkPaidManually(t.Context(), "E25-MANUAL", "Cash", "merchant@e25.com")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestMarkPaidManually_RejectedUnquotedCustomWithDeliveryRate(t *testing.T) {
	t.Parallel()

	// An unquoted custom request whose dispatch area carries a delivery rate
	// has a positive total, but charging it pre-quote violates the scope.
	orders := newFakeOrderRepo()
	order := customRequestOrder("E25-RATE")
	rate := int64(2000)
	order.Delivery = domain.Delivery{Mode: "dispatch", Area: "East Legon", RatePesewas: &rate}
	require.NoError(t, orders.Create(context.Background(), order))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.MarkPaidManually(t.Context(), "E25-RATE", "Cash", "merchant@e25.com")
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = svc.SendPaymentLink(t.Context(), "E25-RATE")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestMarkPaidManually_RejectedTerminalAndPaidOrders(t *testing.T) {
	t.Parallel()

	for _, status := range []domain.OrderStatus{
		domain.OrderStatusCancelled,
		domain.OrderStatusFulfilled,
		domain.OrderStatusInProduction,
	} {
		orders := newFakeOrderRepo()
		order := customRequestOrder("E25-TERM")
		order.Quote = domain.Quote{PricePesewas: 60000, Timeline: "", Notes: ""}
		order.Status = status
		require.NoError(t, orders.Create(context.Background(), order))

		svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
			newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

		err := svc.MarkPaidManually(t.Context(), "E25-TERM", "Cash", "merchant@e25.com")
		require.ErrorIs(t, err, domain.ErrInvalidInput, "status %s", status)
	}
}

func TestUpdateOrderStatus_BlocksUnpaidProduction(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-STATUS")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-STATUS", domain.OrderStatusInProduction, "merchant@e25.com")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestUpdateOrderStatus_TransitionsPaidOrder(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), bookedCustomOrder("E25-STATUS")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-STATUS", domain.OrderStatusInProduction, "merchant@e25.com")
	require.NoError(t, err)

	loaded, err := orders.GetByRef(t.Context(), "E25-STATUS")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusInProduction, loaded.Status)
}

func TestUpdateOrderStatus_RejectsUnknownStatus(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), bookedCustomOrder("E25-UNKNOWN")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-UNKNOWN", domain.OrderStatus("paid_lol"), "merchant@e25.com")
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	loaded, err := orders.GetByRef(t.Context(), "E25-UNKNOWN")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, loaded.Status)
}

func TestUpdateOrderStatus_RejectsBookedTarget(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-NOBOOK")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-NOBOOK", domain.OrderStatusBooked, "merchant@e25.com")
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	loaded, err := orders.GetByRef(t.Context(), "E25-NOBOOK")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusRequested, loaded.Status)
	assert.False(t, loaded.IsPaid())
}

func TestUpdateOrderStatus_CannotSkipProductionStages(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), customRequestOrder("E25-SKIP")))

	svc := newOrderService(orders, newFakeDesignRepo(), newFakeUsers(),
		newFakePaymentProvider(), &fakePaymentEvents{}, &recordingOrderSender{}, &fakeSettingsRepo{})

	// An unpaid requested order cannot jump to ready or fulfilled, which would
	// bypass the paid-before-production gate entirely.
	for _, target := range []domain.OrderStatus{
		domain.OrderStatusReady,
		domain.OrderStatusFulfilled,
		domain.OrderStatusInProduction,
	} {
		err := svc.UpdateOrderStatus(t.Context(), "E25-SKIP", target, "merchant@e25.com")
		require.ErrorIs(t, err, domain.ErrInvalidInput, "target %s", target)
	}

	loaded, err := orders.GetByRef(t.Context(), "E25-SKIP")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusRequested, loaded.Status)
}

func TestUpdateOrderStatus_SendsEmailForInProduction(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	require.NoError(t, orders.Create(context.Background(), bookedCustomOrder("E25-STATUS")))

	users := newFakeUsers()
	_ = users.Upsert(context.Background(), &domain.User{
		ID:    "user-1",
		Email: "ama@example.com",
		Name:  "Ama",
		Role:  domain.RoleCustomer,
	})

	sender := &recordingOrderSender{}
	svc := newOrderService(orders, newFakeDesignRepo(), users,
		newFakePaymentProvider(), &fakePaymentEvents{}, sender, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-STATUS", domain.OrderStatusInProduction, "merchant@e25.com")
	require.NoError(t, err)

	assert.Equal(t, "ama@example.com", sender.lastTo)
	assert.Equal(t, "Ama", sender.lastName)
	assert.Equal(t, "E25-STATUS", sender.lastRef)
	assert.Equal(t, "in production", sender.lastStatus)
	assert.Equal(t, "10 working days", sender.lastTimeframe)
}

func TestUpdateOrderStatus_SendsEmailForReady(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	order := bookedCustomOrder("E25-READY")
	order.Quote.Timeline = "ready by Friday"
	_, _ = order.Transition(domain.OrderStatusInProduction, "merchant", time.Now().UTC())
	require.NoError(t, orders.Create(context.Background(), order))

	users := newFakeUsers()
	_ = users.Upsert(context.Background(), &domain.User{
		ID:    "user-1",
		Email: "ama@example.com",
		Name:  "Ama",
		Role:  domain.RoleCustomer,
	})

	sender := &recordingOrderSender{}
	svc := newOrderService(orders, newFakeDesignRepo(), users,
		newFakePaymentProvider(), &fakePaymentEvents{}, sender, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-READY", domain.OrderStatusReady, "merchant@e25.com")
	require.NoError(t, err)

	assert.Equal(t, "ama@example.com", sender.lastTo)
	assert.Equal(t, "ready", sender.lastStatus)
	assert.Equal(t, "ready by Friday", sender.lastTimeframe)
}

func TestUpdateOrderStatus_DoesNotSendEmailForOtherStatuses(t *testing.T) {
	t.Parallel()

	orders := newFakeOrderRepo()
	order := bookedCustomOrder("E25-OTHER")
	_, _ = order.Transition(domain.OrderStatusInProduction, "merchant", time.Now().UTC())
	_, _ = order.Transition(domain.OrderStatusReady, "merchant", time.Now().UTC())
	require.NoError(t, orders.Create(context.Background(), order))

	users := newFakeUsers()
	_ = users.Upsert(context.Background(), &domain.User{
		ID:    "user-1",
		Email: "ama@example.com",
		Name:  "Ama",
		Role:  domain.RoleCustomer,
	})

	sender := &recordingOrderSender{}
	svc := newOrderService(orders, newFakeDesignRepo(), users,
		newFakePaymentProvider(), &fakePaymentEvents{}, sender, &fakeSettingsRepo{})

	err := svc.UpdateOrderStatus(t.Context(), "E25-OTHER", domain.OrderStatusFulfilled, "merchant@e25.com")
	require.NoError(t, err)

	assert.Empty(t, sender.lastTo)
	assert.Empty(t, sender.lastStatus)
}
