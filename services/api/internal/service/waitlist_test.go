package service_test

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type fakeRepo struct {
	subscribers []domain.Subscriber
	createErr   error
}

func (f *fakeRepo) Create(_ context.Context, s *domain.Subscriber) error {
	if f.createErr != nil {
		return f.createErr
	}

	s.ID = "fake-id"
	f.subscribers = append(f.subscribers, *s)

	return nil
}

func (f *fakeRepo) List(_ context.Context, limit int64) ([]domain.Subscriber, error) {
	if int64(len(f.subscribers)) < limit {
		return f.subscribers, nil
	}

	return f.subscribers[:limit], nil
}

func (f *fakeRepo) Count(_ context.Context) (int64, error) {
	return int64(len(f.subscribers)), nil
}

func (f *fakeRepo) ListPaged(_ context.Context, params domain.PageParams) ([]domain.Subscriber, error) {
	return catalogPageSlice(f.subscribers, params), nil
}

type fakeSender struct {
	sent    []string
	sendErr error
}

func (f *fakeSender) SendWelcome(_ context.Context, to, _ string) error {
	if f.sendErr != nil {
		return f.sendErr
	}

	f.sent = append(f.sent, to)

	return nil
}

func (f *fakeSender) SendLoginLink(_ context.Context, _, _ string) error { return nil }

func (f *fakeSender) SendOrderConfirmation(_ context.Context, _, _, _, _ string) error { return nil }

func (f *fakeSender) SendOrderStatusUpdate(_ context.Context, _, _, _, _, _ string) error { return nil }

func newService(repo *fakeRepo, sender *fakeSender) *service.Waitlist {
	return service.NewWaitlist(repo, sender, slog.New(slog.DiscardHandler))
}

func TestJoin_NormalizesAndPersists(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	sender := &fakeSender{}

	sub, err := newService(repo, sender).Join(t.Context(), "  Ada@Example.COM ", " Ada Lovelace ")
	require.NoError(t, err)

	assert.Equal(t, "ada@example.com", sub.Email)
	assert.Equal(t, "Ada Lovelace", sub.Name)
	assert.Equal(t, "fake-id", sub.ID)
	assert.False(t, sub.CreatedAt.IsZero())
	assert.Equal(t, []string{"ada@example.com"}, sender.sent)
}

func TestJoin_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, email, subName string
	}{
		{"empty name", "ada@example.com", "   "},
		{"bad email", "not-an-email", "Ada"},
		{"empty email", "", "Ada"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}
			_, err := newService(repo, &fakeSender{}).Join(t.Context(), tc.email, tc.subName)
			require.ErrorIs(t, err, domain.ErrInvalidInput)
			assert.Empty(t, repo.subscribers)
		})
	}
}

func TestJoin_PropagatesDuplicateError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{createErr: domain.ErrDuplicateEmail}
	sender := &fakeSender{}

	_, err := newService(repo, sender).Join(t.Context(), "ada@example.com", "Ada")
	require.ErrorIs(t, err, domain.ErrDuplicateEmail)
	assert.Empty(t, sender.sent, "no email should be sent for failed signups")
}

func TestJoin_EmailFailureDoesNotFailSignup(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	sender := &fakeSender{sendErr: errors.New("smtp down")}

	sub, err := newService(repo, sender).Join(t.Context(), "ada@example.com", "Ada")
	require.NoError(t, err)
	assert.NotNil(t, sub)
}

func TestList_CapsLimit(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}

	svc := newService(repo, &fakeSender{})

	for range 3 {
		repo.subscribers = append(repo.subscribers, domain.Subscriber{})
	}

	subs, err := svc.List(t.Context(), -5)
	require.NoError(t, err)
	assert.Len(t, subs, 3)
}

func TestListPaged_ReturnsPageWithTotalAndClampsParams(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := newService(repo, &fakeSender{})

	const seeded = 25
	for i := range seeded {
		repo.subscribers = append(repo.subscribers, domain.Subscriber{ID: strconv.Itoa(i)})
	}

	// Invalid params (page 0, pageSize 0) clamp to the defaults: page 1, size 20.
	page, err := svc.ListPaged(t.Context(), 0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(seeded), page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, domain.DefaultPageSize, page.PageSize)
	assert.Len(t, page.Items, domain.DefaultPageSize)

	// Oversized pageSize is capped at MaxPageSize.
	capped, err := svc.ListPaged(t.Context(), 1, 9999)
	require.NoError(t, err)
	assert.Equal(t, domain.MaxPageSize, capped.PageSize)

	// A later page returns the remainder.
	second, err := svc.ListPaged(t.Context(), 2, 20)
	require.NoError(t, err)
	assert.Len(t, second.Items, seeded-20)
}
