package email

import (
	"context"
	"log/slog"
)

// NoopSender logs instead of sending. Used when no email provider is configured
// (local development, CI) so the rest of the system behaves identically.
type NoopSender struct {
	logger *slog.Logger
}

// NewNoopSender returns a sender that only logs.
func NewNoopSender(logger *slog.Logger) *NoopSender {
	return &NoopSender{logger: logger}
}

// SendWelcome logs the email that would have been sent.
func (s *NoopSender) SendWelcome(ctx context.Context, to, name string) error {
	s.logger.InfoContext(ctx, "email disabled; skipping welcome", "to", to, "name", name)

	return nil
}

// SendLoginLink logs the link so local development can sign in from the log.
func (s *NoopSender) SendLoginLink(ctx context.Context, to, link string) error {
	s.logger.InfoContext(ctx, "email disabled; login link", "to", to, "link", link)

	return nil
}

// SendOrderConfirmation logs the order update so local development can see it.
func (s *NoopSender) SendOrderConfirmation(ctx context.Context, to, name, ref, status string) error {
	s.logger.InfoContext(ctx, "email disabled; order confirmation",
		"to", to, "name", name, "ref", ref, "status", status)

	return nil
}

// SendOrderStatusUpdate logs the status update so local development can see it.
func (s *NoopSender) SendOrderStatusUpdate(
	ctx context.Context, to, name, ref, status, timeframe string,
) error {
	s.logger.InfoContext(ctx, "email disabled; order status update",
		"to", to, "name", name, "ref", ref, "status", status, "timeframe", timeframe)

	return nil
}
