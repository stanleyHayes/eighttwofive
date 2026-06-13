// Package service contains application use-cases, orchestrating domain
// entities through the ports defined in the domain package.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	maxNameLength  = 120
	maxEmailLength = 254
	maxListLimit   = 100
)

// Waitlist implements the waitlist use-cases.
type Waitlist struct {
	repo   domain.SubscriberRepository
	email  domain.EmailSender
	logger *slog.Logger
	now    func() time.Time
}

// NewWaitlist wires the waitlist service with its dependencies.
func NewWaitlist(repo domain.SubscriberRepository, email domain.EmailSender, logger *slog.Logger) *Waitlist {
	return &Waitlist{repo: repo, email: email, logger: logger, now: time.Now}
}

// Join validates and persists a new subscriber, then sends a welcome email.
// The email is best-effort: a delivery failure never fails the signup.
func (s *Waitlist) Join(ctx context.Context, email, name string) (*domain.Subscriber, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if name == "" || len(name) > maxNameLength {
		return nil, fmt.Errorf("%w: name must be 1-%d characters", domain.ErrInvalidInput, maxNameLength)
	}

	if len(email) > maxEmailLength {
		return nil, fmt.Errorf("%w: email too long", domain.ErrInvalidInput)
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid email address", domain.ErrInvalidInput)
	}

	sub := &domain.Subscriber{Email: email, Name: name, CreatedAt: s.now().UTC()}

	err = s.repo.Create(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("create subscriber: %w", err)
	}

	err = s.email.SendWelcome(ctx, sub.Email, sub.Name)
	if err != nil {
		s.logger.WarnContext(ctx, "welcome email failed", "email", sub.Email, "error", err)
	}

	return sub, nil
}

// List returns the most recent subscribers, capped at limit.
func (s *Waitlist) List(ctx context.Context, limit int64) ([]domain.Subscriber, error) {
	if limit <= 0 || limit > maxListLimit {
		limit = maxListLimit
	}

	subs, err := s.repo.List(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list subscribers: %w", err)
	}

	return subs, nil
}

// ListPaged returns one page of subscribers (newest first) along with the
// total count, for the admin subscribers table.
func (s *Waitlist) ListPaged(ctx context.Context, page, pageSize int) (domain.Page[domain.Subscriber], error) {
	params := domain.NormalizePageParams(page, pageSize)

	total, err := s.repo.Count(ctx)
	if err != nil {
		return domain.Page[domain.Subscriber]{}, fmt.Errorf("count subscribers: %w", err)
	}

	subs, err := s.repo.ListPaged(ctx, params)
	if err != nil {
		return domain.Page[domain.Subscriber]{}, fmt.Errorf("list subscribers: %w", err)
	}

	return domain.NewPage(subs, total, params), nil
}
