package httpapi_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/transport/httpapi"
)

// --- in-memory repositories -------------------------------------------------

type memRepo struct {
	byEmail map[string]domain.Subscriber
}

func newMemRepo() *memRepo { return &memRepo{byEmail: map[string]domain.Subscriber{}} }

// pageSlice returns the window of items for a normalized page request.
func pageSlice[T any](items []T, params domain.PageParams) []T {
	skip := int(params.Skip())
	if skip >= len(items) {
		return []T{}
	}

	end := min(skip+params.PageSize, len(items))

	return items[skip:end]
}

func (m *memRepo) Create(_ context.Context, s *domain.Subscriber) error {
	if _, exists := m.byEmail[s.Email]; exists {
		return domain.ErrDuplicateEmail
	}

	s.ID = "id-" + s.Email
	m.byEmail[s.Email] = *s

	return nil
}

func (m *memRepo) List(_ context.Context, _ int64) ([]domain.Subscriber, error) {
	out := make([]domain.Subscriber, 0, len(m.byEmail))
	for _, s := range m.byEmail {
		out = append(out, s)
	}

	return out, nil
}

func (m *memRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.byEmail)), nil
}

func (m *memRepo) ListPaged(_ context.Context, params domain.PageParams) ([]domain.Subscriber, error) {
	out := make([]domain.Subscriber, 0, len(m.byEmail))
	for _, s := range m.byEmail {
		out = append(out, s)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })

	return pageSlice(out, params), nil
}

func (m *memRepo) Delete(_ context.Context, id string) error {
	for email, s := range m.byEmail {
		if s.ID == id {
			delete(m.byEmail, email)

			return nil
		}
	}

	return domain.ErrNotFound
}

type memUsers struct {
	byEmail map[string]*domain.User
	nextID  int
}

func newMemUsers() *memUsers { return &memUsers{byEmail: map[string]*domain.User{}, nextID: 1} }

func (m *memUsers) Upsert(_ context.Context, u *domain.User) error {
	existing, ok := m.byEmail[u.Email]
	if ok {
		if u.Role == domain.RoleAdmin {
			existing.Role = domain.RoleAdmin
		}

		*u = *existing

		return nil
	}

	u.ID = "user-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *u
	m.byEmail[u.Email] = &clone

	return nil
}

func (m *memUsers) GetByID(_ context.Context, id string) (*domain.User, error) {
	for _, u := range m.byEmail {
		if u.ID == id {
			clone := *u

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (m *memUsers) Count(_ context.Context) (int64, error) {
	return int64(len(m.byEmail)), nil
}

func (m *memUsers) ListPaged(_ context.Context, params domain.PageParams) ([]domain.User, error) {
	all := make([]domain.User, 0, len(m.byEmail))
	for _, u := range m.byEmail {
		all = append(all, *u)
	}

	skip := int(params.Skip())
	if skip >= len(all) {
		return []domain.User{}, nil
	}

	end := min(skip+int(params.Limit()), len(all))

	return all[skip:end], nil
}

func (m *memUsers) UpdateRole(_ context.Context, id string, role domain.Role) error {
	for _, u := range m.byEmail {
		if u.ID == id {
			u.Role = role

			return nil
		}
	}

	return domain.ErrNotFound
}

type tokenRecord struct {
	userID    string
	expiresAt time.Time
	used      bool
}

type memTokens struct {
	logins   map[string]*tokenRecord
	sessions map[string]*tokenRecord
}

func newMemTokens() *memTokens {
	return &memTokens{logins: map[string]*tokenRecord{}, sessions: map[string]*tokenRecord{}}
}

func (m *memTokens) StoreLoginToken(_ context.Context, hash, userID string, expiresAt time.Time) error {
	m.logins[hash] = &tokenRecord{userID: userID, expiresAt: expiresAt, used: false}

	return nil
}

func (m *memTokens) ConsumeLoginToken(_ context.Context, hash string) (string, error) {
	rec, ok := m.logins[hash]
	if !ok || rec.used || rec.expiresAt.Before(time.Now()) {
		return "", domain.ErrTokenInvalid
	}

	rec.used = true

	return rec.userID, nil
}

func (m *memTokens) CreateSession(_ context.Context, hash, userID string, expiresAt time.Time) error {
	m.sessions[hash] = &tokenRecord{userID: userID, expiresAt: expiresAt, used: false}

	return nil
}

func (m *memTokens) GetSession(_ context.Context, hash string) (string, error) {
	rec, ok := m.sessions[hash]
	if !ok || rec.expiresAt.Before(time.Now()) {
		return "", domain.ErrTokenInvalid
	}

	return rec.userID, nil
}

func (m *memTokens) DeleteSession(_ context.Context, hash string) error {
	delete(m.sessions, hash)

	return nil
}

type memSettings struct {
	saved *domain.Settings
}

func (m *memSettings) Get(_ context.Context) (*domain.Settings, error) {
	if m.saved == nil {
		return domain.DefaultSettings(), nil
	}

	clone := *m.saved

	return &clone, nil
}

func (m *memSettings) Update(_ context.Context, s *domain.Settings) error {
	clone := *s
	m.saved = &clone

	return nil
}

// --- in-memory catalog repositories ------------------------------------------

type memCollections struct {
	byID   map[string]*domain.Collection
	nextID int
}

func newMemCollections() *memCollections {
	return &memCollections{byID: map[string]*domain.Collection{}, nextID: 1}
}

func (m *memCollections) Create(_ context.Context, c *domain.Collection) error {
	for _, existing := range m.byID {
		if existing.Slug == c.Slug {
			return domain.ErrDuplicateSlug
		}
	}

	c.ID = "col-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *c
	m.byID[c.ID] = &clone

	return nil
}

func (m *memCollections) Update(_ context.Context, id, name, note string) error {
	collection, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}

	collection.Name = name
	collection.Note = note

	return nil
}

func (m *memCollections) GetByID(_ context.Context, id string) (*domain.Collection, error) {
	collection, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}

	clone := *collection

	return &clone, nil
}

func (m *memCollections) GetBySlug(_ context.Context, slug string) (*domain.Collection, error) {
	for _, collection := range m.byID {
		if collection.Slug == slug {
			clone := *collection

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (m *memCollections) List(_ context.Context, includeRetired bool) ([]domain.Collection, error) {
	out := make([]domain.Collection, 0, len(m.byID))

	for _, collection := range m.byID {
		if !includeRetired && collection.Status != domain.StatusLive {
			continue
		}

		out = append(out, *collection)
	}

	return out, nil
}

func (m *memCollections) Count(ctx context.Context, includeRetired bool) (int64, error) {
	out, err := m.List(ctx, includeRetired)

	return int64(len(out)), err
}

func (m *memCollections) ListPaged(
	ctx context.Context, includeRetired bool, params domain.PageParams,
) ([]domain.Collection, error) {
	out, err := m.List(ctx, includeRetired)
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })

	return pageSlice(out, params), nil
}

func (m *memCollections) SetStatus(_ context.Context, id string, status domain.Status, at time.Time) error {
	collection, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}

	collection.Status = status

	collection.RetiredAt = nil
	if status == domain.StatusRetired {
		collection.RetiredAt = &at
	}

	return nil
}

func (m *memCollections) Delete(_ context.Context, id string) error {
	delete(m.byID, id)

	return nil
}

type memDesigns struct {
	byID   map[string]*domain.Design
	nextID int
}

func newMemDesigns() *memDesigns { return &memDesigns{byID: map[string]*domain.Design{}, nextID: 1} }

func (m *memDesigns) Create(_ context.Context, d *domain.Design) error {
	for _, existing := range m.byID {
		if existing.Slug == d.Slug {
			return domain.ErrDuplicateSlug
		}
	}

	d.ID = "des-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *d
	m.byID[d.ID] = &clone

	return nil
}

func (m *memDesigns) Update(_ context.Context, d *domain.Design) error {
	if _, ok := m.byID[d.ID]; !ok {
		return domain.ErrNotFound
	}

	clone := *d
	m.byID[d.ID] = &clone

	return nil
}

func (m *memDesigns) GetByID(_ context.Context, id string) (*domain.Design, error) {
	design, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}

	clone := *design

	return &clone, nil
}

func (m *memDesigns) GetBySlug(_ context.Context, slug string) (*domain.Design, error) {
	for _, design := range m.byID {
		if design.Slug == slug {
			clone := *design

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (m *memDesigns) List(_ context.Context, filter domain.DesignFilter) ([]domain.Design, error) {
	out := make([]domain.Design, 0, len(m.byID))

	for _, design := range m.byID {
		if !filter.IncludeRetired && design.Status != domain.StatusLive {
			continue
		}

		if filter.CollectionID != "" && design.CollectionID != filter.CollectionID {
			continue
		}

		if filter.Query != "" && !strings.Contains(strings.ToLower(design.Name), strings.ToLower(filter.Query)) {
			continue
		}

		out = append(out, *design)
	}

	return out, nil
}

func (m *memDesigns) Count(ctx context.Context, filter domain.DesignFilter) (int64, error) {
	out, err := m.List(ctx, filter)

	return int64(len(out)), err
}

func (m *memDesigns) ListPaged(
	ctx context.Context, filter domain.DesignFilter, params domain.PageParams,
) ([]domain.Design, error) {
	out, err := m.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })

	return pageSlice(out, params), nil
}

func (m *memDesigns) SetStatusBulk(_ context.Context, ids []string, status domain.Status, at time.Time) error {
	for _, id := range ids {
		design, ok := m.byID[id]
		if !ok {
			return domain.ErrNotFound
		}

		design.Status = status

		design.RetiredAt = nil
		if status == domain.StatusRetired {
			design.RetiredAt = &at
		}
	}

	return nil
}

func (m *memDesigns) SetStatusByCollection(
	ctx context.Context, collectionID string, status domain.Status, at time.Time,
) error {
	for id, design := range m.byID {
		if design.CollectionID == collectionID {
			err := m.SetStatusBulk(ctx, []string{id}, status, at)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *memDesigns) Delete(_ context.Context, id string) error {
	delete(m.byID, id)

	return nil
}

func (m *memDesigns) DeleteByCollection(_ context.Context, collectionID string) error {
	for id, design := range m.byID {
		if design.CollectionID == collectionID {
			delete(m.byID, id)
		}
	}

	return nil
}

// --- in-memory order repository -----------------------------------------------

type memOrders struct {
	byID   map[string]*domain.Order
	byRef  map[string]*domain.Order
	nextID int
}

func newMemOrders() *memOrders {
	return &memOrders{byID: map[string]*domain.Order{}, byRef: map[string]*domain.Order{}, nextID: 1}
}

func (m *memOrders) Create(_ context.Context, o *domain.Order) error {
	if _, exists := m.byRef[o.Ref]; exists {
		return domain.ErrDuplicateRef
	}

	o.ID = "ord-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *o
	m.byID[o.ID] = &clone
	m.byRef[o.Ref] = &clone

	return nil
}

func (m *memOrders) Update(_ context.Context, o *domain.Order) error {
	m.byID[o.ID] = o
	m.byRef[o.Ref] = o

	return nil
}

func (m *memOrders) GetByID(_ context.Context, id string) (*domain.Order, error) {
	if o, ok := m.byID[id]; ok {
		clone := *o

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (m *memOrders) GetByRef(_ context.Context, ref string) (*domain.Order, error) {
	if o, ok := m.byRef[ref]; ok {
		clone := *o

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (m *memOrders) ListByCustomer(_ context.Context, customerID string) ([]domain.Order, error) {
	out := make([]domain.Order, 0, len(m.byID))
	for _, o := range m.byID {
		if o.CustomerID == customerID {
			clone := *o
			out = append(out, clone)
		}
	}

	return out, nil
}

func (m *memOrders) List(_ context.Context, _ domain.OrderFilter) ([]domain.Order, error) {
	out := make([]domain.Order, 0, len(m.byID))
	for _, o := range m.byID {
		clone := *o
		out = append(out, clone)
	}

	return out, nil
}

func (m *memOrders) Count(ctx context.Context, filter domain.OrderFilter) (int64, error) {
	out, err := m.List(ctx, filter)

	return int64(len(out)), err
}

func (m *memOrders) ListPaged(
	ctx context.Context, filter domain.OrderFilter, params domain.PageParams,
) ([]domain.Order, error) {
	out, err := m.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Mirror the real repo: group by type, newest first within each type.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}

		return out[i].CreatedAt.After(out[j].CreatedAt)
	})

	return pageSlice(out, params), nil
}

// --- fake payment provider ----------------------------------------------------

type fakePaymentProvider struct {
	secret  string
	authURL string
	linkURL string
}

func newFakePaymentProvider(secret string) *fakePaymentProvider {
	return &fakePaymentProvider{secret: secret, authURL: "https://checkout.test/pay", linkURL: "https://pay.test/link"}
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
	return "success", nil
}

type memPaymentEvents struct{}

func (m *memPaymentEvents) RecordEvent(context.Context, domain.PaymentEvent) error { return nil }

type fakeAnalyticsRepo struct {
	analytics *domain.StoreAnalytics
}

func (f *fakeAnalyticsRepo) GetStoreAnalytics(context.Context) (*domain.StoreAnalytics, error) {
	return f.analytics, nil
}

func defaultFakeAnalytics() *domain.StoreAnalytics {
	return &domain.StoreAnalytics{
		WaitlistCount: 3,
		CustomerCount: 5,
		OrdersByStatus: map[string]int64{
			string(domain.OrderStatusBooked):    2,
			string(domain.OrderStatusRequested): 1,
		},
		OrdersByType: map[string]int64{
			string(domain.OrderTypeStandard):   2,
			string(domain.OrderTypeCustomSize): 1,
		},
		RevenuePesewas:  250_000,
		CollectionViews: 0,
	}
}

// --- in-memory slot and visit repositories ------------------------------------

type memSlots struct {
	byID   map[string]*domain.Slot
	nextID int
}

func newMemSlots() *memSlots { return &memSlots{byID: map[string]*domain.Slot{}, nextID: 1} }

func (m *memSlots) Create(_ context.Context, s *domain.Slot) error {
	s.ID = "slot-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *s
	m.byID[s.ID] = &clone

	return nil
}

func (m *memSlots) GetByID(_ context.Context, id string) (*domain.Slot, error) {
	if s, ok := m.byID[id]; ok {
		clone := *s

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (m *memSlots) List(_ context.Context, filter domain.SlotFilter) ([]domain.Slot, error) {
	out := make([]domain.Slot, 0, len(m.byID))
	for _, s := range m.byID {
		if filter.Status != "" && s.Status != filter.Status {
			continue
		}

		clone := *s
		out = append(out, clone)
	}

	return out, nil
}

func (m *memSlots) Overlaps(_ context.Context, start, end time.Time) (bool, error) {
	for _, s := range m.byID {
		if s.Status == domain.SlotStatusClosed {
			continue
		}

		if s.Start.Before(end) && s.End.After(start) {
			return true, nil
		}
	}

	return false, nil
}

func (m *memSlots) UpdateStatusFrom(_ context.Context, id string, from, to domain.SlotStatus) error {
	s, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}

	if s.Status != from {
		return domain.ErrSlotUnavailable
	}

	s.Status = to

	return nil
}

type memVisits struct {
	byID     map[string]*domain.Visit
	bySlotID map[string]*domain.Visit
	nextID   int
}

func newMemVisits() *memVisits {
	return &memVisits{byID: map[string]*domain.Visit{}, bySlotID: map[string]*domain.Visit{}, nextID: 1}
}

func (m *memVisits) Create(_ context.Context, v *domain.Visit) error {
	v.ID = "visit-" + strconv.Itoa(m.nextID)
	m.nextID++
	clone := *v
	m.byID[v.ID] = &clone
	m.bySlotID[v.SlotID] = &clone

	return nil
}

func (m *memVisits) BookSlot(ctx context.Context, slotID string, v *domain.Visit) error {
	if existing, ok := m.bySlotID[slotID]; ok && existing.Status != domain.VisitStatusCancelled {
		return domain.ErrSlotUnavailable
	}

	return m.Create(ctx, v)
}

func (m *memVisits) GetByID(_ context.Context, id string) (*domain.Visit, error) {
	if v, ok := m.byID[id]; ok {
		clone := *v

		return &clone, nil
	}

	return nil, domain.ErrNotFound
}

func (m *memVisits) GetByOrderID(_ context.Context, orderID string) (*domain.Visit, error) {
	for _, v := range m.byID {
		if v.OrderID == orderID {
			clone := *v

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (m *memVisits) List(_ context.Context, _ domain.VisitFilter) ([]domain.Visit, error) {
	out := make([]domain.Visit, 0, len(m.byID))
	for _, v := range m.byID {
		clone := *v
		out = append(out, clone)
	}

	return out, nil
}

func (m *memVisits) ListExpiredHolds(_ context.Context, now time.Time) ([]domain.Visit, error) {
	out := make([]domain.Visit, 0, len(m.byID))

	for _, v := range m.byID {
		if v.Status != domain.VisitStatusBooked || v.HoldExpiresAt == nil || v.HoldExpiresAt.After(now) {
			continue
		}

		clone := *v
		out = append(out, clone)
	}

	return out, nil
}

func (m *memVisits) Update(_ context.Context, visit *domain.Visit) error {
	if _, ok := m.byID[visit.ID]; !ok {
		return domain.ErrNotFound
	}

	clone := *visit
	m.byID[visit.ID] = &clone
	m.bySlotID[visit.SlotID] = &clone

	return nil
}

// --- recording email sender ---------------------------------------------------

type memRoles struct {
	mu    sync.RWMutex
	byKey map[string]domain.RoleDef
}

func newMemRoles() *memRoles {
	m := &memRoles{byKey: map[string]domain.RoleDef{}}
	for _, def := range domain.BuiltInRoles() {
		m.byKey[def.Key] = def
	}

	return m
}

func (m *memRoles) List(_ context.Context) ([]domain.RoleDef, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]domain.RoleDef, 0, len(m.byKey))
	for _, role := range m.byKey {
		out = append(out, role)
	}

	return out, nil
}

func (m *memRoles) Get(_ context.Context, key string) (*domain.RoleDef, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, ok := m.byKey[key]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return &role, nil
}

func (m *memRoles) Upsert(_ context.Context, role *domain.RoleDef) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.byKey[role.Key] = *role

	return nil
}

func (m *memRoles) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.byKey, key)

	return nil
}

type recordingSender struct {
	lastLink           string
	lastStatusUpdateTo string
	lastStatus         string
	lastTimeframe      string
	loginErr           error // when set, SendLoginLink fails with it
}

func (r *recordingSender) SendWelcome(context.Context, string, string) error { return nil }

func (r *recordingSender) SendLoginLink(_ context.Context, _, link string) error {
	if r.loginErr != nil {
		return r.loginErr
	}

	r.lastLink = link

	return nil
}

func (r *recordingSender) SendOrderConfirmation(context.Context, string, string, string, string) error {
	return nil
}

func (r *recordingSender) SendOrderStatusUpdate(
	_ context.Context, to, _, _, status, timeframe string,
) error {
	r.lastStatusUpdateTo = to
	r.lastStatus = status
	r.lastTimeframe = timeframe

	return nil
}

// tokenFromLink extracts the raw token from a sign-in link.
func (r *recordingSender) tokenFromLink(t *testing.T) string {
	t.Helper()

	_, token, found := strings.Cut(r.lastLink, "token=")
	require.True(t, found, "no token in link %q", r.lastLink)

	return token
}

const sessionCookieName = "e25_session"

// --- test server ----------------------------------------------------------------

type testEnv struct {
	srv             *httptest.Server
	sender          *recordingSender
	orders          *memOrders
	paymentProvider *fakePaymentProvider
	roleStore       *memRoles // editable role store, to prove DB-driven enforcement
}

func newTestEnv(t *testing.T, adminEmails ...string) *testEnv {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	sender := &recordingSender{lastLink: ""}
	users := newMemUsers()
	tokens := newMemTokens()
	roleStore := newMemRoles()
	waitlist := service.NewWaitlist(newMemRepo(), sender, logger)
	auth := service.NewAuth(users, tokens, roleStore, sender, logger, "http://test.local", adminEmails)
	settings := service.NewStoreSettings(&memSettings{saved: nil})
	collections := newMemCollections()
	designs := newMemDesigns()
	catalog := service.NewCatalog(collections, designs)
	orders := newMemOrders()
	paymentProvider := newFakePaymentProvider("secret")
	orderService := newTestOrderService(
		orders, designs, users, paymentProvider, sender, settings, "http://test.local", logger,
	)
	analyticsService := service.NewAnalytics(&fakeAnalyticsRepo{analytics: defaultFakeAnalytics()})
	slots := newMemSlots()
	visitsRepo := newMemVisits()
	slotService := service.NewCalendarSlot(slots)
	visitService := service.NewCalendarVisit(
		slots, visitsRepo, orders, designs, users, paymentProvider, settings, sender, "http://test.local", logger,
	)
	handlers := httpapi.NewHandlers(
		waitlist, auth, settings, catalog, orderService, analyticsService,
		service.NewRoles(roleStore), slotService, visitService, nil, "test-cloud", false, nil,
	)
	srv := httptest.NewServer(httpapi.NewRouter(handlers, logger, []string{"*"}))
	t.Cleanup(srv.Close)

	return &testEnv{
		srv: srv, sender: sender, orders: orders, paymentProvider: paymentProvider, roleStore: roleStore,
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newTestEnv(t).srv
}

// newHandlersWithSigner rebuilds the handlers from env using the supplied signer.
func newHandlersWithSigner(env *testEnv, signer domain.UploadSigner) *httpapi.Handlers {
	users := newMemUsers()
	tokens := newMemTokens()
	roleStore := newMemRoles()
	collections := newMemCollections()
	designs := newMemDesigns()
	catalog := service.NewCatalog(collections, designs)
	orders := newMemOrders()
	paymentProvider := newFakePaymentProvider("secret")
	settings := service.NewStoreSettings(&memSettings{saved: nil})
	orderService := newTestOrderService(
		orders, designs, users, paymentProvider, env.sender, settings, "http://test.local", slog.New(slog.DiscardHandler),
	)
	analyticsService := service.NewAnalytics(&fakeAnalyticsRepo{analytics: defaultFakeAnalytics()})
	slots := newMemSlots()
	visitsRepo := newMemVisits()
	slotService := service.NewCalendarSlot(slots)
	visitService := service.NewCalendarVisit(
		slots, visitsRepo, orders, designs, users, paymentProvider, settings, env.sender,
		"http://test.local", slog.New(slog.DiscardHandler),
	)

	return httpapi.NewHandlers(
		service.NewWaitlist(newMemRepo(), env.sender, slog.New(slog.DiscardHandler)),
		service.NewAuth(users, tokens, roleStore, env.sender, slog.New(slog.DiscardHandler),
			"http://test.local", []string{"boss@e25.com"}),
		settings,
		catalog,
		orderService,
		analyticsService,
		service.NewRoles(roleStore),
		slotService,
		visitService,
		signer,
		"test-cloud",
		false,
		nil,
	)
}

func newTestOrderService(
	orders domain.OrderRepository,
	designs domain.DesignRepository,
	users domain.UserRepository,
	payments domain.PaymentProvider,
	sender domain.EmailSender,
	settings domain.SettingsRepository,
	webURL string,
	logger *slog.Logger,
) *service.Order {
	return service.NewOrder(orders, designs, users, payments, &memPaymentEvents{}, sender, settings, webURL, logger)
}

func newTestServerWithHandlers(t *testing.T, handlers *httpapi.Handlers) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(httpapi.NewRouter(handlers, slog.New(slog.DiscardHandler), []string{"*"}))
	t.Cleanup(srv.Close)

	return srv
}

func signInOnServer(t *testing.T, srv *httptest.Server, sender *recordingSender) *http.Cookie {
	t.Helper()

	status := postJSON(t, srv.URL+"/api/v1/auth/request-link", `{"email":"boss@e25.com","name":"Boss"}`)
	require.Equal(t, http.StatusAccepted, status)

	reply := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/verify",
		`{"token":"`+sender.tokenFromLink(t)+`"}`, nil)
	require.Equal(t, http.StatusOK, reply.status)

	for _, cookie := range reply.cookies {
		if cookie.Name == sessionCookieName && cookie.Value != "" {
			return cookie
		}
	}

	t.Fatal("no session cookie in verify response")

	return nil
}

// signIn runs the full link flow and returns the session cookie.
func (e *testEnv) signIn(t *testing.T, email string) *http.Cookie {
	t.Helper()

	status := postJSON(t, e.srv.URL+"/api/v1/auth/request-link", `{"email":"`+email+`","name":"Test"}`)
	require.Equal(t, http.StatusAccepted, status)

	reply := doJSON(t, http.MethodPost, e.srv.URL+"/api/v1/auth/verify",
		`{"token":"`+e.sender.tokenFromLink(t)+`"}`, nil)
	require.Equal(t, http.StatusOK, reply.status)

	for _, cookie := range reply.cookies {
		if cookie.Name == sessionCookieName && cookie.Value != "" {
			return cookie
		}
	}

	t.Fatal("no session cookie in verify response")

	return nil
}

type jsonReply struct {
	status  int
	body    string
	cookies []*http.Cookie
}

// doJSON issues a request with an optional session cookie; the body is read
// and closed before returning.
func doJSON(t *testing.T, method, url, body string, cookie *http.Cookie) jsonReply {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	if cookie != nil {
		req.AddCookie(cookie)
	}

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = res.Body.Close() }()

	payload, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return jsonReply{status: res.StatusCode, body: string(payload), cookies: res.Cookies()}
}
