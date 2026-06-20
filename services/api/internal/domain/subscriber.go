// Package domain holds the core business entities and the ports (interfaces)
// that adapters must implement. It has no dependencies on infrastructure.
package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrDuplicateEmail is returned when an email is already on the waitlist.
	ErrDuplicateEmail = errors.New("email already subscribed")
	// ErrInvalidInput is returned when user-provided data fails validation.
	ErrInvalidInput = errors.New("invalid input")
)

// Subscriber is a person on the waitlist.
type Subscriber struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
}

// SubscriberRepository is the persistence port for subscribers.
type SubscriberRepository interface {
	Create(ctx context.Context, s *Subscriber) error
	List(ctx context.Context, limit int64) ([]Subscriber, error)
	// Count returns the total number of subscribers.
	Count(ctx context.Context) (int64, error)
	// ListPaged returns one page of subscribers, newest first.
	ListPaged(ctx context.Context, params PageParams) ([]Subscriber, error)
	// Delete removes a subscriber by id. An unknown or malformed id is ErrNotFound.
	Delete(ctx context.Context, id string) error
}
