package service_test

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type fakeUsers struct {
	byEmail map[string]*domain.User
	nextID  int
}

func newFakeUsers() *fakeUsers { return &fakeUsers{byEmail: map[string]*domain.User{}, nextID: 1} }

func (f *fakeUsers) Upsert(_ context.Context, u *domain.User) error {
	existing, ok := f.byEmail[u.Email]
	if ok {
		if u.Role == domain.RoleAdmin {
			existing.Role = domain.RoleAdmin
		}

		*u = *existing

		return nil
	}

	u.ID = "user-" + strconv.Itoa(f.nextID)
	f.nextID++
	clone := *u
	f.byEmail[u.Email] = &clone

	return nil
}

func (f *fakeUsers) GetByID(_ context.Context, id string) (*domain.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			clone := *u

			return &clone, nil
		}
	}

	return nil, domain.ErrNotFound
}

func TestSetUserRole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	users := newFakeUsers()
	auth := newAuth(users, newFakeTokens(), &linkSender{}, "boss@example.com")

	staffer := &domain.User{Email: "staffer@example.com", Name: "Staffer", Role: domain.RoleCustomer}
	require.NoError(t, users.Upsert(ctx, staffer))

	updated, err := auth.SetUserRole(ctx, staffer.ID, domain.RoleStaff)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleStaff, updated.Role)

	reloaded, err := users.GetByID(ctx, staffer.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleStaff, reloaded.Role)

	boss := &domain.User{Email: "boss@example.com", Name: "Boss", Role: domain.RoleAdmin}
	require.NoError(t, users.Upsert(ctx, boss))

	_, err = auth.SetUserRole(ctx, boss.ID, domain.RoleCustomer)
	require.ErrorIs(t, err, domain.ErrInvalidInput, "bootstrap super-admin must not be demotable")

	_, err = auth.SetUserRole(ctx, staffer.ID, domain.Role("wizard"))
	require.ErrorIs(t, err, domain.ErrInvalidInput, "unknown role rejected")
}

func TestInvitePartner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	users := newFakeUsers()
	sender := &linkSender{}
	auth := newAuth(users, newFakeTokens(), sender, "boss@example.com")

	// Invite a brand-new partner as staff: account created, link emailed.
	user, err := auth.InvitePartner(ctx, "Partner@Example.com", "Partner", domain.RoleStaff)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleStaff, user.Role)
	assert.Equal(t, "partner@example.com", user.Email, "email is normalised")
	assert.NotEmpty(t, sender.link, "a sign-in link is emailed")

	reloaded, err := users.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleStaff, reloaded.Role, "the invited role is persisted")

	// An unknown role is rejected.
	_, err = auth.InvitePartner(ctx, "ghost@example.com", "Ghost", domain.Role("wizard"))
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	// A non-dashboard role (customer) cannot be invited as a partner.
	_, err = auth.InvitePartner(ctx, "shopper@example.com", "Shopper", domain.RoleCustomer)
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func (f *fakeUsers) Count(_ context.Context) (int64, error) {
	return int64(len(f.byEmail)), nil
}

func (f *fakeUsers) ListPaged(_ context.Context, params domain.PageParams) ([]domain.User, error) {
	all := make([]domain.User, 0, len(f.byEmail))
	for _, u := range f.byEmail {
		all = append(all, *u)
	}

	skip := int(params.Skip())
	if skip >= len(all) {
		return []domain.User{}, nil
	}

	end := min(skip+int(params.Limit()), len(all))

	return all[skip:end], nil
}

func (f *fakeUsers) UpdateRole(_ context.Context, id string, role domain.Role) error {
	for _, u := range f.byEmail {
		if u.ID == id {
			u.Role = role

			return nil
		}
	}

	return domain.ErrNotFound
}

type fakeTokenRecord struct {
	userID    string
	expiresAt time.Time
	used      bool
}

type fakeTokens struct {
	logins   map[string]*fakeTokenRecord
	sessions map[string]*fakeTokenRecord
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{logins: map[string]*fakeTokenRecord{}, sessions: map[string]*fakeTokenRecord{}}
}

func (f *fakeTokens) StoreLoginToken(_ context.Context, hash, userID string, expiresAt time.Time) error {
	f.logins[hash] = &fakeTokenRecord{userID: userID, expiresAt: expiresAt, used: false}

	return nil
}

func (f *fakeTokens) ConsumeLoginToken(_ context.Context, hash string) (string, error) {
	rec, ok := f.logins[hash]
	if !ok || rec.used || rec.expiresAt.Before(time.Now()) {
		return "", domain.ErrTokenInvalid
	}

	rec.used = true

	return rec.userID, nil
}

func (f *fakeTokens) CreateSession(_ context.Context, hash, userID string, expiresAt time.Time) error {
	f.sessions[hash] = &fakeTokenRecord{userID: userID, expiresAt: expiresAt, used: false}

	return nil
}

func (f *fakeTokens) GetSession(_ context.Context, hash string) (string, error) {
	rec, ok := f.sessions[hash]
	if !ok {
		return "", domain.ErrTokenInvalid
	}

	return rec.userID, nil
}

func (f *fakeTokens) DeleteSession(_ context.Context, hash string) error {
	delete(f.sessions, hash)

	return nil
}

type linkSender struct {
	to      string
	link    string
	failErr error // when set, SendLoginLink returns it instead of recording
}

func (l *linkSender) SendWelcome(context.Context, string, string) error { return nil }

func (l *linkSender) SendOrderConfirmation(context.Context, string, string, string, string) error {
	return nil
}

func (l *linkSender) SendOrderStatusUpdate(context.Context, string, string, string, string, string) error {
	return nil
}

func (l *linkSender) SendLoginLink(_ context.Context, to, link string) error {
	if l.failErr != nil {
		return l.failErr
	}

	l.to = to
	l.link = link

	return nil
}

func (l *linkSender) token(t *testing.T) string {
	t.Helper()

	_, token, found := strings.Cut(l.link, "token=")
	require.True(t, found)

	return token
}

// fakeRoles is a minimal in-memory domain.RoleRepository seeded with the
// built-in roles, so SetUserRole can validate role keys against the store.
type fakeRoles struct {
	byKey map[string]domain.RoleDef
}

func newFakeRoles() *fakeRoles {
	m := &fakeRoles{byKey: map[string]domain.RoleDef{}}
	for _, def := range domain.BuiltInRoles() {
		m.byKey[def.Key] = def
	}

	return m
}

func (f *fakeRoles) List(_ context.Context) ([]domain.RoleDef, error) {
	out := make([]domain.RoleDef, 0, len(f.byKey))
	for _, role := range f.byKey {
		out = append(out, role)
	}

	return out, nil
}

func (f *fakeRoles) Get(_ context.Context, key string) (*domain.RoleDef, error) {
	role, ok := f.byKey[key]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return &role, nil
}

func (f *fakeRoles) Upsert(_ context.Context, role *domain.RoleDef) error {
	f.byKey[role.Key] = *role

	return nil
}

func (f *fakeRoles) Delete(_ context.Context, key string) error {
	delete(f.byKey, key)

	return nil
}

func newAuth(users *fakeUsers, tokens *fakeTokens, sender *linkSender, adminEmails ...string) *service.Auth {
	return service.NewAuth(users, tokens, newFakeRoles(), sender, slog.New(slog.DiscardHandler),
		"https://shop.test/", adminEmails)
}

func TestRequestLink_CreatesUserAndSendsLink(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()
	tokens := newFakeTokens()
	sender := &linkSender{to: "", link: ""}

	err := newAuth(users, tokens, sender).RequestLink(t.Context(), " Ama@Example.COM ", " Ama ")
	require.NoError(t, err)

	user := users.byEmail["ama@example.com"]
	require.NotNil(t, user, "user must be upserted with normalized email")
	assert.Equal(t, domain.RoleCustomer, user.Role)
	assert.Equal(t, "Ama", user.Name)

	assert.Equal(t, "ama@example.com", sender.to)
	assert.True(t, strings.HasPrefix(sender.link, "https://shop.test/auth/verify?token="),
		"link %q must point at the web verify page", sender.link)
	assert.Len(t, tokens.logins, 1, "one hashed login token stored")
}

func TestRequestLink_EmailSendFailure(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()
	tokens := newFakeTokens()
	errSendDown := errors.New("resend 503")
	sender := &linkSender{failErr: errSendDown}

	err := newAuth(users, tokens, sender).RequestLink(t.Context(), "ama@example.com", "Ama")

	// The failure is reported as a send problem (so the API can tell the customer
	// to retry) while keeping the underlying cause reachable through the double
	// wrap — asserted by errors.Is, which a %w->%v regression would break.
	require.ErrorIs(t, err, domain.ErrEmailSendFailed)
	require.ErrorIs(t, err, errSendDown)

	// The token stays stored: it's single-use and TTL-swept, and a send error can
	// follow an actually-delivered message, so deleting it would risk breaking a
	// link the customer holds.
	assert.Len(t, tokens.logins, 1, "the stored token is left for the TTL to sweep")
}

func TestRequestLink_AdminAllowlist(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()
	sender := &linkSender{to: "", link: ""}

	err := newAuth(users, newFakeTokens(), sender, "Boss@E25.com").RequestLink(t.Context(), "boss@e25.com", "Boss")
	require.NoError(t, err)

	assert.Equal(t, domain.RoleAdmin, users.byEmail["boss@e25.com"].Role)
}

func TestRequestLink_InvalidEmail(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()

	err := newAuth(users, newFakeTokens(), &linkSender{to: "", link: ""}).RequestLink(t.Context(), "nope", "X")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Empty(t, users.byEmail)
}

func TestRequestLink_EmptyName(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()

	err := newAuth(users, newFakeTokens(), &linkSender{to: "", link: ""}).RequestLink(t.Context(), "ama@example.com", "  ")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Empty(t, users.byEmail)
}

func TestVerify_HappyPathAndSingleUse(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()
	tokens := newFakeTokens()
	sender := &linkSender{to: "", link: ""}
	auth := newAuth(users, tokens, sender)

	require.NoError(t, auth.RequestLink(t.Context(), "ama@example.com", "Ama"))

	sessionToken, user, err := auth.Verify(t.Context(), sender.token(t))
	require.NoError(t, err)
	assert.Equal(t, "ama@example.com", user.Email)
	assert.NotEmpty(t, sessionToken)
	assert.Len(t, tokens.sessions, 1)

	_, _, err = auth.Verify(t.Context(), sender.token(t))
	require.ErrorIs(t, err, domain.ErrTokenInvalid, "login tokens are single-use")
}

func TestSessionLifecycle(t *testing.T) {
	t.Parallel()

	users := newFakeUsers()
	tokens := newFakeTokens()
	sender := &linkSender{to: "", link: ""}
	auth := newAuth(users, tokens, sender)

	require.NoError(t, auth.RequestLink(t.Context(), "ama@example.com", "Ama"))

	sessionToken, _, err := auth.Verify(t.Context(), sender.token(t))
	require.NoError(t, err)

	user, err := auth.UserFromSession(t.Context(), sessionToken)
	require.NoError(t, err)
	assert.Equal(t, "ama@example.com", user.Email)

	auth.Logout(t.Context(), sessionToken)

	_, err = auth.UserFromSession(t.Context(), sessionToken)
	require.ErrorIs(t, err, domain.ErrTokenInvalid)
}
