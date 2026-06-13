package domain

import (
	"context"
	"errors"
	"time"
)

// ErrTokenInvalid is returned for unknown, expired, or already-used tokens.
var ErrTokenInvalid = errors.New("token invalid or expired")

// TokenRepository is the persistence port for login tokens and sessions.
// Both are stored hashed; raw tokens never touch the database.
type TokenRepository interface {
	StoreLoginToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	// ConsumeLoginToken atomically marks a login token used and returns its
	// user. A second call with the same hash returns ErrTokenInvalid.
	ConsumeLoginToken(ctx context.Context, tokenHash string) (string, error)

	CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	GetSession(ctx context.Context, tokenHash string) (string, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}
