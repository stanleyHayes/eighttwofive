package domain

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// Role distinguishes the merchant from customers.
type Role string

// The two roles in the system: the merchant (admin) and customers.
const (
	RoleCustomer Role = "customer"
	RoleAdmin    Role = "admin"
)

// User is a person with an account — created lightly, at the last step of
// completing an order (scope §4.8), or on first sign-in.
type User struct {
	ID        string
	Email     string
	Name      string
	Role      Role
	CreatedAt time.Time
}

// UserRepository is the persistence port for users.
type UserRepository interface {
	// Upsert creates the user if the email is new and backfills ID, Role,
	// Name and CreatedAt from the stored document otherwise. An admin role
	// on the input promotes an existing user; it never demotes.
	Upsert(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
}
