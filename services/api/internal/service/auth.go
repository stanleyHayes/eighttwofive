package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	loginTokenTTL  = 15 * time.Minute
	sessionTTL     = 30 * 24 * time.Hour
	tokenByteCount = 32
)

// Auth implements passwordless sign-in: an emailed one-time link exchanges
// for a long-lived session. Accounts are "light by design" (scope §4.8) —
// upserted on first contact, so order flows can call RequestLink directly.
type Auth struct {
	users  domain.UserRepository
	tokens domain.TokenRepository
	roles  domain.RoleRepository
	email  domain.EmailSender
	logger *slog.Logger
	webURL string
	admins map[string]struct{}
	now    func() time.Time
}

// NewAuth wires the auth service. Emails in adminEmails sign in as admins.
func NewAuth(
	users domain.UserRepository,
	tokens domain.TokenRepository,
	roles domain.RoleRepository,
	email domain.EmailSender,
	logger *slog.Logger,
	webURL string,
	adminEmails []string,
) *Auth {
	admins := make(map[string]struct{}, len(adminEmails))
	for _, adminEmail := range adminEmails {
		admins[strings.ToLower(strings.TrimSpace(adminEmail))] = struct{}{}
	}

	return &Auth{
		users:  users,
		tokens: tokens,
		roles:  roles,
		email:  email,
		logger: logger,
		webURL: strings.TrimRight(webURL, "/"),
		admins: admins,
		now:    time.Now,
	}
}

// RequestLink upserts the user and emails a single-use sign-in link.
func (a *Auth) RequestLink(ctx context.Context, emailAddr, name string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}

	_, err := mail.ParseAddress(emailAddr)
	if err != nil {
		return fmt.Errorf("%w: invalid email address", domain.ErrInvalidInput)
	}

	role := domain.RoleCustomer
	if _, isAdmin := a.admins[emailAddr]; isAdmin {
		role = domain.RoleAdmin
	}

	user := &domain.User{ID: "", Email: emailAddr, Name: name, Role: role, CreatedAt: a.now().UTC()}

	err = a.users.Upsert(ctx, user)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	token, tokenHash, err := newToken()
	if err != nil {
		return err
	}

	// Store the token hash before sending. A stored-but-unsent token is harmless
	// — it is single-use and the TTL index sweeps it within loginTokenTTL — so we
	// deliberately do not delete it if the send fails: a send error can follow a
	// message that was actually delivered (e.g. a response timeout), and deleting
	// then would break a link the customer already holds.
	err = a.tokens.StoreLoginToken(ctx, tokenHash, user.ID, a.now().Add(loginTokenTTL))
	if err != nil {
		return fmt.Errorf("store login token: %w", err)
	}

	err = a.email.SendLoginLink(ctx, user.Email, a.webURL+"/auth/verify?token="+token)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrEmailSendFailed, err)
	}

	return nil
}

// Verify exchanges a one-time login token for a session token and the user.
func (a *Auth) Verify(ctx context.Context, token string) (string, *domain.User, error) {
	userID, err := a.tokens.ConsumeLoginToken(ctx, hashToken(token))
	if err != nil {
		return "", nil, fmt.Errorf("consume login token: %w", err)
	}

	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("load user: %w", err)
	}

	sessionToken, sessionHash, err := newToken()
	if err != nil {
		return "", nil, err
	}

	err = a.tokens.CreateSession(ctx, sessionHash, user.ID, a.now().Add(sessionTTL))
	if err != nil {
		return "", nil, fmt.Errorf("create session: %w", err)
	}

	return sessionToken, user, nil
}

// UserFromSession resolves a session token to its user.
func (a *Auth) UserFromSession(ctx context.Context, sessionToken string) (*domain.User, error) {
	userID, err := a.tokens.GetSession(ctx, hashToken(sessionToken))
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}

	return user, nil
}

// CreateSession creates a new session for an existing user and returns the
// raw session token. It is used by flows that upsert a user before the
// passwordless link step (e.g. checkout).
func (a *Auth) CreateSession(ctx context.Context, userID string) (string, error) {
	sessionToken, sessionHash, err := newToken()
	if err != nil {
		return "", err
	}

	err = a.tokens.CreateSession(ctx, sessionHash, userID, a.now().Add(sessionTTL))
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return sessionToken, nil
}

// Logout revokes the session. Unknown tokens are not an error.
func (a *Auth) Logout(ctx context.Context, sessionToken string) {
	err := a.tokens.DeleteSession(ctx, hashToken(sessionToken))
	if err != nil {
		a.logger.WarnContext(ctx, "delete session", "error", err)
	}
}

// IsSuperAdmin reports whether the email is in the bootstrap ADMIN_EMAILS
// allow-list. Super-admins are always admin and cannot be demoted.
func (a *Auth) IsSuperAdmin(email string) bool {
	_, ok := a.admins[strings.ToLower(strings.TrimSpace(email))]

	return ok
}

// ListUsers returns one page of users for team management.
func (a *Auth) ListUsers(ctx context.Context, page, pageSize int) (domain.Page[domain.User], error) {
	params := domain.NormalizePageParams(page, pageSize)

	total, err := a.users.Count(ctx)
	if err != nil {
		return domain.Page[domain.User]{}, fmt.Errorf("count users: %w", err)
	}

	users, err := a.users.ListPaged(ctx, params)
	if err != nil {
		return domain.Page[domain.User]{}, fmt.Errorf("list users: %w", err)
	}

	return domain.NewPage(users, total, params), nil
}

// SetUserRole changes a user's role. Bootstrap super-admins cannot be demoted,
// and the role must exist in the store (built-in or custom).
func (a *Auth) SetUserRole(ctx context.Context, userID string, role domain.Role) (*domain.User, error) {
	// A user can only hold a role that exists in the store — built-in or custom.
	_, err := a.roles.Get(ctx, string(role))
	if errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("%w: unknown role %q", domain.ErrInvalidInput, role)
	}

	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}

	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}

	if a.IsSuperAdmin(user.Email) && role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: %s is a protected super-admin and stays an admin", domain.ErrInvalidInput, user.Email)
	}

	err = a.users.UpdateRole(ctx, userID, role)
	if err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	user.Role = role

	return user, nil
}

func newToken() (string, string, error) {
	buf := make([]byte, tokenByteCount)

	_, err := rand.Read(buf)
	if err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(buf)

	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}
